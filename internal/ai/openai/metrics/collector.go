package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector 指标收集器
type Collector struct {
	requestCount    int64
	successCount    int64
	failureCount    int64
	totalDuration   int64 // 纳秒
	minDuration     int64
	maxDuration     int64
	lastRequestTime int64

	// 按模型统计
	modelStats map[string]*ModelStats
	mu         sync.RWMutex
}

// ModelStats 模型统计
type ModelStats struct {
	RequestCount    int64
	SuccessCount    int64
	FailureCount    int64
	TotalTokens     int64
	PromptTokens    int64
	CompletionTokens int64
	TotalDuration   int64
}

// Stats 统计信息
type Stats struct {
	TotalRequests   int64            `json:"total_requests"`
	SuccessCount    int64            `json:"success_count"`
	FailureCount    int64            `json:"failure_count"`
	SuccessRate     float64          `json:"success_rate"`
	AvgDuration     time.Duration    `json:"avg_duration"`
	MinDuration     time.Duration    `json:"min_duration"`
	MaxDuration     time.Duration    `json:"max_duration"`
	LastRequestTime time.Time        `json:"last_request_time"`
	ModelStats      map[string]*ModelStats `json:"model_stats"`
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		modelStats: make(map[string]*ModelStats),
		minDuration: 1<<63 - 1, // 最大值
	}
}

// RecordRequest 记录请求
func (c *Collector) RecordRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&c.requestCount, 1)
	atomic.StoreInt64(&c.lastRequestTime, time.Now().UnixNano())

	durationNs := duration.Nanoseconds()
	atomic.AddInt64(&c.totalDuration, durationNs)

	// 更新最小/最大持续时间
	for {
		min := atomic.LoadInt64(&c.minDuration)
		if durationNs >= min || atomic.CompareAndSwapInt64(&c.minDuration, min, durationNs) {
			break
		}
	}

	for {
		max := atomic.LoadInt64(&c.maxDuration)
		if durationNs <= max || atomic.CompareAndSwapInt64(&c.maxDuration, max, durationNs) {
			break
		}
	}

	if success {
		atomic.AddInt64(&c.successCount, 1)
	} else {
		atomic.AddInt64(&c.failureCount, 1)
	}
}

// RecordModelUsage 记录模型使用
func (c *Collector) RecordModelUsage(model string, promptTokens, completionTokens int, duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, exists := c.modelStats[model]
	if !exists {
		stats = &ModelStats{}
		c.modelStats[model] = stats
	}

	stats.RequestCount++
	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}
	stats.TotalTokens += int64(promptTokens + completionTokens)
	stats.PromptTokens += int64(promptTokens)
	stats.CompletionTokens += int64(completionTokens)
	stats.TotalDuration += duration.Nanoseconds()
}

// GetStats 获取统计信息
func (c *Collector) GetStats() *Stats {
	total := atomic.LoadInt64(&c.requestCount)
	success := atomic.LoadInt64(&c.successCount)
	failure := atomic.LoadInt64(&c.failureCount)
	totalDuration := atomic.LoadInt64(&c.totalDuration)
	minDuration := atomic.LoadInt64(&c.minDuration)
	maxDuration := atomic.LoadInt64(&c.maxDuration)
	lastRequestTime := atomic.LoadInt64(&c.lastRequestTime)

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total)
	}

	var avgDuration time.Duration
	if total > 0 {
		avgDuration = time.Duration(totalDuration / total)
	}

	c.mu.RLock()
	modelStatsCopy := make(map[string]*ModelStats)
	for k, v := range c.modelStats {
		statsCopy := *v
		modelStatsCopy[k] = &statsCopy
	}
	c.mu.RUnlock()

	return &Stats{
		TotalRequests:   total,
		SuccessCount:    success,
		FailureCount:    failure,
		SuccessRate:     successRate,
		AvgDuration:     avgDuration,
		MinDuration:     time.Duration(minDuration),
		MaxDuration:     time.Duration(maxDuration),
		LastRequestTime: time.Unix(0, lastRequestTime),
		ModelStats:      modelStatsCopy,
	}
}

// Reset 重置统计
func (c *Collector) Reset() {
	atomic.StoreInt64(&c.requestCount, 0)
	atomic.StoreInt64(&c.successCount, 0)
	atomic.StoreInt64(&c.failureCount, 0)
	atomic.StoreInt64(&c.totalDuration, 0)
	atomic.StoreInt64(&c.minDuration, 1<<63-1)
	atomic.StoreInt64(&c.maxDuration, 0)
	atomic.StoreInt64(&c.lastRequestTime, 0)

	c.mu.Lock()
	c.modelStats = make(map[string]*ModelStats)
	c.mu.Unlock()
}

// GetRequestRate 获取请求速率（请求/秒）
func (c *Collector) GetRequestRate(window time.Duration) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 简化实现，实际应该使用滑动窗口
	total := atomic.LoadInt64(&c.requestCount)
	if total == 0 {
		return 0
	}

	return float64(total) / window.Seconds()
}

// Histogram 直方图数据
type Histogram struct {
	Buckets []Bucket `json:"buckets"`
}

// Bucket 直方图桶
type Bucket struct {
	UpperBound time.Duration `json:"upper_bound"`
	Count      int64         `json:"count"`
}

// GetLatencyHistogram 获取延迟直方图
func (c *Collector) GetLatencyHistogram() *Histogram {
	// 预定义延迟桶
	buckets := []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	histogram := &Histogram{
		Buckets: make([]Bucket, len(buckets)),
	}

	// 简化实现，实际应该跟踪每个请求的延迟
	for i, bound := range buckets {
		histogram.Buckets[i] = Bucket{
			UpperBound: bound,
			Count:      0,
		}
	}

	return histogram
}
