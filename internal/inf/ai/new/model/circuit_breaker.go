package model

import (
	"sync/atomic"
	"time"
)

// CircuitState 熔断器状态
type CircuitState int32

const (
	// StateClosed 关闭状态（正常）
	StateClosed CircuitState = iota
	// StateOpen 开启状态（熔断）
	StateOpen
	// StateHalfOpen 半开状态（尝试恢复）
	StateHalfOpen
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            int32(StateClosed),
	}
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	state := CircuitState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		cb.mu.RLock()
		defer cb.mu.RUnlock()
		// 检查是否可以尝试恢复
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			// 双重检查
			if time.Since(cb.lastFailureTime) > cb.resetTimeout {
				atomic.StoreInt32(&cb.state, int32(StateHalfOpen))
				cb.mu.Unlock()
				return true
			}
			cb.mu.Unlock()
			cb.mu.RLock()
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, int32(StateClosed))
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	failures := atomic.AddInt32(&cb.failures, 1)

	cb.mu.Lock()
	cb.lastFailureTime = time.Now()
	cb.mu.Unlock()

	if failures >= int32(cb.failureThreshold) {
		atomic.StoreInt32(&cb.state, int32(StateOpen))
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() string {
	state := CircuitState(atomic.LoadInt32(&cb.state))
	switch state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, int32(StateClosed))
}
