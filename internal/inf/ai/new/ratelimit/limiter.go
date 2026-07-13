package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Reserve() Reservation
}

// Reservation 预留
type Reservation struct {
	ok        bool
	timeToAct time.Time
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	rate       int           // 每秒令牌数
	burst      int           // 桶容量
	tokens     float64       // 当前令牌数
	lastUpdate time.Time     // 上次更新时间
	mu         sync.Mutex
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(rate, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Wait 等待直到可以请求
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		if l.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
		}
	}
}

// Reserve 预留令牌
func (l *TokenBucketLimiter) Reserve() Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	if l.tokens >= 1 {
		l.tokens--
		return Reservation{ok: true, timeToAct: time.Now()}
	}

	// 计算等待时间
	waitTime := time.Duration((1 - l.tokens) / float64(l.rate) * float64(time.Second))
	return Reservation{ok: false, timeToAct: time.Now().Add(waitTime)}
}

// refill 补充令牌
func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastUpdate).Seconds()
	l.tokens += float64(l.rate) * elapsed
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}
	l.lastUpdate = now
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	limit      int
	window     time.Duration
	requests   []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0, limit),
	}
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// 移除过期请求
	valid := l.requests[:0]
	for _, t := range l.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	l.requests = valid

	// 检查是否超过限制
	if len(l.requests) >= l.limit {
		return false
	}

	l.requests = append(l.requests, now)
	return true
}

// Wait 等待直到可以请求
func (l *SlidingWindowLimiter) Wait(ctx context.Context) error {
	for {
		if l.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
		}
	}
}

// Reserve 预留
func (l *SlidingWindowLimiter) Reserve() Reservation {
	return Reservation{ok: l.Allow(), timeToAct: time.Now()}
}

// Rate 限流速率
func (l *SlidingWindowLimiter) Rate() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.requests), l.limit
}

// LeakyBucketLimiter 漏桶限流器
type LeakyBucketLimiter struct {
	rate       time.Duration // 漏出速率
	capacity   int           // 桶容量
	water      int           // 当前水量
	lastLeak   time.Time     // 上次漏出时间
	mu         sync.Mutex
}

// NewLeakyBucketLimiter 创建漏桶限流器
func NewLeakyBucketLimiter(rate time.Duration, capacity int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		rate:     rate,
		capacity: capacity,
		water:    0,
		lastLeak: time.Now(),
	}
}

// Allow 检查是否允许请求
func (l *LeakyBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.leak()
	if l.water < l.capacity {
		l.water++
		return true
	}
	return false
}

// Wait 等待直到可以请求
func (l *LeakyBucketLimiter) Wait(ctx context.Context) error {
	for {
		if l.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(l.rate):
		}
	}
}

// Reserve 预留
func (l *LeakyBucketLimiter) Reserve() Reservation {
	return Reservation{ok: l.Allow(), timeToAct: time.Now()}
}

// leak 漏出
func (l *LeakyBucketLimiter) leak() {
	now := time.Now()
	elapsed := now.Sub(l.lastLeak)
	leaked := int(elapsed / l.rate)
	if leaked > 0 {
		l.water -= leaked
		if l.water < 0 {
			l.water = 0
		}
		l.lastLeak = now
	}
}

// MultiLimiter 多级限流器
type MultiLimiter struct {
	limiters []Limiter
}

// NewMultiLimiter 创建多级限流器
func NewMultiLimiter(limiters ...Limiter) *MultiLimiter {
	return &MultiLimiter{limiters: limiters}
}

// Allow 检查是否允许请求
func (l *MultiLimiter) Allow() bool {
	for _, limiter := range l.limiters {
		if !limiter.Allow() {
			return false
		}
	}
	return true
}

// Wait 等待直到可以请求
func (l *MultiLimiter) Wait(ctx context.Context) error {
	for _, limiter := range l.limiters {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Reserve 预留
func (l *MultiLimiter) Reserve() Reservation {
	for _, limiter := range l.limiters {
			r := limiter.Reserve()
		if !r.ok {
			return r
		}
	}
	return Reservation{ok: true, timeToAct: time.Now()}
}
