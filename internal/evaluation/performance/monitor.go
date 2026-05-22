package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"awesome/internal/evaluation/config"
	"awesome/internal/evaluation/models"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	config          *config.EvaluationConfig
	metricsBuffer   []models.PerformanceMetrics
	mu              sync.RWMutex
	stopChan        chan struct{}
	alertChan       chan *models.Alert
	monitoring      bool
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(cfg *config.EvaluationConfig) *PerformanceMonitor {
	return &PerformanceMonitor{
		config:        cfg,
		metricsBuffer: make([]models.PerformanceMetrics, 0, 10000),
		stopChan:      make(chan struct{}),
		alertChan:     make(chan *models.Alert, 1000),
		monitoring:    false,
	}
}

// Start 启动监控
func (pm *PerformanceMonitor) Start(ctx context.Context) error {
	pm.mu.Lock()
	if pm.monitoring {
		pm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	pm.monitoring = true
	pm.mu.Unlock()

	go pm.monitorLoop(ctx)
	return nil
}

// Stop 停止监控
func (pm *PerformanceMonitor) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.monitoring {
		close(pm.stopChan)
		pm.monitoring = false
	}
}

// monitorLoop 监控循环
func (pm *PerformanceMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			metrics := pm.collectMetrics()
			pm.recordMetrics(metrics)
			pm.checkThresholds(metrics)
		}
	}
}

// collectMetrics 收集性能指标
func (pm *PerformanceMonitor) collectMetrics() models.PerformanceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := models.PerformanceMetrics{
		Timestamp:     time.Now(),
		CPUPercent:    pm.getCPUUsage(),
		MemoryPercent: float64(m.Sys) / float64(1<<30), // GB
		MemoryMB:      float64(m.Alloc) / 1024 / 1024,
	}

	// GPU监控（如果可用）
	if gpuMetrics, err := pm.getGPUUsage(); err == nil {
		metrics.GPUPercent = gpuMetrics.UsagePercent
		metrics.GPUMemoryMB = gpuMetrics.MemoryMB
	}

	return metrics
}

// getCPUUsage 获取CPU使用率
func (pm *PerformanceMonitor) getCPUUsage() float64 {
	// 简化实现，实际应使用 gopsutil 等库
	return 0.0
}

// GPUMetrics GPU指标
type GPUMetrics struct {
	UsagePercent float64
	MemoryMB     float64
}

// getGPUUsage 获取GPU使用率
func (pm *PerformanceMonitor) getGPUUsage() (*GPUMetrics, error) {
	// 简化实现，实际应使用 NVIDIA GPU SDK
	return nil, fmt.Errorf("GPU monitoring not available")
}

// recordMetrics 记录指标
func (pm *PerformanceMonitor) recordMetrics(metrics models.PerformanceMetrics) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.metricsBuffer = append(pm.metricsBuffer, metrics)
	
	// 限制缓冲区大小
	if len(pm.metricsBuffer) > 10000 {
		pm.metricsBuffer = pm.metricsBuffer[1000:]
	}
}

// checkThresholds 检查阈值
func (pm *PerformanceMonitor) checkThresholds(metrics models.PerformanceMetrics) {
	thresholds := pm.config.ResourceAlertThresholds

	// CPU阈值检查
	if metrics.CPUPercent > thresholds.CPUPercent {
		pm.sendAlert(&models.Alert{
			Type:      "performance",
			Severity:  "warning",
			Title:     "CPU使用率过高",
			Message:   fmt.Sprintf("CPU使用率 %.2f%% 超过阈值 %.2f%%", metrics.CPUPercent, thresholds.CPUPercent),
			Source:    "performance_monitor",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"cpu_percent": metrics.CPUPercent,
				"threshold":   thresholds.CPUPercent,
			},
		})
	}

	// 内存阈值检查
	if metrics.MemoryPercent > thresholds.MemoryPercent {
		pm.sendAlert(&models.Alert{
			Type:      "performance",
			Severity:  "warning",
			Title:     "内存使用率过高",
			Message:   fmt.Sprintf("内存使用率 %.2f%% 超过阈值 %.2f%%", metrics.MemoryPercent, thresholds.MemoryPercent),
			Source:    "performance_monitor",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"memory_percent": metrics.MemoryPercent,
				"threshold":      thresholds.MemoryPercent,
			},
		})
	}

	// 延迟阈值检查
	if metrics.LatencyMs > thresholds.LatencyMs {
		pm.sendAlert(&models.Alert{
			Type:      "performance",
			Severity:  "warning",
			Title:     "延迟过高",
			Message:   fmt.Sprintf("延迟 %.2fms 超过阈值 %.2fms", metrics.LatencyMs, thresholds.LatencyMs),
			Source:    "performance_monitor",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"latency_ms": metrics.LatencyMs,
				"threshold":  thresholds.LatencyMs,
			},
		})
	}
}

// sendAlert 发送告警
func (pm *PerformanceMonitor) sendAlert(alert *models.Alert) {
	select {
	case pm.alertChan <- alert:
	default:
		// 告警通道已满，丢弃
	}
}

// GetAlertChannel 获取告警通道
func (pm *PerformanceMonitor) GetAlertChannel() <-chan *models.Alert {
	return pm.alertChan
}

// GetMetrics 获取监控数据
func (pm *PerformanceMonitor) GetMetrics(duration time.Duration) []models.PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if duration == 0 {
		return pm.metricsBuffer
	}

	cutoff := time.Now().Add(-duration)
	result := make([]models.PerformanceMetrics, 0)
	for _, m := range pm.metricsBuffer {
		if m.Timestamp.After(cutoff) {
			result = append(result, m)
		}
	}
	return result
}

// GetLatestMetrics 获取最新指标
func (pm *PerformanceMonitor) GetLatestMetrics() *models.PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.metricsBuffer) == 0 {
		return nil
	}
	latest := pm.metricsBuffer[len(pm.metricsBuffer)-1]
	return &latest
}

// StabilityTester 稳定性测试器
type StabilityTester struct {
	config *config.EvaluationConfig
}

// NewStabilityTester 创建稳定性测试器
func NewStabilityTester(cfg *config.EvaluationConfig) *StabilityTester {
	return &StabilityTester{
		config: cfg,
	}
}

// RunStabilityTest 运行稳定性测试
func (st *StabilityTester) RunStabilityTest(ctx context.Context, testFunc func() error) (*models.StabilityTestResult, error) {
	result := &models.StabilityTestResult{
		ID:          generateID(),
		StartTime:   time.Now(),
		Errors:      make([]models.StabilityError, 0),
		ResourceSnapshots: make([]models.ResourceSnapshot, 0),
	}

	// 创建性能监控器
	monitor := NewPerformanceMonitor(st.config)
	if err := monitor.Start(ctx); err != nil {
		return nil, err
	}
	defer monitor.Stop()

	// 运行测试
	ticker := time.NewTicker(st.config.StabilityCheckInterval)
	defer ticker.Stop()

	errorCounts := make(map[string]int)
	var peakMemoryMB float64
	var peakGPUMemoryMB float64

	for {
		select {
		case <-ctx.Done():
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, ctx.Err()

		case <-time.After(st.config.StabilityTestDuration):
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, nil

		case <-ticker.C:
			// 执行测试
			if err := testFunc(); err != nil {
				errorCounts[err.Error()]++
				result.FailureCount++
			} else {
				result.SuccessCount++
			}
			result.TotalRequests++

			// 记录资源快照
			if metrics := monitor.GetLatestMetrics(); metrics != nil {
				snapshot := models.ResourceSnapshot{
					Timestamp:   metrics.Timestamp,
					CPUPercent:  metrics.CPUPercent,
					MemoryMB:    metrics.MemoryMB,
					GPUPercent:  metrics.GPUPercent,
					GPUMemoryMB: metrics.GPUMemoryMB,
				}
				result.ResourceSnapshots = append(result.ResourceSnapshots, snapshot)

				// 更新峰值
				if metrics.MemoryMB > peakMemoryMB {
					peakMemoryMB = metrics.MemoryMB
				}
				if metrics.GPUMemoryMB > peakGPUMemoryMB {
					peakGPUMemoryMB = metrics.GPUMemoryMB
				}
			}
		}
	}

	// 检测内存泄漏
	result.MemoryLeaks = st.detectMemoryLeak(result.ResourceSnapshots)
	result.PeakMemoryMB = peakMemoryMB
	result.PeakGPUMemoryMB = peakGPUMemoryMB
	result.AvailabilityRate = float64(result.SuccessCount) / float64(result.TotalRequests) * 100

	// 转换错误计数
	for errMsg, count := range errorCounts {
		result.Errors = append(result.Errors, models.StabilityError{
			Timestamp: time.Now(),
			Type:      "test_error",
			Message:   errMsg,
			Count:     count,
		})
	}

	return result, nil
}

// detectMemoryLeak 检测内存泄漏
func (st *StabilityTester) detectMemoryLeak(snapshots []models.ResourceSnapshot) bool {
	if len(snapshots) < 10 {
		return false
	}

	// 简单的线性回归检测内存增长趋势
	// 如果内存持续增长，可能存在泄漏
	firstHalf := snapshots[:len(snapshots)/2]
	secondHalf := snapshots[len(snapshots)/2:]

	var firstAvg, secondAvg float64
	for _, s := range firstHalf {
		firstAvg += s.MemoryMB
	}
	firstAvg /= float64(len(firstHalf))

	for _, s := range secondHalf {
		secondAvg += s.MemoryMB
	}
	secondAvg /= float64(len(secondHalf))

	// 如果后半段平均内存比前半段增长超过20%，可能存在泄漏
	return secondAvg > firstAvg*1.2
}

// generateID 生成ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
