package service

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"awesome/internal/evaluation/errors"
	"awesome/internal/evaluation/logger"
	"awesome/internal/evaluation/models"

	"go.uber.org/zap"
)

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger       *logger.Logger
	errorChan    chan *ErrorEvent
	alertManager *AlertManager
	mu           sync.RWMutex
	errorHistory []*ErrorEvent
	maxHistory   int
}

// ErrorEvent 错误事件
type ErrorEvent struct {
	ID         string                 `json:"id"`
	Error      error                  `json:"error"`
	Timestamp  time.Time              `json:"timestamp"`
	Component  string                 `json:"component"`
	Operation  string                 `json:"operation"`
	Context    map[string]interface{} `json:"context"`
	Severity   string                 `json:"severity"`
	Recovered  bool                   `json:"recovered"`
	RetryCount int                    `json:"retry_count"`
	TaskID     string                 `json:"task_id,omitempty"`
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(log *logger.Logger) *ErrorHandler {
	handler := &ErrorHandler{
		logger:       log.WithComponent("error_handler"),
		errorChan:    make(chan *ErrorEvent, 1000),
		alertManager: NewAlertManager(log),
		errorHistory: make([]*ErrorEvent, 0, 1000),
		maxHistory:   1000,
	}

	go handler.processErrors()

	return handler
}

// HandleError 处理错误
func (h *ErrorHandler) HandleError(err error, message string, fields ...zap.Field) *ErrorEvent {
	if err == nil {
		return nil
	}

	event := &ErrorEvent{
		ID:        generateErrorID(),
		Error:     err,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}

	if evalErr, ok := err.(*errors.EvaluationError); ok {
		event.Component = evalErr.Component
		event.Operation = evalErr.Operation
		event.Severity = evalErr.Severity
		event.Context = evalErr.Details
	} else {
		event.Severity = "medium"
	}

	// 将 zap.Field 转换为 map
	for _, f := range fields {
		event.Context[f.Key] = f.Interface
	}

	allFields := append(fields,
		zap.String("error_id", event.ID),
		zap.String("severity", event.Severity),
		zap.String("component", event.Component),
		zap.String("operation", event.Operation),
	)

	switch event.Severity {
	case "critical":
		h.logger.Error(fmt.Sprintf("CRITICAL: %s - %v", message, err), allFields...)
	case "high":
		h.logger.Error(fmt.Sprintf("HIGH: %s - %v", message, err), allFields...)
	case "medium":
		h.logger.Warn(fmt.Sprintf("MEDIUM: %s - %v", message, err), allFields...)
	default:
		h.logger.Info(fmt.Sprintf("LOW: %s - %v", message, err), allFields...)
	}

	select {
	case h.errorChan <- event:
	default:
		h.logger.Warn("Error channel full, dropping error event",
			zap.String("error_id", event.ID))
	}

	h.saveToHistory(event)

	return event
}

// HandlePanic 处理panic
func (h *ErrorHandler) HandlePanic(r interface{}, message string, fields ...zap.Field) {
	var err error
	if e, ok := r.(error); ok {
		err = e
	} else {
		err = fmt.Errorf("panic: %v", r)
	}

	stack := string(debug.Stack())

	allFields := append(fields,
		zap.Error(err),
		zap.String("stack", stack),
	)

	h.logger.Fatal(fmt.Sprintf("PANIC: %s", message), allFields...)

	event := &ErrorEvent{
		ID:        generateErrorID(),
		Error:     err,
		Timestamp: time.Now(),
		Severity:  "critical",
		Context: map[string]interface{}{
			"stack":   stack,
			"message": message,
		},
	}

	h.alertManager.SendCriticalAlert("panic", message, event)
	h.saveToHistory(event)
}

// processErrors 处理错误事件
func (h *ErrorHandler) processErrors() {
	for event := range h.errorChan {
		switch event.Severity {
		case "critical":
			h.handleCriticalError(event)
		case "high":
			h.handleHighError(event)
		case "medium":
			h.handleMediumError(event)
		default:
			h.handleLowError(event)
		}
	}
}

// handleCriticalError 处理严重错误
func (h *ErrorHandler) handleCriticalError(event *ErrorEvent) {
	h.alertManager.SendCriticalAlert(
		event.Component,
		fmt.Sprintf("Critical error in %s: %v", event.Operation, event.Error),
		event,
	)

	h.logger.Error("Critical error occurred",
		zap.String("error_id", event.ID),
		zap.Error(event.Error),
		zap.Any("context", event.Context))
}

// handleHighError 处理高优先级错误
func (h *ErrorHandler) handleHighError(event *ErrorEvent) {
	h.alertManager.SendHighAlert(
		event.Component,
		fmt.Sprintf("High severity error in %s: %v", event.Operation, event.Error),
		event,
	)
}

// handleMediumError 处理中等优先级错误
func (h *ErrorHandler) handleMediumError(event *ErrorEvent) {
	h.alertManager.SendWarning(
		event.Component,
		fmt.Sprintf("Medium severity error in %s: %v", event.Operation, event.Error),
		event,
	)
}

// handleLowError 处理低优先级错误
func (h *ErrorHandler) handleLowError(event *ErrorEvent) {
	h.logger.Info("Low severity error",
		zap.String("error_id", event.ID),
		zap.Error(event.Error))
}

// saveToHistory 保存到历史记录
func (h *ErrorHandler) saveToHistory(event *ErrorEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errorHistory = append(h.errorHistory, event)

	if len(h.errorHistory) > h.maxHistory {
		h.errorHistory = h.errorHistory[len(h.errorHistory)-h.maxHistory:]
	}
}

// GetErrorHistory 获取错误历史
func (h *ErrorHandler) GetErrorHistory(limit int) []*ErrorEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.errorHistory) {
		limit = len(h.errorHistory)
	}

	result := make([]*ErrorEvent, limit)
	copy(result, h.errorHistory[len(h.errorHistory)-limit:])

	return result
}

// GetErrorStatistics 获取错误统计
func (h *ErrorHandler) GetErrorStatistics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_errors": len(h.errorHistory),
		"by_severity":  make(map[string]int),
		"by_component": make(map[string]int),
	}

	severityCount := stats["by_severity"].(map[string]int)
	componentCount := stats["by_component"].(map[string]int)

	for _, event := range h.errorHistory {
		severityCount[event.Severity]++
		if event.Component != "" {
			componentCount[event.Component]++
		}
	}

	return stats
}

// AlertManager 告警管理器
type AlertManager struct {
	logger     *logger.Logger
	alertChan  chan *models.Alert
	alertStore map[string]*models.Alert
	mu         sync.RWMutex
}

// NewAlertManager 创建告警管理器
func NewAlertManager(log *logger.Logger) *AlertManager {
	return &AlertManager{
		logger:     log.WithComponent("alert_manager"),
		alertChan:  make(chan *models.Alert, 1000),
		alertStore: make(map[string]*models.Alert),
	}
}

// SendCriticalAlert 发送严重告警
func (am *AlertManager) SendCriticalAlert(source, message string, event *ErrorEvent) {
	alert := &models.Alert{
		ID:        generateAlertID(),
		Type:      "error",
		Severity:  "critical",
		Title:     fmt.Sprintf("Critical Error in %s", source),
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Details:   event.Context,
	}

	am.sendAlert(alert)
}

// SendHighAlert 发送高优先级告警
func (am *AlertManager) SendHighAlert(source, message string, event *ErrorEvent) {
	alert := &models.Alert{
		ID:        generateAlertID(),
		Type:      "error",
		Severity:  "warning",
		Title:     fmt.Sprintf("High Severity Error in %s", source),
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Details:   event.Context,
	}

	am.sendAlert(alert)
}

// SendWarning 发送警告
func (am *AlertManager) SendWarning(source, message string, event *ErrorEvent) {
	alert := &models.Alert{
		ID:        generateAlertID(),
		Type:      "warning",
		Severity:  "warning",
		Title:     fmt.Sprintf("Warning in %s", source),
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Details:   event.Context,
	}

	am.sendAlert(alert)
}

// sendAlert 发送告警
func (am *AlertManager) sendAlert(alert *models.Alert) {
	am.mu.Lock()
	am.alertStore[alert.ID] = alert
	am.mu.Unlock()

	select {
	case am.alertChan <- alert:
		am.logger.Info("Alert sent",
			zap.String("alert_id", alert.ID),
			zap.String("severity", alert.Severity),
			zap.String("source", alert.Source))
	default:
		am.logger.Warn("Alert channel full, dropping alert",
			zap.String("alert_id", alert.ID))
	}

	am.logger.Warn("Alert generated",
		zap.String("alert_id", alert.ID),
		zap.String("type", alert.Type),
		zap.String("severity", alert.Severity),
		zap.String("title", alert.Title),
		zap.String("message", alert.Message))
}

// GetAlertChannel 获取告警通道
func (am *AlertManager) GetAlertChannel() <-chan *models.Alert {
	return am.alertChan
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManager) GetActiveAlerts() []*models.Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*models.Alert, 0)
	for _, alert := range am.alertStore {
		if !alert.Acknowledged {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// AcknowledgeAlert 确认告警
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alertStore[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	alert.Acknowledged = true
	now := time.Now()
	alert.ResolvedAt = &now

	am.logger.Info("Alert acknowledged",
		zap.String("alert_id", alertID))

	return nil
}

func generateErrorID() string {
	return fmt.Sprintf("err_%d", time.Now().UnixNano())
}

func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
