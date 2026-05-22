package retry

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// Policy 重试策略
type Policy struct {
	maxRetries     int
	initialDelay   time.Duration
	maxDelay       time.Duration
	multiplier     float64
	jitter         bool
	retryableErrors map[string]bool
}

// Option 重试策略选项
type Option func(*Policy)

// NewPolicy 创建重试策略
func NewPolicy(opts ...Option) *Policy {
	p := &Policy{
		maxRetries:     3,
		initialDelay:   100 * time.Millisecond,
		maxDelay:       5 * time.Second,
		multiplier:     2.0,
		jitter:         true,
		retryableErrors: map[string]bool{
			"rate_limit_exceeded": true,
			"timeout":             true,
			"server_error":        true,
			"connection_error":    true,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) Option {
	return func(p *Policy) {
		p.maxRetries = n
	}
}

// WithInitialDelay 设置初始延迟
func WithInitialDelay(d time.Duration) Option {
	return func(p *Policy) {
		p.initialDelay = d
	}
}

// WithMaxDelay 设置最大延迟
func WithMaxDelay(d time.Duration) Option {
	return func(p *Policy) {
		p.maxDelay = d
	}
}

// WithMultiplier 设置延迟倍数
func WithMultiplier(m float64) Option {
	return func(p *Policy) {
		p.multiplier = m
	}
}

// WithJitter 设置是否使用抖动
func WithJitter(j bool) Option {
	return func(p *Policy) {
		p.jitter = j
	}
}

// Execute 执行带重试的操作
func (p *Policy) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := p.initialDelay

	for i := 0; i <= p.maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if !p.isRetryable(err) {
			return err
		}

		// 最后一次尝试不等待
		if i == p.maxRetries {
			break
		}

		// 计算延迟
		currentDelay := delay
		if p.jitter {
			// 添加随机抖动 (0.5x - 1.5x)
			jitter := 0.5 + rand.Float64()
			currentDelay = time.Duration(float64(currentDelay) * jitter)
		}

		// 等待或取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(currentDelay):
		}

		// 增加延迟
		delay = time.Duration(float64(delay) * p.multiplier)
		if delay > p.maxDelay {
			delay = p.maxDelay
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryable 检查错误是否可重试
func (p *Policy) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查错误类型
	errStr := err.Error()
	for key := range p.retryableErrors {
		if contains(errStr, key) {
			return true
		}
	}

	// 检查是否为超时或取消
	if err == context.DeadlineExceeded || err == context.Canceled {
		return true
	}

	return false
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryableError 可重试错误
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError 创建可重试错误
func NewRetryableError(err error, retryable bool) *RetryableError {
	return &RetryableError{
		Err:       err,
		Retryable: retryable,
	}
}
