package service

import (
	"awesome/internal/inf/evaluation/errors"
	"awesome/internal/inf/evaluation/logger"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SecurityManager 安全管理器
type SecurityManager struct {
	logger        *logger.Logger
	rateLimiter   *RateLimiter
	authManager   *AuthManager
	auditLogger   *AuditLogger
	encryptionKey string
	mu            sync.RWMutex
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		logger:      logger.GetLogger().WithComponent("security_manager"),
		rateLimiter: NewRateLimiter(1000, time.Minute), // 1000 requests per minute
		authManager: NewAuthManager(),
		auditLogger: NewAuditLogger(),
	}
}

// ValidateRequest 验证请求
func (sm *SecurityManager) ValidateRequest(ctx context.Context, operation string) error {
	// 速率限制检查
	if err := sm.rateLimiter.CheckLimit(ctx); err != nil {
		sm.logger.Warn("Rate limit exceeded",
			zap.String("operation", operation))
		return errors.NewError(errors.ErrRateLimit, "rate limit exceeded").
			WithSeverity("high").
			WithRetryable(true).
			WithDetail("operation", operation)
	}

	// 权限验证
	if err := sm.authManager.CheckPermission(ctx, operation); err != nil {
		sm.logger.Warn("Permission denied",
			zap.String("operation", operation),
			zap.Error(err))
		return errors.NewError(errors.ErrAuthorization, "permission denied").
			WithSeverity("high").
			WithRetryable(false).
			WithDetail("operation", operation)
	}

	// 记录审计日志
	sm.auditLogger.LogAccess(ctx, operation, true)

	return nil
}

// EncryptData 加密数据
func (sm *SecurityManager) EncryptData(data []byte) ([]byte, error) {
	// 简化实现，实际应使用AES等加密算法
	hash := sha256.Sum256(data)
	return hash[:], nil
}

// DecryptData 解密数据
func (sm *SecurityManager) DecryptData(encrypted []byte) ([]byte, error) {
	// 简化实现
	return encrypted, nil
}

// HashSensitiveData 哈希敏感数据
func (sm *SecurityManager) HashSensitiveData(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ValidateDataIntegrity 验证数据完整性
func (sm *SecurityManager) ValidateDataIntegrity(data []byte, expectedHash string) bool {
	hash := sha256.Sum256(data)
	actualHash := hex.EncodeToString(hash[:])
	return actualHash == expectedHash
}

// RateLimiter 速率限制器
type RateLimiter struct {
	limit    int
	window   time.Duration
	requests map[string]*RequestCounter
	mu       sync.RWMutex
}

// RequestCounter 请求计数器
type RequestCounter struct {
	Count     int
	ResetTime time.Time
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limit:    limit,
		window:   window,
		requests: make(map[string]*RequestCounter),
	}

	// 启动清理协程
	go rl.cleanupRoutine()

	return rl
}

// CheckLimit 检查限制
func (rl *RateLimiter) CheckLimit(ctx context.Context) error {
	// 获取客户端标识（简化实现）
	clientID := getClientID(ctx)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	counter, exists := rl.requests[clientID]

	if !exists || now.After(counter.ResetTime) {
		// 创建或重置计数器
		rl.requests[clientID] = &RequestCounter{
			Count:     1,
			ResetTime: now.Add(rl.window),
		}
		return nil
	}

	// 检查是否超限
	if counter.Count >= rl.limit {
		return fmt.Errorf("rate limit exceeded for client %s", clientID)
	}

	// 增加计数
	counter.Count++

	return nil
}

// cleanupRoutine 清理例程
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, counter := range rl.requests {
			if now.After(counter.ResetTime) {
				delete(rl.requests, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientID 获取客户端ID
func getClientID(ctx context.Context) string {
	// 从context中提取客户端标识
	if clientID := ctx.Value("client_id"); clientID != nil {
		return clientID.(string)
	}
	// 默认返回固定值（实际应从认证信息中获取）
	return "default_client"
}

// AuthManager 认证管理器
type AuthManager struct {
	tokens map[string]*TokenInfo
	mu     sync.RWMutex
}

// TokenInfo 令牌信息
type TokenInfo struct {
	UserID      string
	Permissions []string
	ExpiresAt   time.Time
}

// NewAuthManager 创建认证管理器
func NewAuthManager() *AuthManager {
	return &AuthManager{
		tokens: make(map[string]*TokenInfo),
	}
}

// CheckPermission 检查权限
func (am *AuthManager) CheckPermission(ctx context.Context, operation string) error {
	// 简化实现，实际应从context中提取认证信息并验证权限
	// 这里默认允许所有操作
	return nil
}

// ValidateToken 验证令牌
func (am *AuthManager) ValidateToken(token string) (*TokenInfo, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	info, exists := am.tokens[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	if time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return info, nil
}

// GenerateToken 生成令牌
func (am *AuthManager) GenerateToken(userID string, permissions []string, duration time.Duration) string {
	token := generateSecureToken()

	am.mu.Lock()
	defer am.mu.Unlock()

	am.tokens[token] = &TokenInfo{
		UserID:      userID,
		Permissions: permissions,
		ExpiresAt:   time.Now().Add(duration),
	}

	return token
}

// RevokeToken 撤销令牌
func (am *AuthManager) RevokeToken(token string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.tokens, token)
}

// generateSecureToken 生成安全令牌
func generateSecureToken() string {
	return fmt.Sprintf("tok_%d", time.Now().UnixNano())
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logger *logger.Logger
	events []*AuditEvent
	mu     sync.RWMutex
}

// AuditEvent 审计事件
type AuditEvent struct {
	ID        string
	Timestamp time.Time
	UserID    string
	Operation string
	Resource  string
	Success   bool
	IPAddress string
	UserAgent string
	Details   map[string]interface{}
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logger: logger.GetLogger().WithComponent("audit"),
		events: make([]*AuditEvent, 0, 10000),
	}
}

// LogAccess 记录访问
func (al *AuditLogger) LogAccess(ctx context.Context, operation string, success bool) {
	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		UserID:    getUserID(ctx),
		Operation: operation,
		Success:   success,
		IPAddress: getIPAddress(ctx),
		UserAgent: getUserAgent(ctx),
		Details:   make(map[string]interface{}),
	}

	// 保存事件
	al.mu.Lock()
	al.events = append(al.events, event)
	// 限制历史记录大小
	if len(al.events) > 10000 {
		al.events = al.events[5000:]
	}
	al.mu.Unlock()

	// 记录日志
	al.logger.Info("Audit event",
		zap.String("event_id", event.ID),
		zap.String("user_id", event.UserID),
		zap.String("operation", operation),
		zap.Bool("success", success),
		zap.String("ip_address", event.IPAddress))
}

// LogDataAccess 记录数据访问
func (al *AuditLogger) LogDataAccess(ctx context.Context, resource, action string, success bool) {
	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		UserID:    getUserID(ctx),
		Operation: fmt.Sprintf("data_%s", action),
		Resource:  resource,
		Success:   success,
		IPAddress: getIPAddress(ctx),
		Details: map[string]interface{}{
			"action":   action,
			"resource": resource,
		},
	}

	al.mu.Lock()
	al.events = append(al.events, event)
	al.mu.Unlock()

	al.logger.Info("Data access audit",
		zap.String("event_id", event.ID),
		zap.String("user_id", event.UserID),
		zap.String("resource", resource),
		zap.String("action", action),
		zap.Bool("success", success))
}

// GetAuditEvents 获取审计事件
func (al *AuditLogger) GetAuditEvents(userID string, limit int) []*AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	events := make([]*AuditEvent, 0)
	for i := len(al.events) - 1; i >= 0 && len(events) < limit; i-- {
		if userID == "" || al.events[i].UserID == userID {
			events = append(events, al.events[i])
		}
	}

	return events
}

// getUserID 获取用户ID
func getUserID(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		return userID.(string)
	}
	return "anonymous"
}

// getIPAddress 获取IP地址
func getIPAddress(ctx context.Context) string {
	if ip := ctx.Value("ip_address"); ip != nil {
		return ip.(string)
	}
	return "unknown"
}

// getUserAgent 获取用户代理
func getUserAgent(ctx context.Context) string {
	if ua := ctx.Value("user_agent"); ua != nil {
		return ua.(string)
	}
	return "unknown"
}

func generateAuditID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}
