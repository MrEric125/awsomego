package security

import (
	"regexp"
	"strings"
	"sync"
)

// Filter 安全过滤器
type Filter struct {
	patterns     []*regexp.Regexp
	sensitiveWords []string
	replacement  string
	mu           sync.RWMutex
}

// FilterOption 过滤器选项
type FilterOption func(*Filter)

// NewFilter 创建安全过滤器
func NewFilter(opts ...FilterOption) *Filter {
	f := &Filter{
		replacement: "[FILTERED]",
		patterns: []*regexp.Regexp{
			// API 密钥模式
			regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)\s*[=:]\s*['"]?[a-zA-Z0-9_-]{20,}['"]?`),
			// 密码模式
			regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*['"]?[^\s'"]{8,}['"]?`),
			// 令牌模式
			regexp.MustCompile(`(?i)(token|bearer|jwt)\s*[=:]\s*['"]?[a-zA-Z0-9_.-]{20,}['"]?`),
			// 私钥模式
			regexp.MustCompile(`(?i)-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----`),
			// 信用卡号模式
			regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
			// SSN 模式
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
			// 邮箱模式
			regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
			// IP 地址模式
			regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
			// 手机号模式（中国）
			regexp.MustCompile(`\b1[3-9]\d{9}\b`),
			// 身份证号模式（中国）
			regexp.MustCompile(`\b\d{17}[\dXx]\b`),
		},
		sensitiveWords: []string{
			"password", "secret", "token", "key", "credential",
			"private", "confidential", "sensitive",
		},
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithReplacement 设置替换字符串
func WithReplacement(replacement string) FilterOption {
	return func(f *Filter) {
		f.replacement = replacement
	}
}

// WithPatterns 设置自定义模式
func WithPatterns(patterns []string) FilterOption {
	return func(f *Filter) {
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				f.patterns = append(f.patterns, re)
			}
		}
	}
}

// WithSensitiveWords 设置敏感词
func WithSensitiveWords(words []string) FilterOption {
	return func(f *Filter) {
		f.sensitiveWords = append(f.sensitiveWords, words...)
	}
}

// Filter 过滤敏感内容
func (f *Filter) Filter(content string) (string, []string) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var warnings []string
	result := content

	// 应用正则模式
	for _, pattern := range f.patterns {
		matches := pattern.FindAllString(result, -1)
		if len(matches) > 0 {
			warnings = append(warnings, "sensitive pattern detected: "+pattern.String())
			result = pattern.ReplaceAllString(result, f.replacement)
		}
	}

	// 检查敏感词
	for _, word := range f.sensitiveWords {
		if strings.Contains(strings.ToLower(result), strings.ToLower(word)) {
			warnings = append(warnings, "sensitive word detected: "+word)
		}
	}

	return result, warnings
}

// AddPattern 添加过滤模式
func (f *Filter) AddPattern(pattern string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	f.patterns = append(f.patterns, re)
	return nil
}

// AddSensitiveWord 添加敏感词
func (f *Filter) AddSensitiveWord(word string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sensitiveWords = append(f.sensitiveWords, word)
}

// Validate 验证内容是否安全
func (f *Filter) Validate(content string) (bool, []string) {
	_, warnings := f.Filter(content)
	return len(warnings) == 0, warnings
}

// Sanitize 清理内容
func (f *Filter) Sanitize(content string) string {
	filtered, _ := f.Filter(content)
	return filtered
}

// MaskEmail 掩码邮箱
func (f *Filter) MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	name := parts[0]
	domain := parts[1]

	if len(name) <= 2 {
		return name[:1] + "***@" + domain
	}

	return name[:2] + "***@" + domain
}

// MaskPhone 掩码手机号
func (f *Filter) MaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskIDCard 掩码身份证号
func (f *Filter) MaskIDCard(idCard string) string {
	if len(idCard) < 8 {
		return idCard
	}
	return idCard[:4] + "**********" + idCard[len(idCard)-4:]
}
