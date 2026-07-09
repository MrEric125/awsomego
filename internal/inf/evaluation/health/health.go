package health

import (
	"awesome/internal/inf/evaluation/logger"
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	checks   map[string]HealthCheck
	logger   *logger.Logger
	mu       sync.RWMutex
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthStatusReport 健康状态报告
type HealthStatusReport struct {
	Status    HealthStatus                 `json:"status"`
	Timestamp time.Time                    `json:"timestamp"`
	Checks    map[string]HealthCheckResult `json:"checks"`
	System    SystemInfo                   `json:"system"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	MemAllocMB   uint64 `json:"mem_alloc_mb"`
	MemTotalMB   uint64 `json:"mem_total_mb"`
	MemSysMB     uint64 `json:"mem_sys_mb"`
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(interval time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		checks:   make(map[string]HealthCheck),
		logger:   logger.GetLogger().WithComponent("health_checker"),
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterCheck 注册健康检查
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[check.Name()] = check
}

// Start 启动健康检查
func (hc *HealthChecker) Start() error {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return fmt.Errorf("health checker already running")
	}
	hc.running = true
	hc.mu.Unlock()

	go hc.checkLoop()
	return nil
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.running {
		hc.cancel()
		hc.running = false
	}
}

// checkLoop 检查循环
func (hc *HealthChecker) checkLoop() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.runChecks()
		}
	}
}

// runChecks 运行所有检查
func (hc *HealthChecker) runChecks() {
	hc.mu.RLock()
	checks := make([]HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	for _, check := range checks {
		go hc.runCheck(check)
	}
}

// runCheck 运行单个检查
func (hc *HealthChecker) runCheck(check HealthCheck) {
	ctx, cancel := context.WithTimeout(hc.ctx, 5*time.Second)
	defer cancel()

	startTime := time.Now()
	err := check.Check(ctx)
	duration := time.Since(startTime)

	if err != nil {
		hc.logger.Warn("Health check failed",
			zap.String("check", check.Name()),
			zap.Error(err),
			zap.Duration("duration", duration))
	} else {
		hc.logger.Debug("Health check passed",
			zap.String("check", check.Name()),
			zap.Duration("duration", duration))
	}
}

// GetStatus 获取健康状态
func (hc *HealthChecker) GetStatus() *HealthStatusReport {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	report := &HealthStatusReport{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]HealthCheckResult),
		System:    hc.getSystemInfo(),
	}

	// 运行所有检查
	for name, check := range hc.checks {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		startTime := time.Now()
		err := check.Check(ctx)
		duration := time.Since(startTime)
		cancel()

		result := HealthCheckResult{
			Name:      name,
			Duration:  duration,
			Timestamp: time.Now(),
		}

		if err != nil {
			result.Status = HealthStatusUnhealthy
			result.Error = err.Error()
			report.Status = HealthStatusUnhealthy
		} else {
			result.Status = HealthStatusHealthy
		}

		report.Checks[name] = result
	}

	return report
}

// getSystemInfo 获取系统信息
func (hc *HealthChecker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		MemAllocMB:   m.Alloc / 1024 / 1024,
		MemTotalMB:   m.TotalAlloc / 1024 / 1024,
		MemSysMB:     m.Sys / 1024 / 1024,
	}
}

// DatabaseHealthCheck 数据库健康检查
type DatabaseHealthCheck struct {
	name string
	ping func() error
}

// NewDatabaseHealthCheck 创建数据库健康检查
func NewDatabaseHealthCheck(name string, ping func() error) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		name: name,
		ping: ping,
	}
}

// Name 获取检查名称
func (c *DatabaseHealthCheck) Name() string {
	return c.name
}

// Check 执行检查
func (c *DatabaseHealthCheck) Check(ctx context.Context) error {
	if c.ping == nil {
		return fmt.Errorf("ping function not set")
	}
	return c.ping()
}

// MemoryHealthCheck 内存健康检查
type MemoryHealthCheck struct {
	name        string
	maxMemoryMB uint64
}

// NewMemoryHealthCheck 创建内存健康检查
func NewMemoryHealthCheck(name string, maxMemoryMB uint64) *MemoryHealthCheck {
	return &MemoryHealthCheck{
		name:        name,
		maxMemoryMB: maxMemoryMB,
	}
}

// Name 获取检查名称
func (c *MemoryHealthCheck) Name() string {
	return c.name
}

// Check 执行检查
func (c *MemoryHealthCheck) Check(ctx context.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := m.Alloc / 1024 / 1024
	if allocMB > c.maxMemoryMB {
		return fmt.Errorf("memory usage %d MB exceeds limit %d MB", allocMB, c.maxMemoryMB)
	}

	return nil
}

// GoroutineHealthCheck Goroutine健康检查
type GoroutineHealthCheck struct {
	name         string
	maxGoroutine int
}

// NewGoroutineHealthCheck 创建Goroutine健康检查
func NewGoroutineHealthCheck(name string, maxGoroutine int) *GoroutineHealthCheck {
	return &GoroutineHealthCheck{
		name:         name,
		maxGoroutine: maxGoroutine,
	}
}

// Name 获取检查名称
func (c *GoroutineHealthCheck) Name() string {
	return c.name
}

// Check 执行检查
func (c *GoroutineHealthCheck) Check(ctx context.Context) error {
	count := runtime.NumGoroutine()
	if count > c.maxGoroutine {
		return fmt.Errorf("goroutine count %d exceeds limit %d", count, c.maxGoroutine)
	}

	return nil
}

// CustomHealthCheck 自定义健康检查
type CustomHealthCheck struct {
	name  string
	check func(ctx context.Context) error
}

// NewCustomHealthCheck 创建自定义健康检查
func NewCustomHealthCheck(name string, check func(ctx context.Context) error) *CustomHealthCheck {
	return &CustomHealthCheck{
		name:  name,
		check: check,
	}
}

// Name 获取检查名称
func (c *CustomHealthCheck) Name() string {
	return c.name
}

// Check 执行检查
func (c *CustomHealthCheck) Check(ctx context.Context) error {
	if c.check == nil {
		return fmt.Errorf("check function not set")
	}
	return c.check(ctx)
}
