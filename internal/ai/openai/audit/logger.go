package audit

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

// Logger 审计日志器
type Logger struct {
	logger   *zap.SugaredLogger
	enabled  bool
	sanitize bool
}

// NewLogger 创建审计日志器
func NewLogger(logger *zap.SugaredLogger) *Logger {
	return &Logger{
		logger:   logger,
		enabled:  true,
		sanitize: true,
	}
}

// AuditEntry 审计条目
type AuditEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Sensitive   bool                   `json:"sensitive,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

// LogRequest 记录请求
func (l *Logger) LogRequest(ctx context.Context, req interface{}) {
	if !l.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "request",
		Operation: "chat_completion",
		Status:    "started",
	}

	// 从 context 提取信息
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}
	if ipAddress, ok := ctx.Value("ip_address").(string); ok {
		entry.IPAddress = ipAddress
	}

	// 记录请求详情（脱敏）
	if l.sanitize {
		entry.Metadata = l.sanitizeRequest(req)
	} else {
		entry.Metadata = map[string]interface{}{
			"request": req,
		}
	}

	l.logEntry(entry)
}

// LogResponse 记录响应
func (l *Logger) LogResponse(ctx context.Context, resp interface{}) {
	if !l.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "response",
		Operation: "chat_completion",
		Status:    "completed",
	}

	// 从 context 提取信息
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}

	// 记录响应详情（脱敏）
	if l.sanitize {
		entry.Metadata = l.sanitizeResponse(resp)
	} else {
		entry.Metadata = map[string]interface{}{
			"response": resp,
		}
	}

	l.logEntry(entry)
}

// LogError 记录错误
func (l *Logger) LogError(ctx context.Context, err error) {
	if !l.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "error",
		Operation: "chat_completion",
		Status:    "failed",
		Error:     err.Error(),
	}

	// 从 context 提取信息
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}

	l.logEntry(entry)
}

// LogSecurityEvent 记录安全事件
func (l *Logger) LogSecurityEvent(ctx context.Context, eventType string, details map[string]interface{}) {
	if !l.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "security",
		Operation: eventType,
		Status:    "alert",
		Sensitive: true,
		Metadata:  details,
	}

	// 从 context 提取信息
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}
	if ipAddress, ok := ctx.Value("ip_address").(string); ok {
		entry.IPAddress = ipAddress
	}

	l.logEntry(entry)
}

// LogRateLimit 记录限流事件
func (l *Logger) LogRateLimit(ctx context.Context, limitType string, current, max int) {
	if !l.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "rate_limit",
		Operation: limitType,
		Status:    "throttled",
		Metadata: map[string]interface{}{
			"current": current,
			"max":     max,
		},
	}

	l.logEntry(entry)
}

// logEntry 记录审计条目
func (l *Logger) logEntry(entry AuditEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		l.logger.Errorw("failed to marshal audit entry", "error", err)
		return
	}

	l.logger.Infow("audit_log", "entry", string(data))
}

// sanitizeRequest 脱敏请求
func (l *Logger) sanitizeRequest(req interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})

	// 类型断言获取请求信息
	if chatReq, ok := req.(map[string]interface{}); ok {
		if model, ok := chatReq["model"].(string); ok {
			metadata["model"] = model
		}
		if messages, ok := chatReq["messages"].([]interface{}); ok {
			metadata["message_count"] = len(messages)
		}
		if temp, ok := chatReq["temperature"].(float64); ok {
			metadata["temperature"] = temp
		}
		if maxTokens, ok := chatReq["max_tokens"].(int); ok {
			metadata["max_tokens"] = maxTokens
		}
	}

	return metadata
}

// sanitizeResponse 脱敏响应
func (l *Logger) sanitizeResponse(resp interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})

	// 类型断言获取响应信息
	if chatResp, ok := resp.(map[string]interface{}); ok {
		if model, ok := chatResp["model"].(string); ok {
			metadata["model"] = model
		}
		if usage, ok := chatResp["usage"].(map[string]interface{}); ok {
			metadata["usage"] = usage
		}
		if choices, ok := chatResp["choices"].([]interface{}); ok {
			metadata["choice_count"] = len(choices)
		}
	}

	return metadata
}

// Enable 启用审计
func (l *Logger) Enable() {
	l.enabled = true
}

// Disable 禁用审计
func (l *Logger) Disable() {
	l.enabled = false
}

// SetSanitize 设置脱敏
func (l *Logger) SetSanitize(sanitize bool) {
	l.sanitize = sanitize
}
