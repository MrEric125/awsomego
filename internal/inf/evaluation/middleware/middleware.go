package middleware

import (
	"awesome/internal/inf/evaluation/logger"
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Middleware 中间件接口
type Middleware interface {
	Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error)
	Name() string
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
	logger      *logger.Logger
	mu          sync.RWMutex
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
		logger:      logger.GetLogger().WithComponent("middleware_chain"),
	}
}

// AddMiddleware 添加中间件
func (mc *MiddlewareChain) AddMiddleware(middleware Middleware) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.middlewares = append(mc.middlewares, middleware)
}

// Execute 执行中间件链
func (mc *MiddlewareChain) Execute(ctx context.Context, req interface{}, handler func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	mc.mu.RLock()
	middlewares := make([]Middleware, len(mc.middlewares))
	copy(middlewares, mc.middlewares)
	mc.mu.RUnlock()

	// 构建中间件链
	current := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		next := current
		current = func(ctx context.Context, req interface{}) (interface{}, error) {
			return middleware.Process(ctx, req, next)
		}
	}

	return current(ctx, req)
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	logger *logger.Logger
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger.GetLogger().WithComponent("logging_middleware"),
	}
}

// Process 处理请求
func (m *LoggingMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	startTime := time.Now()

	// 记录请求
	m.logger.Info("Request started",
		zap.Any("request", req),
		zap.String("middleware", "logging"))

	// 执行下一个处理器
	resp, err := next(ctx, req)

	// 记录响应
	duration := time.Since(startTime)
	if err != nil {
		m.logger.Error("Request failed",
			zap.Error(err),
			zap.Duration("duration", duration),
			zap.String("middleware", "logging"))
	} else {
		m.logger.Info("Request completed",
			zap.Duration("duration", duration),
			zap.String("middleware", "logging"))
	}

	return resp, err
}

// Name 获取中间件名称
func (m *LoggingMiddleware) Name() string {
	return "logging"
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	logger  *logger.Logger
	metrics map[string]*OperationMetrics
	mu      sync.RWMutex
}

// OperationMetrics 操作指标
type OperationMetrics struct {
	TotalRequests  int64
	SuccessCount   int64
	FailureCount   int64
	TotalDuration  time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
	AvgDuration    time.Duration
	LastAccessTime time.Time
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		logger:  logger.GetLogger().WithComponent("metrics_middleware"),
		metrics: make(map[string]*OperationMetrics),
	}
}

// Process 处理请求
func (m *MetricsMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	startTime := time.Now()

	// 执行下一个处理器
	resp, err := next(ctx, req)

	// 更新指标
	duration := time.Since(startTime)
	operation := getOperationName(req)

	m.updateMetrics(operation, duration, err == nil)

	return resp, err
}

// Name 获取中间件名称
func (m *MetricsMiddleware) Name() string {
	return "metrics"
}

// updateMetrics 更新指标
func (m *MetricsMiddleware) updateMetrics(operation string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.metrics[operation]
	if !exists {
		metrics = &OperationMetrics{
			MinDuration: time.Duration(1<<63 - 1),
		}
		m.metrics[operation] = metrics
	}

	metrics.TotalRequests++
	metrics.TotalDuration += duration
	metrics.LastAccessTime = time.Now()

	if success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	metrics.AvgDuration = time.Duration(int64(metrics.TotalDuration) / metrics.TotalRequests)
}

// GetMetrics 获取指标
func (m *MetricsMiddleware) GetMetrics() map[string]*OperationMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*OperationMetrics)
	for k, v := range m.metrics {
		metricsCopy := *v
		result[k] = &metricsCopy
	}

	return result
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	logger  *logger.Logger
	limiter *RateLimiter
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(requestsPerSecond int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		logger:  logger.GetLogger().WithComponent("rate_limit_middleware"),
		limiter: NewRateLimiter(requestsPerSecond),
	}
}

// Process 处理请求
func (m *RateLimitMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	if !m.limiter.Allow() {
		m.logger.Warn("Rate limit exceeded",
			zap.String("middleware", "rate_limit"))
		return nil, fmt.Errorf("rate limit exceeded")
	}

	return next(ctx, req)
}

// Name 获取中间件名称
func (m *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

// RateLimiter 限流器
type RateLimiter struct {
	requestsPerSecond int
	tokens            int
	maxTokens         int
	lastRefill        time.Time
	mu                sync.Mutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		requestsPerSecond: requestsPerSecond,
		tokens:            requestsPerSecond,
		maxTokens:         requestsPerSecond,
		lastRefill:        time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(rl.requestsPerSecond))

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// 检查是否有可用令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// TimeoutMiddleware 超时中间件
type TimeoutMiddleware struct {
	logger  *logger.Logger
	timeout time.Duration
}

// NewTimeoutMiddleware 创建超时中间件
func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		logger:  logger.GetLogger().WithComponent("timeout_middleware"),
		timeout: timeout,
	}
}

// Process 处理请求
func (m *TimeoutMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	resultChan := make(chan struct {
		resp interface{}
		err  error
	}, 1)

	go func() {
		resp, err := next(ctx, req)
		resultChan <- struct {
			resp interface{}
			err  error
		}{resp: resp, err: err}
	}()

	select {
	case result := <-resultChan:
		return result.resp, result.err
	case <-ctx.Done():
		m.logger.Warn("Request timeout",
			zap.Duration("timeout", m.timeout),
			zap.String("middleware", "timeout"))
		return nil, fmt.Errorf("request timeout after %v", m.timeout)
	}
}

// Name 获取中间件名称
func (m *TimeoutMiddleware) Name() string {
	return "timeout"
}

// RecoveryMiddleware 恢复中间件
type RecoveryMiddleware struct {
	logger *logger.Logger
}

// NewRecoveryMiddleware 创建恢复中间件
func NewRecoveryMiddleware() *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger.GetLogger().WithComponent("recovery_middleware"),
	}
}

// Process 处理请求
func (m *RecoveryMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("Panic recovered",
				zap.Any("panic", r),
				zap.String("middleware", "recovery"))
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return next(ctx, req)
}

// Name 获取中间件名称
func (m *RecoveryMiddleware) Name() string {
	return "recovery"
}

// TracingMiddleware 追踪中间件
type TracingMiddleware struct {
	logger *logger.Logger
}

// NewTracingMiddleware 创建追踪中间件
func NewTracingMiddleware() *TracingMiddleware {
	return &TracingMiddleware{
		logger: logger.GetLogger().WithComponent("tracing_middleware"),
	}
}

// Process 处理请求
func (m *TracingMiddleware) Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	// 生成trace ID
	traceID := generateTraceID()
	ctx = context.WithValue(ctx, "trace_id", traceID)

	m.logger.Info("Trace started",
		zap.String("trace_id", traceID),
		zap.String("middleware", "tracing"))

	// 执行下一个处理器
	resp, err := next(ctx, req)

	m.logger.Info("Trace completed",
		zap.String("trace_id", traceID),
		zap.String("middleware", "tracing"))

	return resp, err
}

// Name 获取中间件名称
func (m *TracingMiddleware) Name() string {
	return "tracing"
}

// getOperationName 获取操作名称
func getOperationName(req interface{}) string {
	// 简化实现，实际应根据请求类型提取
	return fmt.Sprintf("%T", req)
}

func generateTraceID() string {
	return fmt.Sprintf("trace_%d", time.Now().UnixNano())
}
