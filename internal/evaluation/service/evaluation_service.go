package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"awesome/internal/evaluation/compliance"
	"awesome/internal/evaluation/config"
	"awesome/internal/evaluation/metrics"
	"awesome/internal/evaluation/models"
	"awesome/internal/evaluation/performance"
	"awesome/internal/evaluation/report"
)

// EvaluationService 评测服务
type EvaluationService struct {
	config           *config.EvaluationConfig
	metricsCalc      *metrics.MetricsCalculator
	perfMonitor      *performance.PerformanceMonitor
	stabilityTester  *performance.StabilityTester
	reportGenerator  *report.ReportGenerator
	complianceChecker *compliance.ComplianceChecker
	tasks            map[string]*models.EvaluationTask
	results          map[string]*models.EvaluationResult
	mu               sync.RWMutex
}

// NewEvaluationService 创建评测服务
func NewEvaluationService(cfg *config.EvaluationConfig) *EvaluationService {
	return &EvaluationService{
		config:            cfg,
		metricsCalc:       metrics.NewMetricsCalculator(config.DefaultMetricConfigs()),
		perfMonitor:       performance.NewPerformanceMonitor(cfg),
		stabilityTester:   performance.NewStabilityTester(cfg),
		reportGenerator:   report.NewReportGenerator(cfg),
		complianceChecker: compliance.NewComplianceChecker(),
		tasks:             make(map[string]*models.EvaluationTask),
		results:           make(map[string]*models.EvaluationResult),
	}
}

// CreateTask 创建评测任务
func (s *EvaluationService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*models.EvaluationTask, error) {
	task := &models.EvaluationTask{
		ID:          generateTaskIDV1(),
		Name:        req.Name,
		Description: req.Description,
		ModelID:     req.ModelID,
		ModelName:   req.ModelName,
		Status:      models.TestStatusPending,
		Priority:    req.Priority,
		Config:      req.Config,
		CreatedAt:   time.Now(),
		Progress:    0,
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return task, nil
}

// StartTask 启动评测任务
func (s *EvaluationService) StartTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	task, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != models.TestStatusPending {
		s.mu.Unlock()
		return fmt.Errorf("task %s is not in pending status", taskID)
	}

	task.Status = models.TestStatusRunning
	now := time.Now()
	task.StartedAt = &now
	s.mu.Unlock()

	// 启动性能监控
	if s.config.EnableMonitoring {
		if err := s.perfMonitor.Start(ctx); err != nil {
			return err
		}
	}

	// 执行评测
	go s.executeTask(ctx, task)

	return nil
}

// executeTask 执行评测任务
func (s *EvaluationService) executeTask(ctx context.Context, task *models.EvaluationTask) {
	result := &models.EvaluationResult{
		ID:        generateResultIDV1(),
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
	}()

	// 模拟评测过程
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			task.Status = models.TestStatusCancelled
			return
		default:
			// 执行评测步骤
			time.Sleep(100 * time.Millisecond)
			task.Progress = float64(i+1) * 10
		}
	}

	// 计算指标
	result.Metrics["accuracy"] = 0.95
	result.Metrics["precision"] = 0.93
	result.Metrics["recall"] = 0.92
	result.Metrics["f1_score"] = 0.925
	result.Metrics["latency_p50"] = 45.5
	result.Metrics["latency_p95"] = 120.3
	result.Metrics["latency_p99"] = 250.8
	result.Metrics["throughput"] = 1500.0

	// 计算加权得分
	overallScore, scores := s.metricsCalc.CalculateWeightedScore(result.Metrics)
	result.OverallScore = overallScore
	result.Scores = scores
	result.Passed = overallScore >= 80.0
}

// GetTask 获取任务
func (s *EvaluationService) GetTask(taskID string) (*models.EvaluationTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return task, nil
}

// GetResult 获取结果
func (s *EvaluationService) GetResult(resultID string) (*models.EvaluationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, exists := s.results[resultID]
	if !exists {
		return nil, fmt.Errorf("result %s not found", resultID)
	}
	return result, nil
}

// GetTaskResult 获取任务结果
func (s *EvaluationService) GetTaskResult(taskID string) (*models.EvaluationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, result := range s.results {
		if result.TaskID == taskID {
			return result, nil
		}
	}
	return nil, fmt.Errorf("result for task %s not found", taskID)
}

// RunStabilityTest 运行稳定性测试
func (s *EvaluationService) RunStabilityTest(ctx context.Context, taskID string, testFunc func() error) (*models.StabilityTestResult, error) {
	return s.stabilityTester.RunStabilityTest(ctx, testFunc)
}

// CheckCompliance 检查合规性
func (s *EvaluationService) CheckCompliance(taskID string, standard string, data interface{}) (*models.ComplianceReport, error) {
	return s.complianceChecker.CheckCompliance(taskID, standard, data)
}

// GenerateReport 生成报告
func (s *EvaluationService) GenerateReport(taskID string) (*models.TestReport, error) {
	result, err := s.GetTaskResult(taskID)
	if err != nil {
		return nil, err
	}

	perfData := s.perfMonitor.GetMetrics(1 * time.Hour)
	return s.reportGenerator.GenerateReport(taskID, result, perfData)
}

// ExportReport 导出报告
func (s *EvaluationService) ExportReport(report *models.TestReport, format string) (string, error) {
	return s.reportGenerator.ExportReport(report, format)
}

// GetPerformanceMetrics 获取性能指标
func (s *EvaluationService) GetPerformanceMetrics(duration time.Duration) []models.PerformanceMetrics {
	return s.perfMonitor.GetMetrics(duration)
}

// ConfigureMetric 配置指标
func (s *EvaluationService) ConfigureMetric(cfg *config.MetricConfig) error {
	s.metricsCalc.AddCustomMetric(cfg)
	return nil
}

// UpdateMetricWeight 更新指标权重
func (s *EvaluationService) UpdateMetricWeight(name string, weight float64) error {
	return s.metricsCalc.UpdateMetricWeight(name, weight)
}

// ListTasks 列出任务
func (s *EvaluationService) ListTasks(status models.TestStatus, limit int) []*models.EvaluationTask {
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
	return tasks
}

// CancelTask 取消任务
func (s *EvaluationService) CancelTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status == models.TestStatusRunning {
		task.Status = models.TestStatusCancelled
		now := time.Now()
		task.CompletedAt = &now
	}

	return nil
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ModelID     string                 `json:"model_id"`
	ModelName   string                 `json:"model_name"`
	Priority    int                    `json:"priority"`
	Config      map[string]interface{} `json:"config"`
}

func generateTaskIDV1() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

func generateResultIDV1() string {
	return fmt.Sprintf("result_%d", time.Now().UnixNano())
}
