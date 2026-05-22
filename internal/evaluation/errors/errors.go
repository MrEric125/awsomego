package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorCode 错误码类型
type ErrorCode string

const (
	// 通用错误码
	ErrUnknown          ErrorCode = "UNKNOWN_ERROR"
	ErrInvalidParameter ErrorCode = "INVALID_PARAMETER"
	ErrNotFound         ErrorCode = "NOT_FOUND"
	ErrAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrTimeout          ErrorCode = "TIMEOUT"
	ErrCanceled         ErrorCode = "CANCELED"

	// 评测相关错误码
	ErrTaskNotFound      ErrorCode = "TASK_NOT_FOUND"
	ErrTaskAlreadyExists ErrorCode = "TASK_ALREADY_EXISTS"
	ErrTaskNotPending    ErrorCode = "TASK_NOT_PENDING"
	ErrTaskNotRunning    ErrorCode = "TASK_NOT_RUNNING"
	ErrTaskFailed        ErrorCode = "TASK_FAILED"
	ErrResultNotFound    ErrorCode = "RESULT_NOT_FOUND"
	ErrReportGeneration  ErrorCode = "REPORT_GENERATION_FAILED"
	ErrMetricCalculation ErrorCode = "METRIC_CALCULATION_FAILED"

	// 性能监控错误码
	ErrMonitorStart      ErrorCode = "MONITOR_START_FAILED"
	ErrMonitorStop       ErrorCode = "MONITOR_STOP_FAILED"
	ErrResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrMemoryLeak        ErrorCode = "MEMORY_LEAK_DETECTED"

	// 合规检查错误码
	ErrComplianceCheck    ErrorCode = "COMPLIANCE_CHECK_FAILED"
	ErrStandardNotFound   ErrorCode = "STANDARD_NOT_FOUND"
	ErrViolationDetected  ErrorCode = "VIOLATION_DETECTED"
	ErrCertificationFailed ErrorCode = "CERTIFICATION_FAILED"

	// 调度相关错误码
	ErrSchedulerStart    ErrorCode = "SCHEDULER_START_FAILED"
	ErrSchedulerStop     ErrorCode = "SCHEDULER_STOP_FAILED"
	ErrPipelineNotFound  ErrorCode = "PIPELINE_NOT_FOUND"
	ErrPipelineFailed    ErrorCode = "PIPELINE_FAILED"
	ErrTriggerFailed     ErrorCode = "TRIGGER_FAILED"

	// 数据持久化错误码
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION_FAILED"
	ErrDatabaseQuery      ErrorCode = "DATABASE_QUERY_FAILED"
	ErrDatabaseInsert     ErrorCode = "DATABASE_INSERT_FAILED"
	ErrDatabaseUpdate     ErrorCode = "DATABASE_UPDATE_FAILED"
	ErrDatabaseDelete     ErrorCode = "DATABASE_DELETE_FAILED"

	// 配置相关错误码
	ErrConfigLoad    ErrorCode = "CONFIG_LOAD_FAILED"
	ErrConfigParse   ErrorCode = "CONFIG_PARSE_FAILED"
	ErrConfigValidate ErrorCode = "CONFIG_VALIDATE_FAILED"

	// 安全相关错误码
	ErrAuthentication ErrorCode = "AUTHENTICATION_FAILED"
	ErrAuthorization  ErrorCode = "AUTHORIZATION_FAILED"
	ErrTokenExpired   ErrorCode = "TOKEN_EXPIRED"
	ErrRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// EvaluationError 评测系统错误
type EvaluationError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Stack      string                 `json:"stack,omitempty"`
	Cause      error                  `json:"cause,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Retryable  bool                   `json:"retryable"`
	Severity   string                 `json:"severity"` // critical, high, medium, low
	Component  string                 `json:"component"`
	Operation  string                 `json:"operation"`
}

// Error 实现error接口
func (e *EvaluationError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.Message))
	
	if e.Component != "" {
		sb.WriteString(fmt.Sprintf(" (component: %s", e.Component))
		if e.Operation != "" {
			sb.WriteString(fmt.Sprintf(", operation: %s", e.Operation))
		}
		sb.WriteString(")")
	}
	
	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.Cause))
	}
	
	if len(e.Details) > 0 {
		sb.WriteString(fmt.Sprintf(" details: %v", e.Details))
	}
	
	return sb.String()
}

// Unwrap 实现errors.Unwrap
func (e *EvaluationError) Unwrap() error {
	return e.Cause
}

// Is 实现errors.Is
func (e *EvaluationError) Is(target error) bool {
	t, ok := target.(*EvaluationError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewError 创建新错误
func NewError(code ErrorCode, message string) *EvaluationError {
	return &EvaluationError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Stack:     getStackTrace(),
		Details:   make(map[string]interface{}),
	}
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, message string) *EvaluationError {
	if err == nil {
		return nil
	}
	
	if evalErr, ok := err.(*EvaluationError); ok {
		return &EvaluationError{
			Code:      code,
			Message:   message,
			Cause:     evalErr,
			Timestamp: time.Now(),
			Stack:     getStackTrace(),
			Details:   evalErr.Details,
			Retryable: evalErr.Retryable,
			Severity:  evalErr.Severity,
		}
	}
	
	return &EvaluationError{
		Code:      code,
		Message:   message,
		Cause:     err,
		Timestamp: time.Now(),
		Stack:     getStackTrace(),
		Details:   make(map[string]interface{}),
	}
}

// WithDetails 添加详细信息
func (e *EvaluationError) WithDetails(details map[string]interface{}) *EvaluationError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithDetail 添加单个详细信息
func (e *EvaluationError) WithDetail(key string, value interface{}) *EvaluationError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRetryable 设置是否可重试
func (e *EvaluationError) WithRetryable(retryable bool) *EvaluationError {
	e.Retryable = retryable
	return e
}

// WithSeverity 设置严重程度
func (e *EvaluationError) WithSeverity(severity string) *EvaluationError {
	e.Severity = severity
	return e
}

// WithComponent 设置组件
func (e *EvaluationError) WithComponent(component string) *EvaluationError {
	e.Component = component
	return e
}

// WithOperation 设置操作
func (e *EvaluationError) WithOperation(operation string) *EvaluationError {
	e.Operation = operation
	return e
}

// getStackTrace 获取调用栈
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	
	var sb strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		sb.WriteString(fmt.Sprintf("\n\t%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	
	return sb.String()
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if evalErr, ok := err.(*EvaluationError); ok {
		return evalErr.Retryable
	}
	return false
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if evalErr, ok := err.(*EvaluationError); ok {
		return evalErr.Code
	}
	return ErrUnknown
}

// GetErrorSeverity 获取错误严重程度
func GetErrorSeverity(err error) string {
	if evalErr, ok := err.(*EvaluationError); ok {
		return evalErr.Severity
	}
	return "medium"
}

// 预定义错误构造函数

// ErrInvalidParam 参数无效错误
func ErrInvalidParam(param string, reason string) *EvaluationError {
	return NewError(ErrInvalidParameter, fmt.Sprintf("invalid parameter '%s': %s", param, reason)).
		WithSeverity("high").
		WithRetryable(false)
}

// ErrTaskNotFound 任务未找到错误
func ErrTaskNotFoundWithID(taskID string) *EvaluationError {
	return NewError(ErrTaskNotFound, fmt.Sprintf("task '%s' not found", taskID)).
		WithSeverity("medium").
		WithRetryable(false).
		WithDetail("task_id", taskID)
}

// ErrResourceExhausted 资源耗尽错误
func ErrResourceExhaustedWithDetails(resource string, current, limit float64) *EvaluationError {
	return NewError(ErrResourceExhausted, fmt.Sprintf("%s usage %.2f%% exceeds limit %.2f%%", resource, current, limit)).
		WithSeverity("critical").
		WithRetryable(true).
		WithDetails(map[string]interface{}{
			"resource": resource,
			"current":  current,
			"limit":    limit,
		})
}

// ErrDatabaseError 数据库错误
func ErrDatabaseErrorWithCause(operation string, cause error) *EvaluationError {
	code := ErrDatabaseQuery
	switch operation {
	case "insert":
		code = ErrDatabaseInsert
	case "update":
		code = ErrDatabaseUpdate
	case "delete":
		code = ErrDatabaseDelete
	}
	
	return Wrap(cause, code, fmt.Sprintf("database %s failed", operation)).
		WithSeverity("high").
		WithRetryable(true).
		WithComponent("database").
		WithOperation(operation)
}

// ErrTimeout 超时错误
func ErrTimeoutWithDetails(operation string, duration time.Duration) *EvaluationError {
	return NewError(ErrTimeout, fmt.Sprintf("operation '%s' timed out after %v", operation, duration)).
		WithSeverity("high").
		WithRetryable(true).
		WithDetail("operation", operation).
		WithDetail("duration", duration.String())
}

// ErrComplianceViolation 合规违规错误
func ErrComplianceViolationWithDetails(standard, checkID, description string) *EvaluationError {
	return NewError(ErrViolationDetected, description).
		WithSeverity("critical").
		WithRetryable(false).
		WithDetails(map[string]interface{}{
			"standard": standard,
			"check_id": checkID,
		})
}
