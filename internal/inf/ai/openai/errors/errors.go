package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorCode 错误码
type ErrorCode string

const (
	// 通用错误
	ErrInternal         ErrorCode = "internal_error"
	ErrInvalidRequest   ErrorCode = "invalid_request"
	ErrInvalidParameter ErrorCode = "invalid_parameter"
	ErrTimeout          ErrorCode = "timeout"
	ErrCancelled        ErrorCode = "cancelled"

	// 认证错误
	ErrUnauthorized     ErrorCode = "unauthorized"
	ErrInvalidAPIKey    ErrorCode = "invalid_api_key"
	ErrPermissionDenied ErrorCode = "permission_denied"

	// 限流错误
	ErrRateLimitExceeded ErrorCode = "rate_limit_exceeded"
	ErrQuotaExceeded     ErrorCode = "quota_exceeded"

	// 模型错误
	ErrModelNotFound    ErrorCode = "model_not_found"
	ErrModelOverloaded  ErrorCode = "model_overloaded"
	ErrModelUnavailable ErrorCode = "model_unavailable"

	// 内容错误
	ErrContentFiltered  ErrorCode = "content_filtered"
	ErrSensitiveContent ErrorCode = "sensitive_content"
	ErrInvalidContent   ErrorCode = "invalid_content"

	// 网络错误
	ErrConnectionFailed ErrorCode = "connection_failed"
	ErrDNSError         ErrorCode = "dns_error"
	ErrSSLError         ErrorCode = "ssl_error"

	// 服务错误
	ErrServiceUnavailable ErrorCode = "service_unavailable"
	ErrServerError        ErrorCode = "server_error"
)

// OpenAIError OpenAI 错误
type OpenAIError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Type       string                 `json:"type,omitempty"`
	Param      string                 `json:"param,omitempty"`
	HTTPStatus int                    `json:"-"`
	Retryable  bool                   `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Error 实现 error 接口
func (e *OpenAIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// JSON 返回 JSON 格式错误
func (e *OpenAIError) JSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// NewError 创建错误
func NewError(code ErrorCode, message string) *OpenAIError {
	return &OpenAIError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithStatus 创建带状态码的错误
func NewErrorWithStatus(code ErrorCode, message string, status int) *OpenAIError {
	return &OpenAIError{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
	}
}

// NewRetryableError 创建可重试错误
func NewRetryableError(code ErrorCode, message string) *OpenAIError {
	return &OpenAIError{
		Code:      code,
		Message:   message,
		Retryable: true,
	}
}

// IsRetryable 检查是否可重试
func IsRetryable(err error) bool {
	if oaiErr, ok := err.(*OpenAIError); ok {
		return oaiErr.Retryable
	}
	return false
}

// IsTimeout 检查是否超时
func IsTimeout(err error) bool {
	if oaiErr, ok := err.(*OpenAIError); ok {
		return oaiErr.Code == ErrTimeout
	}
	return false
}

// IsRateLimit 检查是否限流
func IsRateLimit(err error) bool {
	if oaiErr, ok := err.(*OpenAIError); ok {
		return oaiErr.Code == ErrRateLimitExceeded || oaiErr.Code == ErrQuotaExceeded
	}
	return false
}

// ParseAPIError 解析 API 错误
func ParseAPIError(statusCode int, body []byte) *OpenAIError {
	var apiErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Type    string `json:"type"`
			Param   string `json:"param"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &OpenAIError{
			Code:       ErrServerError,
			Message:    string(body),
			HTTPStatus: statusCode,
		}
	}

	code := ErrorCode(apiErr.Error.Code)
	if code == "" {
		code = mapHTTPStatusToCode(statusCode)
	}

	return &OpenAIError{
		Code:       code,
		Message:    apiErr.Error.Message,
		Type:       apiErr.Error.Type,
		Param:      apiErr.Error.Param,
		HTTPStatus: statusCode,
		Retryable:  isRetryableStatus(statusCode),
	}
}

// mapHTTPStatusToCode 映射 HTTP 状态码到错误码
func mapHTTPStatusToCode(status int) ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return ErrInvalidRequest
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrPermissionDenied
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimitExceeded
	case http.StatusInternalServerError:
		return ErrServerError
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return ErrInternal
	}
}

// isRetryableStatus 检查状态码是否可重试
func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusInternalServerError ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, message string) *OpenAIError {
	if err == nil {
		return nil
	}

	if oaiErr, ok := err.(*OpenAIError); ok {
		return oaiErr
	}

	return &OpenAIError{
		Code:    code,
		Message: fmt.Sprintf("%s: %v", message, err),
		Details: map[string]interface{}{
			"original_error": err.Error(),
		},
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors 验证错误集合
type ValidationErrors []*ValidationError

// Error 实现 error 接口
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	return fmt.Sprintf("validation failed: %s - %s", ve[0].Field, ve[0].Message)
}

// Add 添加验证错误
func (ve *ValidationErrors) Add(field, message string) {
	*ve = append(*ve, &ValidationError{Field: field, Message: message})
}

// HasErrors 检查是否有错误
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}
