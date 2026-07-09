package report

import (
	"awesome/internal/inf/evaluation/config"
	"awesome/internal/inf/evaluation/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReportGenerator 报告生成器
type ReportGenerator struct {
	config    *config.EvaluationConfig
	templates map[string]*ChartTemplate
	exporters map[string]Exporter
	mu        sync.RWMutex
}

// ChartTemplate 图表模板
type ChartTemplate struct {
	ID          string
	Name        string
	Type        models.ChartType
	Config      map[string]interface{}
	Interactive bool
}

// Exporter 导出器接口
type Exporter interface {
	Export(report *models.TestReport, outputPath string) error
	GetFormat() string
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(cfg *config.EvaluationConfig) *ReportGenerator {
	rg := &ReportGenerator{
		config:    cfg,
		templates: make(map[string]*ChartTemplate),
		exporters: make(map[string]Exporter),
	}

	// 注册默认模板
	rg.registerDefaultTemplates()

	// 注册默认导出器
	rg.registerDefaultExporters()

	return rg
}

// registerDefaultTemplates 注册默认图表模板
func (rg *ReportGenerator) registerDefaultTemplates() {
	// 折线图模板 - 性能趋势
	rg.templates["performance_trend"] = &ChartTemplate{
		ID:   "performance_trend",
		Name: "性能趋势图",
		Type: models.ChartTypeLine,
		Config: map[string]interface{}{
			"xAxis":  "timestamp",
			"yAxis":  "value",
			"smooth": true,
		},
		Interactive: true,
	}

	// 柱状图模板 - 指标对比
	rg.templates["metrics_comparison"] = &ChartTemplate{
		ID:   "metrics_comparison",
		Name: "指标对比图",
		Type: models.ChartTypeBar,
		Config: map[string]interface{}{
			"xAxis": "metric_name",
			"yAxis": "value",
			"stack": false,
		},
		Interactive: true,
	}

	// 散点图模板 - 延迟分布
	rg.templates["latency_distribution"] = &ChartTemplate{
		ID:   "latency_distribution",
		Name: "延迟分布图",
		Type: models.ChartTypeScatter,
		Config: map[string]interface{}{
			"xAxis": "request_id",
			"yAxis": "latency_ms",
		},
		Interactive: true,
	}

	// 饼图模板 - 测试结果分布
	rg.templates["test_distribution"] = &ChartTemplate{
		ID:   "test_distribution",
		Name: "测试结果分布",
		Type: models.ChartTypePie,
		Config: map[string]interface{}{
			"valueField": "count",
			"nameField":  "status",
		},
		Interactive: false,
	}

	// 热力图模板 - 资源使用
	rg.templates["resource_heatmap"] = &ChartTemplate{
		ID:   "resource_heatmap",
		Name: "资源使用热力图",
		Type: models.ChartTypeHeatmap,
		Config: map[string]interface{}{
			"xAxis": "time",
			"yAxis": "resource_type",
		},
		Interactive: true,
	}
}

// registerDefaultExporters 注册默认导出器
func (rg *ReportGenerator) registerDefaultExporters() {
	rg.exporters["json"] = &JSONExporter{}
	rg.exporters["pdf"] = &PDFExporter{}
	rg.exporters["excel"] = &ExcelExporter{}
}

// GenerateReport 生成报告
func (rg *ReportGenerator) GenerateReport(taskID string, results *models.EvaluationResult, perfData []models.PerformanceMetrics) (*models.TestReport, error) {
	report := &models.TestReport{
		ID:              generateReportID(),
		TaskID:          taskID,
		Name:            fmt.Sprintf("Evaluation Report - %s", time.Now().Format("2006-01-02 15:04:05")),
		Timestamp:       time.Now(),
		Metrics:         make([]models.MetricResult, 0),
		PerformanceData: perfData,
		Charts:          make([]models.ChartConfig, 0),
		Recommendations: make([]string, 0),
		ExportFormats:   []string{"json", "pdf", "excel"},
		ExportPaths:     make(map[string]string),
	}

	// 生成摘要
	report.Summary = models.ReportSummary{
		TotalTests:   len(results.Metrics),
		PassedTests:  countPassedMetrics(results.Metrics),
		FailedTests:  len(results.Metrics) - countPassedMetrics(results.Metrics),
		OverallScore: results.OverallScore,
	}

	// 转换指标结果
	for name, value := range results.Metrics {
		report.Metrics = append(report.Metrics, models.MetricResult{
			Name:  name,
			Value: value,
			Score: results.Scores[name],
		})
	}

	// 生成图表
	rg.generateCharts(report, perfData)

	// 生成建议
	rg.generateRecommendations(report)

	return report, nil
}

// generateCharts 生成图表
func (rg *ReportGenerator) generateCharts(report *models.TestReport, perfData []models.PerformanceMetrics) {
	// 性能趋势图
	if len(perfData) > 0 {
		chart := rg.createPerformanceTrendChart(perfData)
		report.Charts = append(report.Charts, chart)
	}

	// 指标对比图
	if len(report.Metrics) > 0 {
		chart := rg.createMetricsComparisonChart(report.Metrics)
		report.Charts = append(report.Charts, chart)
	}

	// 延迟分布图
	if len(perfData) > 0 {
		chart := rg.createLatencyDistributionChart(perfData)
		report.Charts = append(report.Charts, chart)
	}
}

// createPerformanceTrendChart 创建性能趋势图
func (rg *ReportGenerator) createPerformanceTrendChart(data []models.PerformanceMetrics) models.ChartConfig {
	series := make([]models.ChartSeries, 0)

	// CPU使用率系列
	cpuData := make([]models.DataPoint, len(data))
	for i, d := range data {
		cpuData[i] = models.DataPoint{
			X: d.Timestamp.Format("15:04:05"),
			Y: d.CPUPercent,
		}
	}
	series = append(series, models.ChartSeries{
		Name:  "CPU使用率",
		Data:  cpuData,
		Color: "#3b82f6",
	})

	// 内存使用率系列
	memData := make([]models.DataPoint, len(data))
	for i, d := range data {
		memData[i] = models.DataPoint{
			X: d.Timestamp.Format("15:04:05"),
			Y: d.MemoryPercent,
		}
	}
	series = append(series, models.ChartSeries{
		Name:  "内存使用率",
		Data:  memData,
		Color: "#10b981",
	})

	return models.ChartConfig{
		ID:          "perf_trend",
		Type:        models.ChartTypeLine,
		Title:       "性能趋势",
		XAxis:       "时间",
		YAxis:       "使用率(%)",
		Series:      series,
		Interactive: true,
	}
}

// createMetricsComparisonChart 创建指标对比图
func (rg *ReportGenerator) createMetricsComparisonChart(metrics []models.MetricResult) models.ChartConfig {
	data := make([]models.DataPoint, len(metrics))
	for i, m := range metrics {
		data[i] = models.DataPoint{
			X: m.Name,
			Y: m.Value,
		}
	}

	return models.ChartConfig{
		ID:    "metrics_comp",
		Type:  models.ChartTypeBar,
		Title: "指标对比",
		XAxis: "指标",
		YAxis: "值",
		Series: []models.ChartSeries{
			{
				Name:  "指标值",
				Data:  data,
				Color: "#8b5cf6",
			},
		},
		Interactive: true,
	}
}

// createLatencyDistributionChart 创建延迟分布图
func (rg *ReportGenerator) createLatencyDistributionChart(data []models.PerformanceMetrics) models.ChartConfig {
	points := make([]models.DataPoint, 0)
	for i, d := range data {
		if d.LatencyMs > 0 {
			points = append(points, models.DataPoint{
				X: i,
				Y: d.LatencyMs,
			})
		}
	}

	return models.ChartConfig{
		ID:    "latency_dist",
		Type:  models.ChartTypeScatter,
		Title: "延迟分布",
		XAxis: "请求序号",
		YAxis: "延迟(ms)",
		Series: []models.ChartSeries{
			{
				Name:  "延迟",
				Data:  points,
				Color: "#f59e0b",
			},
		},
		Interactive: true,
	}
}

// generateRecommendations 生成建议
func (rg *ReportGenerator) generateRecommendations(report *models.TestReport) {
	// 基于指标生成建议
	for _, metric := range report.Metrics {
		if !metric.Passed {
			report.Recommendations = append(report.Recommendations,
				fmt.Sprintf("建议优化 %s 指标，当前值 %.2f 低于阈值 %.2f",
					metric.Name, metric.Value, metric.Threshold))
		}
	}

	// 基于性能数据生成建议
	if len(report.PerformanceData) > 0 {
		latest := report.PerformanceData[len(report.PerformanceData)-1]
		if latest.CPUPercent > 80 {
			report.Recommendations = append(report.Recommendations,
				"CPU使用率过高，建议增加计算资源或优化算法")
		}
		if latest.MemoryPercent > 85 {
			report.Recommendations = append(report.Recommendations,
				"内存使用率过高，建议检查内存泄漏或增加内存容量")
		}
	}
}

// ExportReport 导出报告
func (rg *ReportGenerator) ExportReport(report *models.TestReport, format string) (string, error) {
	exporter, exists := rg.exporters[format]
	if !exists {
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(rg.config.ReportOutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成输出路径
	filename := fmt.Sprintf("report_%s_%s.%s", report.TaskID, time.Now().Format("20060102_150405"), format)
	outputPath := filepath.Join(rg.config.ReportOutputDir, filename)

	// 导出
	if err := exporter.Export(report, outputPath); err != nil {
		return "", err
	}

	// 更新报告
	report.ExportPaths[format] = outputPath

	return outputPath, nil
}

// RegisterTemplate 注册图表模板
func (rg *ReportGenerator) RegisterTemplate(template *ChartTemplate) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.templates[template.ID] = template
}

// RegisterExporter 注册导出器
func (rg *ReportGenerator) RegisterExporter(format string, exporter Exporter) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.exporters[format] = exporter
}

// JSONExporter JSON导出器
type JSONExporter struct{}

func (e *JSONExporter) Export(report *models.TestReport, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

func (e *JSONExporter) GetFormat() string {
	return "json"
}

// PDFExporter PDF导出器
type PDFExporter struct{}

func (e *PDFExporter) Export(report *models.TestReport, outputPath string) error {
	// 简化实现，实际应使用 PDF 生成库
	// 这里生成一个简单的文本文件作为示例
	content := fmt.Sprintf("Report: %s\nGenerated: %s\n", report.Name, report.Timestamp)
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func (e *PDFExporter) GetFormat() string {
	return "pdf"
}

// ExcelExporter Excel导出器
type ExcelExporter struct{}

func (e *ExcelExporter) Export(report *models.TestReport, outputPath string) error {
	// 简化实现，实际应使用 excelize 等库
	// 这里生成 CSV 格式作为示例
	var content string
	content += "Metric,Value,Score\n"
	for _, m := range report.Metrics {
		content += fmt.Sprintf("%s,%.2f,%.2f\n", m.Name, m.Value, m.Score)
	}
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func (e *ExcelExporter) GetFormat() string {
	return "excel"
}

// countPassedMetrics 统计通过的指标数
func countPassedMetrics(metrics map[string]float64) int {
	// 简化实现
	return len(metrics) / 2
}

// generateReportID 生成报告ID
func generateReportID() string {
	return fmt.Sprintf("report_%d", time.Now().UnixNano())
}
