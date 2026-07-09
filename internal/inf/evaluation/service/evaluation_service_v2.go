package service

import (
	"awesome/internal/inf/evaluation/compliance"
	"awesome/internal/inf/evaluation/config"
	"awesome/internal/inf/evaluation/errors"
	"awesome/internal/inf/evaluation/logger"
	"awesome/internal/inf/evaluation/metrics"
	"awesome/internal/inf/evaluation/models"
	"awesome/internal/inf/evaluation/performance"
	"awesome/internal/inf/evaluation/report"
	"awesome/internal/inf/evaluation/validator"
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EvaluationServiceV2 企业级评测服务
type EvaluationServiceV2 struct {
	config            *config.EvaluationConfig
	metricsCalc       *metrics.MetricsCalculator
	perfMonitor       *performance.PerformanceMonitor
	stabilityTester   *performance.StabilityTester
	reportGenerator   *report.ReportGenerator
	complianceChecker *compliance.ComplianceChecker

	tasks   map[string]*models.EvaluationTask
	results map[string]*models.EvaluationResult

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool

	logger          *logger.Logger
	errorHandler    *ErrorHandler
	securityManager *SecurityManager
}

// NewEvaluationServiceV2 创建企业级评测服务
func NewEvaluationServiceV2(cfg *config.EvaluationConfig) (*EvaluationServiceV2, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, errors.Wrap(err, errors.ErrConfigValidate, "invalid configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 初始化日志
	log := logger.GetLogger().WithComponent("evaluation_service")

	// 创建服务实例
	service := &EvaluationServiceV2{
		config:            cfg,
		metricsCalc:       metrics.NewMetricsCalculator(config.DefaultMetricConfigs()),
		perfMonitor:       performance.NewPerformanceMonitor(cfg),
		stabilityTester:   performance.NewStabilityTester(cfg),
		reportGenerator:   report.NewReportGenerator(cfg),
		complianceChecker: compliance.NewComplianceChecker(),
		tasks:             make(map[string]*models.EvaluationTask),
		results:           make(map[string]*models.EvaluationResult),
		ctx:               ctx,
		cancel:            cancel,
		logger:            log,
		errorHandler:      NewErrorHandler(log),
		securityManager:   NewSecurityManager(),
	}

	// 启动后台任务
	go service.cleanupRoutine()
	go service.monitorRoutine()

	log.Info("Evaluation service initialized successfully",
		zap.Int("max_concurrent_tests", cfg.MaxConcurrentTests),
		zap.Duration("default_timeout", cfg.DefaultTimeout))

	return service, nil
}

// CreateTask 创建评测任务
func (s *EvaluationServiceV2) CreateTask(ctx context.Context, req *CreateTaskRequest) (*models.EvaluationTask, error) {
	startTime := time.Now()
	defer func() {
		s.logger.LogOperation("create_task", "evaluation_service", true, time.Since(startTime),
			zap.String("task_name", req.Name))
	}()

	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "create_task"); err != nil {
		return nil, err
	}

	// 参数验证
	if err := s.validateCreateTaskRequest(req); err != nil {
		return nil, err
	}

	// 检查并发限制
	if len(s.tasks) >= s.config.MaxConcurrentTests {
		return nil, errors.NewError(errors.ErrResourceExhausted, "maximum concurrent tasks limit reached").
			WithRetryable(true).
			WithSeverity("high")
	}

	// 创建任务
	task := &models.EvaluationTask{
		ID:          generateTaskIDV2(),
		Name:        validator.SanitizeString(req.Name),
		Description: validator.SanitizeString(req.Description),
		ModelID:     req.ModelID,
		ModelName:   req.ModelName,
		Status:      models.TestStatusPending,
		Priority:    req.Priority,
		Config:      validator.SanitizeMap(req.Config),
		CreatedAt:   time.Now(),
		Progress:    0,
		RetryCount:  0,
	}

	// 保存任务
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	s.logger.Info("Task created successfully",
		zap.String("task_id", task.ID),
		zap.String("model_id", task.ModelID),
		zap.Int("priority", task.Priority))

	return task, nil
}

// StartTask 启动评测任务
func (s *EvaluationServiceV2) StartTask(ctx context.Context, taskID string) error {
	startTime := time.Now()
	defer func() {
		s.logger.LogOperation("start_task", "evaluation_service", true, time.Since(startTime),
			zap.String("task_id", taskID))
	}()

	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "start_task"); err != nil {
		return err
	}

	// 验证任务ID
	if err := validator.ValidateTaskID(taskID); err != nil {
		return err
	}

	// 获取任务
	s.mu.Lock()
	task, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		return errors.ErrTaskNotFoundWithID(taskID)
	}

	// 检查任务状态
	if task.Status != models.TestStatusPending {
		s.mu.Unlock()
		return errors.NewError(errors.ErrTaskNotPending,
			fmt.Sprintf("task status is %s, expected pending", task.Status)).
			WithDetail("current_status", task.Status).
			WithDetail("task_id", taskID)
	}

	// 更新任务状态
	task.Status = models.TestStatusRunning
	now := time.Now()
	task.StartedAt = &now
	s.mu.Unlock()

	// 启动性能监控
	if s.config.EnableMonitoring {
		if err := s.perfMonitor.Start(ctx); err != nil {
			s.errorHandler.HandleError(err, "failed to start performance monitor",
				zap.String("task_id", taskID))
			// 继续执行，监控失败不应阻止评测
		}
	}

	// 执行评测任务
	go s.executeTaskWithRecovery(ctx, task)

	s.logger.Info("Task started successfully",
		zap.String("task_id", taskID),
		zap.String("model_id", task.ModelID))

	return nil
}

// executeTaskWithRecovery 带恢复机制的评测执行
func (s *EvaluationServiceV2) executeTaskWithRecovery(ctx context.Context, task *models.EvaluationTask) {
	defer func() {
		if r := recover(); r != nil {
			s.errorHandler.HandlePanic(r, "task execution panic",
				zap.String("task_id", task.ID))

			s.mu.Lock()
			task.Status = models.TestStatusFailed
			task.Error = fmt.Sprintf("panic: %v", r)
			now := time.Now()
			task.CompletedAt = &now
			s.mu.Unlock()
		}
	}()

	s.executeTask(ctx, task)
}

// executeTask 执行评测任务
func (s *EvaluationServiceV2) executeTask(ctx context.Context, task *models.EvaluationTask) {
	result := &models.EvaluationResult{
		ID:        generateResultIDV2(),
		TaskID:    task.ID,
		ModelID:   task.ModelID,
		Timestamp: time.Now(),
		Metrics:   make(map[string]float64),
		Scores:    make(map[string]float64),
		Details:   make(map[string]interface{}),
		Warnings:  make([]string, 0),
	}

	defer func() {
		s.mu.Lock()
		task.Status = models.TestStatusCompleted
		now := time.Now()
		task.CompletedAt = &now
		task.Duration = now.Sub(task.CreatedAt)
		task.Progress = 100
		s.results[result.ID] = result
		s.mu.Unlock()

		s.logger.Info("Task completed",
			zap.String("task_id", task.ID),
			zap.Duration("duration", task.Duration),
			zap.Float64("overall_score", result.OverallScore))
	}()

	// 执行评测步骤
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			task.Status = models.TestStatusCancelled
			task.Error = "task cancelled by user"
			s.mu.Unlock()
			s.logger.Warn("Task cancelled",
				zap.String("task_id", task.ID))
			return
		case <-s.ctx.Done():
			s.mu.Lock()
			task.Status = models.TestStatusCancelled
			task.Error = "service shutting down"
			s.mu.Unlock()
			return
		default:
			// 执行评测步骤
			if err := s.executeEvaluationStep(ctx, task, i); err != nil {
				s.errorHandler.HandleError(err, "evaluation step failed",
					zap.String("task_id", task.ID),
					zap.Int("step", i))

				// 检查是否需要重试
				if task.RetryCount < s.config.MaxRetryAttempts && errors.IsRetryable(err) {
					task.RetryCount++
					task.Status = models.TestStatusRetrying
					s.logger.Info("Retrying task",
						zap.String("task_id", task.ID),
						zap.Int("retry_count", task.RetryCount))
					i-- // 重试当前步骤
					continue
				}

				// 标记为失败
				s.mu.Lock()
				task.Status = models.TestStatusFailed
				task.Error = err.Error()
				now := time.Now()
				task.CompletedAt = &now
				s.mu.Unlock()
				return
			}

			task.Progress = float64(i+1) * 10
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 计算指标
	s.calculateMetrics(result)

	// 计算加权得分
	overallScore, scores := s.metricsCalc.CalculateWeightedScore(result.Metrics)
	result.OverallScore = overallScore
	result.Scores = scores
	result.Passed = overallScore >= 80.0

	// 添加警告
	if result.OverallScore < 90 {
		result.Warnings = append(result.Warnings, "Overall score is below optimal threshold")
	}
}

// executeEvaluationStep 执行评测步骤
func (s *EvaluationServiceV2) executeEvaluationStep(ctx context.Context, task *models.EvaluationTask, step int) error {
	// 模拟评测步骤执行
	// 实际实现中，这里应该调用具体的评测逻辑

	// 检查超时
	if task.StartedAt != nil && time.Since(*task.StartedAt) > s.config.DefaultTimeout {
		return errors.ErrTimeoutWithDetails("evaluation_step", s.config.DefaultTimeout)
	}

	// 检查资源使用
	if s.config.EnableMonitoring {
		if latestMetrics := s.perfMonitor.GetLatestMetrics(); latestMetrics != nil {
			if latestMetrics.CPUPercent > s.config.ResourceAlertThresholds.CPUPercent {
				return errors.ErrResourceExhaustedWithDetails("CPU",
					latestMetrics.CPUPercent, s.config.ResourceAlertThresholds.CPUPercent)
			}
			if latestMetrics.MemoryPercent > s.config.ResourceAlertThresholds.MemoryPercent {
				return errors.ErrResourceExhaustedWithDetails("Memory",
					latestMetrics.MemoryPercent, s.config.ResourceAlertThresholds.MemoryPercent)
			}
		}
	}

	return nil
}

// calculateMetrics 计算指标
func (s *EvaluationServiceV2) calculateMetrics(result *models.EvaluationResult) {
	// 模拟指标计算
	result.Metrics["accuracy"] = 0.95
	result.Metrics["precision"] = 0.93
	result.Metrics["recall"] = 0.92
	result.Metrics["f1_score"] = 0.925
	result.Metrics["latency_p50"] = 45.5
	result.Metrics["latency_p95"] = 120.3
	result.Metrics["latency_p99"] = 250.8
	result.Metrics["throughput"] = 1500.0
}

// GetTask 获取任务
func (s *EvaluationServiceV2) GetTask(ctx context.Context, taskID string) (*models.EvaluationTask, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "get_task"); err != nil {
		return nil, err
	}

	// 验证任务ID
	if err := validator.ValidateTaskID(taskID); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, errors.ErrTaskNotFoundWithID(taskID)
	}

	return task, nil
}

// GetResult 获取结果
func (s *EvaluationServiceV2) GetResult(ctx context.Context, resultID string) (*models.EvaluationResult, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "get_result"); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.results[resultID]
	if !exists {
		return nil, errors.NewError(errors.ErrResultNotFound,
			fmt.Sprintf("result '%s' not found", resultID)).
			WithDetail("result_id", resultID)
	}

	return result, nil
}

// GetTaskResult 获取任务结果
func (s *EvaluationServiceV2) GetTaskResult(ctx context.Context, taskID string) (*models.EvaluationResult, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "get_task_result"); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, result := range s.results {
		if result.TaskID == taskID {
			return result, nil
		}
	}

	return nil, errors.NewError(errors.ErrResultNotFound,
		fmt.Sprintf("result for task '%s' not found", taskID)).
		WithDetail("task_id", taskID)
}

// CancelTask 取消任务
func (s *EvaluationServiceV2) CancelTask(ctx context.Context, taskID string) error {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "cancel_task"); err != nil {
		return err
	}

	// 验证任务ID
	if err := validator.ValidateTaskID(taskID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return errors.ErrTaskNotFoundWithID(taskID)
	}

	if task.Status == models.TestStatusRunning {
		task.Status = models.TestStatusCancelled
		now := time.Now()
		task.CompletedAt = &now
		task.Error = "cancelled by user"
	}

	s.logger.Info("Task cancelled",
		zap.String("task_id", taskID))

	return nil
}

// RunStabilityTest 运行稳定性测试
func (s *EvaluationServiceV2) RunStabilityTest(ctx context.Context, taskID string, testFunc func() error) (*models.StabilityTestResult, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "run_stability_test"); err != nil {
		return nil, err
	}

	result, err := s.stabilityTester.RunStabilityTest(ctx, testFunc)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrTaskFailed, "stability test failed")
	}

	return result, nil
}

// CheckCompliance 检查合规性
func (s *EvaluationServiceV2) CheckCompliance(ctx context.Context, taskID string, standard string, data interface{}) (*models.ComplianceReport, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "check_compliance"); err != nil {
		return nil, err
	}

	// 验证合规标准
	if err := validator.ValidateComplianceStandard(standard); err != nil {
		return nil, err
	}

	report, err := s.complianceChecker.CheckCompliance(taskID, standard, data)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrComplianceCheck, "compliance check failed")
	}

	return report, nil
}

// GenerateReport 生成报告
func (s *EvaluationServiceV2) GenerateReport(ctx context.Context, taskID string) (*models.TestReport, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "generate_report"); err != nil {
		return nil, err
	}

	result, err := s.GetTaskResult(ctx, taskID)
	if err != nil {
		return nil, err
	}

	perfData := s.perfMonitor.GetMetrics(1 * time.Hour)
	report, err := s.reportGenerator.GenerateReport(taskID, result, perfData)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrReportGeneration, "failed to generate report")
	}

	return report, nil
}

// ExportReport 导出报告
func (s *EvaluationServiceV2) ExportReport(ctx context.Context, report *models.TestReport, format string) (string, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "export_report"); err != nil {
		return "", err
	}

	// 验证导出格式
	validFormats := []string{"json", "pdf", "excel"}
	v := validator.NewValidator()
	v.InStringList("format", format, validFormats)
	if v.HasErrors() {
		return "", v.GetError()
	}

	path, err := s.reportGenerator.ExportReport(report, format)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrReportGeneration, "failed to export report")
	}

	return path, nil
}

// ListTasks 列出任务
func (s *EvaluationServiceV2) ListTasks(ctx context.Context, status models.TestStatus, limit int) ([]*models.EvaluationTask, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "list_tasks"); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*models.EvaluationTask, 0)
	for _, task := range s.tasks {
		if status == "" || task.Status == status {
			tasks = append(tasks, task)
			if limit > 0 && len(tasks) >= limit {
				break
			}
		}
	}

	return tasks, nil
}

// GetPerformanceMetrics 获取性能指标
func (s *EvaluationServiceV2) GetPerformanceMetrics(ctx context.Context, duration time.Duration) ([]models.PerformanceMetrics, error) {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "get_performance_metrics"); err != nil {
		return nil, err
	}

	return s.perfMonitor.GetMetrics(duration), nil
}

// ConfigureMetric 配置指标
func (s *EvaluationServiceV2) ConfigureMetric(ctx context.Context, cfg *config.MetricConfig) error {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "configure_metric"); err != nil {
		return err
	}

	// 验证指标配置
	if err := s.validateMetricConfig(cfg); err != nil {
		return err
	}

	s.metricsCalc.AddCustomMetric(cfg)
	return nil
}

// UpdateMetricWeight 更新指标权重
func (s *EvaluationServiceV2) UpdateMetricWeight(ctx context.Context, name string, weight float64) error {
	// 安全验证
	if err := s.securityManager.ValidateRequest(ctx, "update_metric_weight"); err != nil {
		return err
	}

	// 验证权重
	if err := validator.ValidateWeight(weight); err != nil {
		return err
	}

	return s.metricsCalc.UpdateMetricWeight(name, weight)
}

// Shutdown 关闭服务
func (s *EvaluationServiceV2) Shutdown() error {
	s.logger.Info("Shutting down evaluation service")

	s.cancel()

	// 停止性能监控
	s.perfMonitor.Stop()

	// 等待所有任务完成
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			s.logger.Warn("Shutdown timeout, forcing exit")
			return nil
		case <-ticker.C:
			s.mu.RLock()
			runningCount := 0
			for _, task := range s.tasks {
				if task.Status == models.TestStatusRunning {
					runningCount++
				}
			}
			s.mu.RUnlock()

			if runningCount == 0 {
				s.logger.Info("All tasks completed, shutdown successful")
				return nil
			}

			s.logger.Info("Waiting for tasks to complete",
				zap.Int("running_count", runningCount))
		}
	}
}

// cleanupRoutine 清理例程
func (s *EvaluationServiceV2) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOldTasks()
		}
	}
}

// cleanupOldTasks 清理旧任务
func (s *EvaluationServiceV2) cleanupOldTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, task := range s.tasks {
		if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			delete(s.tasks, id)
			delete(s.results, id)
		}
	}
}

// monitorRoutine 监控例程
func (s *EvaluationServiceV2) monitorRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.logStatistics()
		}
	}
}

// logStatistics 记录统计信息
func (s *EvaluationServiceV2) logStatistics() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statusCount := make(map[models.TestStatus]int)
	for _, task := range s.tasks {
		statusCount[task.Status]++
	}

	s.logger.Info("Service statistics",
		zap.Int("total_tasks", len(s.tasks)),
		zap.Int("total_results", len(s.results)),
		zap.Any("status_distribution", statusCount))
}

// validateCreateTaskRequest 验证创建任务请求
func (s *EvaluationServiceV2) validateCreateTaskRequest(req *CreateTaskRequest) error {
	v := validator.NewValidator()
	v.Required("name", req.Name).
		MinLength("name", req.Name, 1).
		MaxLength("name", req.Name, 200).
		NoSpecialChars("name", req.Name)

	v.Required("model_id", req.ModelID)
	if err := validator.ValidateModelID(req.ModelID); err != nil {
		v.AddError("model_id", "invalid model ID format", req.ModelID)
	}

	v.Range("priority", float64(req.Priority), 0, 100)

	if req.Config != nil {
		if err := validator.ValidateConfig(req.Config); err != nil {
			v.AddError("config", "invalid configuration", req.Config)
		}
	}

	return v.GetError()
}

// validateMetricConfig 验证指标配置
func (s *EvaluationServiceV2) validateMetricConfig(cfg *config.MetricConfig) error {
	v := validator.NewValidator()
	v.Required("name", cfg.Name).
		MinLength("name", cfg.Name, 1).
		MaxLength("name", cfg.Name, 50)

	v.Range("weight", cfg.Weight, 0, 10)
	v.Range("threshold", cfg.Threshold, 0, 1)

	return v.GetError()
}

// validateConfig 验证配置
func validateConfig(cfg *config.EvaluationConfig) error {
	if cfg == nil {
		return errors.NewError(errors.ErrInvalidParameter, "config cannot be nil")
	}

	v := validator.NewValidator()
	v.Positive("max_concurrent_tests", float64(cfg.MaxConcurrentTests))
	v.Duration("default_timeout", cfg.DefaultTimeout, 1*time.Second, 24*time.Hour)
	v.Duration("monitoring_interval", cfg.MonitoringInterval, 100*time.Millisecond, 1*time.Hour)

	return v.GetError()
}

func generateTaskIDV2() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

func generateResultIDV2() string {
	return fmt.Sprintf("result_%d", time.Now().UnixNano())
}
