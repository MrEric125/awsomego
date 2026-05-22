package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"awesome/internal/evaluation/errors"
)

// Validator 验证器
type Validator struct {
	errors []*ValidationError
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{
		errors: make([]*ValidationError, 0),
	}
}

// AddError 添加错误
func (v *Validator) AddError(field, message string, value interface{}) {
	v.errors = append(v.errors, &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors 检查是否有错误
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors 获取所有错误
func (v *Validator) GetErrors() []*ValidationError {
	return v.errors
}

// GetError 获取第一个错误
func (v *Validator) GetError() *errors.EvaluationError {
	if !v.HasErrors() {
		return nil
	}
	
	firstErr := v.errors[0]
	return errors.ErrInvalidParam(firstErr.Field, firstErr.Message).
		WithDetail("validation_errors", v.errors)
}

// Required 必填验证
func (v *Validator) Required(field string, value interface{}) *Validator {
	if value == nil || value == "" {
		v.AddError(field, "is required", value)
	}
	return v
}

// StringLength 字符串长度验证
func (v *Validator) StringLength(field string, value string, min, max int) *Validator {
	length := len(value)
	if length < min || length > max {
		v.AddError(field, fmt.Sprintf("length must be between %d and %d", min, max), value)
	}
	return v
}

// MinLength 最小长度验证
func (v *Validator) MinLength(field string, value string, min int) *Validator {
	if len(value) < min {
		v.AddError(field, fmt.Sprintf("length must be at least %d", min), value)
	}
	return v
}

// MaxLength 最大长度验证
func (v *Validator) MaxLength(field string, value string, max int) *Validator {
	if len(value) > max {
		v.AddError(field, fmt.Sprintf("length must be at most %d", max), value)
	}
	return v
}

// Range 数值范围验证
func (v *Validator) Range(field string, value, min, max float64) *Validator {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %.2f and %.2f", min, max), value)
	}
	return v
}

// Min 最小值验证
func (v *Validator) Min(field string, value, min float64) *Validator {
	if value < min {
		v.AddError(field, fmt.Sprintf("must be at least %.2f", min), value)
	}
	return v
}

// Max 最大值验证
func (v *Validator) Max(field string, value, max float64) *Validator {
	if value > max {
		v.AddError(field, fmt.Sprintf("must be at most %.2f", max), value)
	}
	return v
}

// InList 列表包含验证
func (v *Validator) InList(field string, value interface{}, list []interface{}) *Validator {
	found := false
	for _, item := range list {
		if item == value {
			found = true
			break
		}
	}
	if !found {
		v.AddError(field, fmt.Sprintf("must be one of: %v", list), value)
	}
	return v
}

// InStringList 字符串列表包含验证
func (v *Validator) InStringList(field string, value string, list []string) *Validator {
	found := false
	for _, item := range list {
		if item == value {
			found = true
			break
		}
	}
	if !found {
		v.AddError(field, fmt.Sprintf("must be one of: %v", list), value)
	}
	return v
}

// Regex 正则表达式验证
func (v *Validator) Regex(field string, value string, pattern string) *Validator {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		v.AddError(field, fmt.Sprintf("invalid regex pattern: %s", pattern), value)
		return v
	}
	if !matched {
		v.AddError(field, fmt.Sprintf("does not match pattern: %s", pattern), value)
	}
	return v
}

// Email 邮箱验证
func (v *Validator) Email(field string, value string) *Validator {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	return v.Regex(field, value, emailRegex)
}

// URL URL验证
func (v *Validator) URL(field string, value string) *Validator {
	urlRegex := `^https?://[^\s/$.?#].[^\s]*$`
	return v.Regex(field, value, urlRegex)
}

// Alphanumeric 字母数字验证
func (v *Validator) Alphanumeric(field string, value string) *Validator {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			v.AddError(field, "must contain only letters and digits", value)
			return v
		}
	}
	return v
}

// NoSpecialChars 无特殊字符验证
func (v *Validator) NoSpecialChars(field string, value string) *Validator {
	specialChars := `<>:"/\|?*`
	if strings.ContainsAny(value, specialChars) {
		v.AddError(field, "must not contain special characters", value)
	}
	return v
}

// Positive 正数验证
func (v *Validator) Positive(field string, value float64) *Validator {
	if value <= 0 {
		v.AddError(field, "must be positive", value)
	}
	return v
}

// NonNegative 非负数验证
func (v *Validator) NonNegative(field string, value float64) *Validator {
	if value < 0 {
		v.AddError(field, "must be non-negative", value)
	}
	return v
}

// Percentage 百分比验证 (0-100)
func (v *Validator) Percentage(field string, value float64) *Validator {
	return v.Range(field, value, 0, 100)
}

// Duration 时长验证
func (v *Validator) Duration(field string, value time.Duration, min, max time.Duration) *Validator {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %v and %v", min, max), value)
	}
	return v
}

// FutureTime 未来时间验证
func (v *Validator) FutureTime(field string, value time.Time) *Validator {
	if value.Before(time.Now()) {
		v.AddError(field, "must be a future time", value)
	}
	return v
}

// PastTime 过去时间验证
func (v *Validator) PastTime(field string, value time.Time) *Validator {
	if value.After(time.Now()) {
		v.AddError(field, "must be a past time", value)
	}
	return v
}

// Custom 自定义验证
func (v *Validator) Custom(field string, value interface{}, validateFunc func(interface{}) bool, message string) *Validator {
	if !validateFunc(value) {
		v.AddError(field, message, value)
	}
	return v
}

// ValidateTaskID 验证任务ID
func ValidateTaskID(taskID string) error {
	v := NewValidator()
	v.Required("task_id", taskID).
		MinLength("task_id", taskID, 1).
		MaxLength("task_id", taskID, 100).
		Regex("task_id", taskID, `^task_[a-zA-Z0-9_-]+$`)
	
	return v.GetError()
}

// ValidateModelID 验证模型ID
func ValidateModelID(modelID string) error {
	v := NewValidator()
	v.Required("model_id", modelID).
		MinLength("model_id", modelID, 1).
		MaxLength("model_id", modelID, 100).
		Alphanumeric("model_id", modelID)
	
	return v.GetError()
}

// ValidatePriority 验证优先级
func ValidatePriority(priority int) error {
	v := NewValidator()
	v.Range("priority", float64(priority), 0, 100)
	return v.GetError()
}

// ValidateScore 验证分数
func ValidateScore(score float64) error {
	v := NewValidator()
	v.Range("score", score, 0, 100)
	return v.GetError()
}

// ValidateThreshold 验证阈值
func ValidateThreshold(threshold float64) error {
	v := NewValidator()
	v.Range("threshold", threshold, 0, 1)
	return v.GetError()
}

// ValidateWeight 验证权重
func ValidateWeight(weight float64) error {
	v := NewValidator()
	v.Range("weight", weight, 0, 10)
	return v.GetError()
}

// ValidateTimeout 验证超时时间
func ValidateTimeout(timeout time.Duration) error {
	v := NewValidator()
	v.Duration("timeout", timeout, 1*time.Second, 24*time.Hour)
	return v.GetError()
}

// ValidateComplianceStandard 验证合规标准
func ValidateComplianceStandard(standard string) error {
	validStandards := []string{"GDPR", "CCPA", "HIPAA", "ISO27001", "SOC2", "PCI-DSS"}
	v := NewValidator()
	v.Required("standard", standard).
		InStringList("standard", standard, validStandards)
	return v.GetError()
}

// ValidateConfig 验证配置
func ValidateConfig(config map[string]interface{}) error {
	v := NewValidator()
	
	if config == nil {
		v.AddError("config", "cannot be nil", nil)
		return v.GetError()
	}
	
	// 检查配置大小
	if len(config) > 100 {
		v.AddError("config", "too many configuration items", len(config))
	}
	
	// 检查配置键名
	for key := range config {
		if len(key) > 50 {
			v.AddError("config_key", "key name too long", key)
		}
		if strings.Contains(key, "..") || strings.Contains(key, "/") {
			v.AddError("config_key", "invalid characters in key", key)
		}
	}
	
	return v.GetError()
}

// SanitizeString 清理字符串
func SanitizeString(input string) string {
	// 移除前后空格
	input = strings.TrimSpace(input)
	
	// 移除控制字符
	var result strings.Builder
	for _, r := range input {
		if !unicode.IsControl(r) || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// SanitizeMap 清理map
func SanitizeMap(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range input {
		// 清理键名
		key := SanitizeString(k)
		if key == "" {
			continue
		}
		
		// 清理值
		switch val := v.(type) {
		case string:
			result[key] = SanitizeString(val)
		case map[string]interface{}:
			result[key] = SanitizeMap(val)
		default:
			result[key] = v
		}
	}
	return result
}
