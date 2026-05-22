package config

import (
	"time"
)

// EvaluationConfig 评测系统配置
type EvaluationConfig struct {
	// 基础配置
	MaxConcurrentTests int           `json:"max_concurrent_tests"` // 最大并发测试数
	DefaultTimeout     time.Duration `json:"default_timeout"`      // 默认超时时间
	EnableMetrics      bool          `json:"enable_metrics"`       // 启用指标收集
	EnableMonitoring   bool          `json:"enable_monitoring"`    // 启用性能监控

	// 性能监控配置
	MonitoringInterval   time.Duration `json:"monitoring_interval"`   // 监控采样间隔
	ResourceAlertThresholds ResourceThresholds `json:"resource_alert_thresholds"` // 资源告警阈值

	// 稳定性测试配置
	StabilityTestDuration time.Duration `json:"stability_test_duration"` // 稳定性测试时长
	StabilityCheckInterval time.Duration `json:"stability_check_interval"` // 稳定性检查间隔

	// 报告配置
	ReportOutputDir string `json:"report_output_dir"` // 报告输出目录
	EnableAutoExport bool   `json:"enable_auto_export"` // 启用自动导出

	// 合规配置
	EnableComplianceCheck bool     `json:"enable_compliance_check"` // 启用合规检查
	ComplianceStandards  []string `json:"compliance_standards"`    // 合规标准列表

	// 调度配置
	SchedulerEnabled bool `json:"scheduler_enabled"` // 启用调度器
	MaxRetryAttempts int  `json:"max_retry_attempts"` // 最大重试次数
}

// ResourceThresholds 资源阈值配置
type ResourceThresholds struct {
	CPUPercent    float64 `json:"cpu_percent"`     // CPU使用率阈值
	MemoryPercent float64 `json:"memory_percent"` // 内存使用率阈值
	GPUPercent    float64 `json:"gpu_percent"`     // GPU使用率阈值
	LatencyMs     float64 `json:"latency_ms"`      // 延迟阈值(毫秒)
}

// MetricConfig 指标配置
type MetricConfig struct {
	Name        string  `json:"name"`        // 指标名称
	DisplayName string  `json:"display_name"` // 显示名称
	Weight      float64 `json:"weight"`      // 权重
	Enabled     bool    `json:"enabled"`     // 是否启用
	Threshold   float64 `json:"threshold"`   // 阈值
	Formula     string  `json:"formula"`     // 计算公式
}

// PipelineConfig 流水线配置
type PipelineConfig struct {
	Name            string        `json:"name"`             // 流水线名称
	Description     string        `json:"description"`      // 描述
	Schedule        string        `json:"schedule"`         // Cron表达式
	TriggerEvents   []string      `json:"trigger_events"`   // 触发事件
	Timeout         time.Duration `json:"timeout"`          // 超时时间
	RetryOnFailure  bool          `json:"retry_on_failure"` // 失败重试
	NotifyOnComplete bool         `json:"notify_on_complete"` // 完成通知
	NotifyOnError   bool          `json:"notify_on_error"`   // 错误通知
}

// DefaultEvaluationConfig 默认评测配置
func DefaultEvaluationConfig() *EvaluationConfig {
	return &EvaluationConfig{
		MaxConcurrentTests:    1000,
		DefaultTimeout:        30 * time.Minute,
		EnableMetrics:         true,
		EnableMonitoring:      true,
		MonitoringInterval:    1 * time.Second,
		StabilityTestDuration: 72 * time.Hour,
		StabilityCheckInterval: 5 * time.Minute,
		ReportOutputDir:       "./reports",
		EnableAutoExport:      true,
		EnableComplianceCheck: true,
		ComplianceStandards:   []string{"GDPR", "CCPA", "HIPAA"},
		SchedulerEnabled:      true,
		MaxRetryAttempts:      3,
		ResourceAlertThresholds: ResourceThresholds{
			CPUPercent:    80.0,
			MemoryPercent: 85.0,
			GPUPercent:    90.0,
			LatencyMs:     1000.0,
		},
	}
}

// DefaultMetricConfigs 默认指标配置
func DefaultMetricConfigs() []*MetricConfig {
	return []*MetricConfig{
		{
			Name:        "accuracy",
			DisplayName: "准确率",
			Weight:      1.0,
			Enabled:     true,
			Threshold:   0.9,
			Formula:     "correct_predictions / total_predictions",
		},
		{
			Name:        "precision",
			DisplayName: "精确率",
			Weight:      0.8,
			Enabled:     true,
			Threshold:   0.85,
			Formula:     "true_positives / (true_positives + false_positives)",
		},
		{
			Name:        "recall",
			DisplayName: "召回率",
			Weight:      0.8,
			Enabled:     true,
			Threshold:   0.85,
			Formula:     "true_positives / (true_positives + false_negatives)",
		},
		{
			Name:        "f1_score",
			DisplayName: "F1值",
			Weight:      1.0,
			Enabled:     true,
			Threshold:   0.87,
			Formula:     "2 * (precision * recall) / (precision + recall)",
		},
		{
			Name:        "latency_p50",
			DisplayName: "P50延迟",
			Weight:      0.5,
			Enabled:     true,
			Threshold:   100,
			Formula:     "percentile(latencies, 50)",
		},
		{
			Name:        "latency_p95",
			DisplayName: "P95延迟",
			Weight:      0.7,
			Enabled:     true,
			Threshold:   500,
			Formula:     "percentile(latencies, 95)",
		},
		{
			Name:        "latency_p99",
			DisplayName: "P99延迟",
			Weight:      0.9,
			Enabled:     true,
			Threshold:   1000,
			Formula:     "percentile(latencies, 99)",
		},
		{
			Name:        "throughput",
			DisplayName: "吞吐量",
			Weight:      0.6,
			Enabled:     true,
			Threshold:   1000,
			Formula:     "requests_per_second",
		},
	}
}
