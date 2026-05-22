package models

import (
	"time"
)

// TestStatus 测试状态
type TestStatus string

const (
	TestStatusPending    TestStatus = "pending"
	TestStatusRunning    TestStatus = "running"
	TestStatusCompleted  TestStatus = "completed"
	TestStatusFailed     TestStatus = "failed"
	TestStatusCancelled  TestStatus = "cancelled"
	TestStatusRetrying   TestStatus = "retrying"
)

// EvaluationTask 评测任务
type EvaluationTask struct {
	ID          string                 `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ModelID     string                 `json:"model_id"`
	ModelName   string                 `json:"model_name"`
	Status      TestStatus             `json:"status"`
	Priority    int                    `json:"priority"`
	Config      map[string]interface{} `json:"config" gorm:"type:json"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at"`
	Duration    time.Duration          `json:"duration"`
	Progress    float64                `json:"progress"` // 0-100
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
}

// EvaluationResult 评测结果
type EvaluationResult struct {
	ID           string                 `json:"id" gorm:"primaryKey"`
	TaskID       string                 `json:"task_id"`
	ModelID      string                 `json:"model_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Duration     time.Duration          `json:"duration"`
	Metrics      map[string]float64     `json:"metrics" gorm:"type:json"`
	Scores       map[string]float64     `json:"scores" gorm:"type:json"` // 加权得分
	OverallScore float64                `json:"overall_score"`
	Passed       bool                   `json:"passed"`
	Details      map[string]interface{} `json:"details" gorm:"type:json"`
	Warnings     []string               `json:"warnings" gorm:"type:json"`
}

// MetricResult 指标结果
type MetricResult struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"` // 0-100
	Passed      bool    `json:"passed"`
	Threshold   float64 `json:"threshold"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	LatencyMs         float64   `json:"latency_ms"`          // 推理延迟(毫秒)
	CPUPercent        float64   `json:"cpu_percent"`        // CPU使用率
	MemoryPercent     float64   `json:"memory_percent"`     // 内存使用率
	MemoryMB          float64   `json:"memory_mb"`          // 内存使用量(MB)
	GPUPercent        float64   `json:"gpu_percent"`        // GPU使用率
	GPUMemoryMB       float64   `json:"gpu_memory_mb"`      // GPU显存使用量(MB)
	ThroughputRPS     float64   `json:"throughput_rps"`     // 吞吐量(请求/秒)
	ErrorRate         float64   `json:"error_rate"`         // 错误率
	ActiveConnections int       `json:"active_connections"` // 活跃连接数
}

// StabilityTestResult 稳定性测试结果
type StabilityTestResult struct {
	ID                string              `json:"id" gorm:"primaryKey"`
	TaskID            string              `json:"task_id"`
	StartTime         time.Time           `json:"start_time"`
	EndTime           time.Time           `json:"end_time"`
	Duration          time.Duration       `json:"duration"`
	TotalRequests     int64               `json:"total_requests"`
	SuccessCount      int64               `json:"success_count"`
	FailureCount      int64               `json:"failure_count"`
	AvailabilityRate  float64             `json:"availability_rate"` // 可用性(99.99%)
	MeanLatencyMs     float64             `json:"mean_latency_ms"`
	P50LatencyMs      float64             `json:"p50_latency_ms"`
	P95LatencyMs      float64             `json:"p95_latency_ms"`
	P99LatencyMs      float64             `json:"p99_latency_ms"`
	MaxLatencyMs      float64             `json:"max_latency_ms"`
	MinLatencyMs      float64             `json:"min_latency_ms"`
	MemoryLeaks       bool                `json:"memory_leaks"`      // 是否检测到内存泄漏
	PeakMemoryMB      float64             `json:"peak_memory_mb"`
	PeakGPUMemoryMB   float64             `json:"peak_gpu_memory_mb"`
	Errors            []StabilityError    `json:"errors" gorm:"type:json"`
	ResourceSnapshots []ResourceSnapshot  `json:"resource_snapshots" gorm:"type:json"`
}

// StabilityError 稳定性测试错误
type StabilityError struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
}

// ResourceSnapshot 资源快照
type ResourceSnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryMB      float64   `json:"memory_mb"`
	GPUPercent    float64   `json:"gpu_percent"`
	GPUMemoryMB   float64   `json:"gpu_memory_mb"`
	Connections   int       `json:"connections"`
}

// ComplianceReport 合规报告
type ComplianceReport struct {
	ID              string                    `json:"id" gorm:"primaryKey"`
	TaskID          string                    `json:"task_id"`
	Standard        string                    `json:"standard"` // GDPR, CCPA, HIPAA
	Timestamp       time.Time                 `json:"timestamp"`
	OverallStatus   ComplianceStatus          `json:"overall_status"`
	Score           float64                   `json:"score"` // 0-100
	Checks          []ComplianceCheck         `json:"checks" gorm:"type:json"`
	Violations      []ComplianceViolation     `json:"violations" gorm:"type:json"`
	Recommendations []string                  `json:"recommendations" gorm:"type:json"`
	Certified       bool                      `json:"certified"`
	ExpiryDate      *time.Time                `json:"expiry_date"`
}

// ComplianceStatus 合规状态
type ComplianceStatus string

const (
	ComplianceStatusCompliant    ComplianceStatus = "compliant"
	ComplianceStatusNonCompliant ComplianceStatus = "non_compliant"
	ComplianceStatusPartial      ComplianceStatus = "partial"
	ComplianceStatusUnknown      ComplianceStatus = "unknown"
)

// ComplianceCheck 合规检查项
type ComplianceCheck struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      ComplianceStatus `json:"status"`
	Required    bool             `json:"required"`
	Details     string           `json:"details"`
}

// ComplianceViolation 合规违规
type ComplianceViolation struct {
	CheckID     string    `json:"check_id"`
	Severity    string    `json:"severity"` // critical, high, medium, low
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Remediation string    `json:"remediation"`
	DetectedAt  time.Time `json:"detected_at"`
}

// TestReport 测试报告
type TestReport struct {
	ID              string                 `json:"id" gorm:"primaryKey"`
	TaskID          string                 `json:"task_id"`
	Name            string                 `json:"name"`
	ModelName       string                 `json:"model_name"`
	Timestamp       time.Time              `json:"timestamp"`
	Summary         ReportSummary          `json:"summary"`
	Metrics         []MetricResult         `json:"metrics" gorm:"type:json"`
	PerformanceData []PerformanceMetrics   `json:"performance_data" gorm:"type:json"`
	Charts          []ChartConfig          `json:"charts" gorm:"type:json"`
	Recommendations []string               `json:"recommendations" gorm:"type:json"`
	ExportFormats   []string               `json:"export_formats"` // PDF, Excel, JSON
	ExportPaths     map[string]string      `json:"export_paths" gorm:"type:json"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalTests      int     `json:"total_tests"`
	PassedTests     int     `json:"passed_tests"`
	FailedTests     int     `json:"failed_tests"`
	OverallScore    float64 `json:"overall_score"`
	Availability    float64 `json:"availability"` // 99.99%
	MeanLatencyMs   float64 `json:"mean_latency_ms"`
	PeakMemoryMB    float64 `json:"peak_memory_mb"`
	ComplianceScore float64 `json:"compliance_score"`
}

// ChartConfig 图表配置
type ChartConfig struct {
	ID          string                 `json:"id"`
	Type        ChartType              `json:"type"` // line, bar, scatter, pie
	Title       string                 `json:"title"`
	XAxis       string                 `json:"x_axis"`
	YAxis       string                 `json:"y_axis"`
	Series      []ChartSeries          `json:"series"`
	Options     map[string]interface{} `json:"options"`
	Interactive bool                   `json:"interactive"`
}

// ChartType 图表类型
type ChartType string

const (
	ChartTypeLine    ChartType = "line"
	ChartTypeBar     ChartType = "bar"
	ChartTypeScatter ChartType = "scatter"
	ChartTypePie     ChartType = "pie"
	ChartTypeArea    ChartType = "area"
	ChartTypeHeatmap ChartType = "heatmap"
)

// ChartSeries 图表数据系列
type ChartSeries struct {
	Name   string      `json:"name"`
	Data   []DataPoint `json:"data"`
	Color  string      `json:"color"`
	Format string      `json:"format"` // line style, bar style
}

// DataPoint 数据点
type DataPoint struct {
	X     interface{} `json:"x"`
	Y     interface{} `json:"y"`
	Label string      `json:"label,omitempty"`
}

// PipelineRun 流水线运行记录
type PipelineRun struct {
	ID           string                 `json:"id" gorm:"primaryKey"`
	PipelineID   string                 `json:"pipeline_id"`
	Name         string                 `json:"name"`
	TriggerType  string                 `json:"trigger_type"` // scheduled, manual, event
	TriggeredBy  string                 `json:"triggered_by"`
	Status       TestStatus             `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	TaskIDs      []string               `json:"task_ids" gorm:"type:json"`
	Progress     float64                `json:"progress"`
	Error        string                 `json:"error,omitempty"`
	Artifacts    map[string]string      `json:"artifacts" gorm:"type:json"`
	Metadata     map[string]interface{} `json:"metadata" gorm:"type:json"`
}

// Alert 告警
type Alert struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Type        string    `json:"type"` // performance, error, compliance
	Severity    string    `json:"severity"` // critical, warning, info
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	TaskID      string    `json:"task_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Acknowledged bool     `json:"acknowledged"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Details     map[string]interface{} `json:"details" gorm:"type:json"`
}
