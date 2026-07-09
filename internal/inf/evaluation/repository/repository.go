package repository

import (
	"awesome/internal/inf/evaluation/errors"
	"awesome/internal/inf/evaluation/logger"
	"awesome/internal/inf/evaluation/models"
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EvaluationRepository 评测数据仓库
type EvaluationRepository struct {
	db     *gorm.DB
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewEvaluationRepository 创建评测数据仓库
func NewEvaluationRepository(db *gorm.DB) (*EvaluationRepository, error) {
	if db == nil {
		return nil, errors.NewError(errors.ErrInvalidParameter, "database connection cannot be nil")
	}

	repo := &EvaluationRepository{
		db:     db,
		logger: logger.GetLogger().WithComponent("evaluation_repository"),
	}

	// 自动迁移
	if err := repo.autoMigrate(); err != nil {
		return nil, errors.Wrap(err, errors.ErrDatabaseConnection, "failed to migrate database")
	}

	return repo, nil
}

// autoMigrate 自动迁移
func (r *EvaluationRepository) autoMigrate() error {
	return r.db.AutoMigrate(
		&models.EvaluationTask{},
		&models.EvaluationResult{},
		&models.StabilityTestResult{},
		&models.ComplianceReport{},
		&models.TestReport{},
		&models.Alert{},
	)
}

// SaveTask 保存任务
func (r *EvaluationRepository) SaveTask(ctx context.Context, task *models.EvaluationTask) error {
	startTime := time.Now()
	defer func() {
		r.logger.LogOperation("save_task", "repository", true, time.Since(startTime),
			zap.String("task_id", task.ID))
	}()

	if err := r.db.WithContext(ctx).Save(task).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("task_id", task.ID)
	}

	return nil
}

// GetTask 获取任务
func (r *EvaluationRepository) GetTask(ctx context.Context, taskID string) (*models.EvaluationTask, error) {
	var task models.EvaluationTask
	if err := r.db.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrTaskNotFoundWithID(taskID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("task_id", taskID)
	}
	return &task, nil
}

// ListTasks 列出任务
func (r *EvaluationRepository) ListTasks(ctx context.Context, status models.TestStatus, limit, offset int) ([]*models.EvaluationTask, error) {
	var tasks []*models.EvaluationTask
	query := r.db.WithContext(ctx).Model(&models.EvaluationTask{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}

	return tasks, nil
}

// DeleteTask 删除任务
func (r *EvaluationRepository) DeleteTask(ctx context.Context, taskID string) error {
	result := r.db.WithContext(ctx).Delete(&models.EvaluationTask{}, "id = ?", taskID)
	if result.Error != nil {
		return errors.ErrDatabaseErrorWithCause("delete", result.Error).
			WithDetail("task_id", taskID)
	}

	if result.RowsAffected == 0 {
		return errors.ErrTaskNotFoundWithID(taskID)
	}

	return nil
}

// SaveResult 保存结果
func (r *EvaluationRepository) SaveResult(ctx context.Context, result *models.EvaluationResult) error {
	startTime := time.Now()
	defer func() {
		r.logger.LogOperation("save_result", "repository", true, time.Since(startTime),
			zap.String("result_id", result.ID))
	}()

	if err := r.db.WithContext(ctx).Save(result).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("result_id", result.ID)
	}

	return nil
}

// GetResult 获取结果
func (r *EvaluationRepository) GetResult(ctx context.Context, resultID string) (*models.EvaluationResult, error) {
	var result models.EvaluationResult
	if err := r.db.WithContext(ctx).First(&result, "id = ?", resultID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrResultNotFound, "result not found").
				WithDetail("result_id", resultID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("result_id", resultID)
	}
	return &result, nil
}

// GetTaskResult 获取任务结果
func (r *EvaluationRepository) GetTaskResult(ctx context.Context, taskID string) (*models.EvaluationResult, error) {
	var result models.EvaluationResult
	if err := r.db.WithContext(ctx).First(&result, "task_id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrResultNotFound, "result not found for task").
				WithDetail("task_id", taskID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("task_id", taskID)
	}
	return &result, nil
}

// SaveStabilityResult 保存稳定性测试结果
func (r *EvaluationRepository) SaveStabilityResult(ctx context.Context, result *models.StabilityTestResult) error {
	if err := r.db.WithContext(ctx).Save(result).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("stability_result_id", result.ID)
	}
	return nil
}

// GetStabilityResult 获取稳定性测试结果
func (r *EvaluationRepository) GetStabilityResult(ctx context.Context, resultID string) (*models.StabilityTestResult, error) {
	var result models.StabilityTestResult
	if err := r.db.WithContext(ctx).First(&result, "id = ?", resultID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrNotFound, "stability result not found").
				WithDetail("result_id", resultID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("result_id", resultID)
	}
	return &result, nil
}

// SaveComplianceReport 保存合规报告
func (r *EvaluationRepository) SaveComplianceReport(ctx context.Context, report *models.ComplianceReport) error {
	if err := r.db.WithContext(ctx).Save(report).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("compliance_report_id", report.ID)
	}
	return nil
}

// GetComplianceReport 获取合规报告
func (r *EvaluationRepository) GetComplianceReport(ctx context.Context, reportID string) (*models.ComplianceReport, error) {
	var report models.ComplianceReport
	if err := r.db.WithContext(ctx).First(&report, "id = ?", reportID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrNotFound, "compliance report not found").
				WithDetail("report_id", reportID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("report_id", reportID)
	}
	return &report, nil
}

// SaveTestReport 保存测试报告
func (r *EvaluationRepository) SaveTestReport(ctx context.Context, report *models.TestReport) error {
	if err := r.db.WithContext(ctx).Save(report).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("test_report_id", report.ID)
	}
	return nil
}

// GetTestReport 获取测试报告
func (r *EvaluationRepository) GetTestReport(ctx context.Context, reportID string) (*models.TestReport, error) {
	var report models.TestReport
	if err := r.db.WithContext(ctx).First(&report, "id = ?", reportID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrNotFound, "test report not found").
				WithDetail("report_id", reportID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("report_id", reportID)
	}
	return &report, nil
}

// SaveAlert 保存告警
func (r *EvaluationRepository) SaveAlert(ctx context.Context, alert *models.Alert) error {
	if err := r.db.WithContext(ctx).Save(alert).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("save", err).
			WithDetail("alert_id", alert.ID)
	}
	return nil
}

// GetAlert 获取告警
func (r *EvaluationRepository) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	var alert models.Alert
	if err := r.db.WithContext(ctx).First(&alert, "id = ?", alertID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewError(errors.ErrNotFound, "alert not found").
				WithDetail("alert_id", alertID)
		}
		return nil, errors.ErrDatabaseErrorWithCause("query", err).
			WithDetail("alert_id", alertID)
	}
	return &alert, nil
}

// ListAlerts 列出告警
func (r *EvaluationRepository) ListAlerts(ctx context.Context, acknowledged *bool, limit int) ([]*models.Alert, error) {
	var alerts []*models.Alert
	query := r.db.WithContext(ctx).Model(&models.Alert{})

	if acknowledged != nil {
		query = query.Where("acknowledged = ?", *acknowledged)
	}

	if err := query.Order("timestamp DESC").Limit(limit).Find(&alerts).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}

	return alerts, nil
}

// UpdateAlertAcknowledged 更新告警确认状态
func (r *EvaluationRepository) UpdateAlertAcknowledged(ctx context.Context, alertID string, acknowledged bool) error {
	result := r.db.WithContext(ctx).Model(&models.Alert{}).
		Where("id = ?", alertID).
		Update("acknowledged", acknowledged)

	if result.Error != nil {
		return errors.ErrDatabaseErrorWithCause("update", result.Error).
			WithDetail("alert_id", alertID)
	}

	if result.RowsAffected == 0 {
		return errors.NewError(errors.ErrNotFound, "alert not found").
			WithDetail("alert_id", alertID)
	}

	return nil
}

// GetStatistics 获取统计信息
func (r *EvaluationRepository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 任务统计
	var taskCount int64
	if err := r.db.WithContext(ctx).Model(&models.EvaluationTask{}).Count(&taskCount).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}
	stats["total_tasks"] = taskCount

	// 按状态统计
	var statusStats []struct {
		Status models.TestStatus
		Count  int64
	}
	if err := r.db.WithContext(ctx).Model(&models.EvaluationTask{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusStats).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}
	stats["tasks_by_status"] = statusStats

	// 结果统计
	var resultCount int64
	if err := r.db.WithContext(ctx).Model(&models.EvaluationResult{}).Count(&resultCount).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}
	stats["total_results"] = resultCount

	// 平均得分
	var avgScore float64
	if err := r.db.WithContext(ctx).Model(&models.EvaluationResult{}).
		Select("AVG(overall_score)").
		Scan(&avgScore).Error; err != nil {
		return nil, errors.ErrDatabaseErrorWithCause("query", err)
	}
	stats["average_score"] = avgScore

	return stats, nil
}

// CleanupOldData 清理旧数据
func (r *EvaluationRepository) CleanupOldData(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// 删除旧任务
	if err := r.db.WithContext(ctx).
		Where("completed_at < ? AND status IN ?", cutoff, []models.TestStatus{
			models.TestStatusCompleted,
			models.TestStatusFailed,
			models.TestStatusCancelled,
		}).
		Delete(&models.EvaluationTask{}).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("delete", err)
	}

	// 删除旧结果
	if err := r.db.WithContext(ctx).
		Where("timestamp < ?", cutoff).
		Delete(&models.EvaluationResult{}).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("delete", err)
	}

	// 删除旧告警
	if err := r.db.WithContext(ctx).
		Where("timestamp < ? AND acknowledged = ?", cutoff, true).
		Delete(&models.Alert{}).Error; err != nil {
		return errors.ErrDatabaseErrorWithCause("delete", err)
	}

	r.logger.Info("Old data cleaned up",
		zap.Int("retention_days", retentionDays),
		zap.Time("cutoff", cutoff))

	return nil
}

// Transaction 执行事务
func (r *EvaluationRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// Ping 测试数据库连接
func (r *EvaluationRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseConnection, "failed to get database connection")
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseConnection, "database ping failed")
	}

	return nil
}

// Close 关闭数据库连接
func (r *EvaluationRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
