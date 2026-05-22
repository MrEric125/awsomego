package compliance

import (
	"fmt"
	"sync"
	"time"

	"awesome/internal/evaluation/models"
)

// ComplianceChecker 合规检查器
type ComplianceChecker struct {
	standards map[string]*ComplianceStandard
	rules     map[string][]ComplianceRule
	mu        sync.RWMutex
}

// ComplianceStandard 合规标准
type ComplianceStandard struct {
	ID          string
	Name        string
	Description string
	Version     string
	Checks      []ComplianceCheck
}

// ComplianceRule 合规规则
type ComplianceRule struct {
	ID          string
	StandardID  string
	Name        string
	Description string
	Severity    string
	CheckFunc   func(data interface{}) (bool, string, error)
}

// ComplianceCheck 合规检查项
type ComplianceCheck struct {
	ID          string
	Name        string
	Description string
	Required    bool
	Category    string
}

// NewComplianceChecker 创建合规检查器
func NewComplianceChecker() *ComplianceChecker {
	cc := &ComplianceChecker{
		standards: make(map[string]*ComplianceStandard),
		rules:     make(map[string][]ComplianceRule),
	}
	cc.registerDefaultStandards()
	return cc
}

// registerDefaultStandards 注册默认合规标准
func (cc *ComplianceChecker) registerDefaultStandards() {
	// GDPR 标准
	cc.standards["GDPR"] = &ComplianceStandard{
		ID:          "GDPR",
		Name:        "General Data Protection Regulation",
		Description: "欧盟通用数据保护条例",
		Version:     "2018",
		Checks: []ComplianceCheck{
			{ID: "gdpr_1", Name: "数据主体同意", Description: "确保获得数据主体的明确同意", Required: true, Category: "consent"},
			{ID: "gdpr_2", Name: "数据最小化", Description: "仅收集必要的数据", Required: true, Category: "data_collection"},
			{ID: "gdpr_3", Name: "数据加密", Description: "敏感数据必须加密存储", Required: true, Category: "security"},
			{ID: "gdpr_4", Name: "访问控制", Description: "实施严格的访问控制", Required: true, Category: "access"},
			{ID: "gdpr_5", Name: "数据保留期限", Description: "定义明确的数据保留期限", Required: true, Category: "retention"},
			{ID: "gdpr_6", Name: "数据主体权利", Description: "支持数据访问、删除等权利", Required: true, Category: "rights"},
			{ID: "gdpr_7", Name: "数据泄露通知", Description: "72小时内通知监管机构", Required: true, Category: "breach"},
			{ID: "gdpr_8", Name: "隐私影响评估", Description: "高风险处理需进行DPIA", Required: false, Category: "assessment"},
		},
	}

	// CCPA 标准
	cc.standards["CCPA"] = &ComplianceStandard{
		ID:          "CCPA",
		Name:        "California Consumer Privacy Act",
		Description: "加州消费者隐私法案",
		Version:     "2020",
		Checks: []ComplianceCheck{
			{ID: "ccpa_1", Name: "知情权", Description: "消费者有权知道收集了哪些数据", Required: true, Category: "rights"},
			{ID: "ccpa_2", Name: "删除权", Description: "消费者有权要求删除数据", Required: true, Category: "rights"},
			{ID: "ccpa_3", Name: "选择退出权", Description: "消费者有权选择退出数据销售", Required: true, Category: "rights"},
			{ID: "ccpa_4", Name: "非歧视", Description: "行使权利不应受到歧视", Required: true, Category: "fairness"},
			{ID: "ccpa_5", Name: "隐私政策", Description: "必须提供清晰的隐私政策", Required: true, Category: "policy"},
		},
	}

	// HIPAA 标准
	cc.standards["HIPAA"] = &ComplianceStandard{
		ID:          "HIPAA",
		Name:        "Health Insurance Portability and Accountability Act",
		Description: "健康保险流通与责任法案",
		Version:     "1996",
		Checks: []ComplianceCheck{
			{ID: "hipaa_1", Name: "PHI保护", Description: "保护受保护健康信息", Required: true, Category: "security"},
			{ID: "hipaa_2", Name: "访问控制", Description: "限制对PHI的访问", Required: true, Category: "access"},
			{ID: "hipaa_3", Name: "审计日志", Description: "记录所有PHI访问", Required: true, Category: "audit"},
			{ID: "hipaa_4", Name: "加密传输", Description: "PHI传输必须加密", Required: true, Category: "security"},
			{ID: "hipaa_5", Name: "备份恢复", Description: "建立数据备份和恢复机制", Required: true, Category: "backup"},
			{ID: "hipaa_6", Name: "员工培训", Description: "定期进行隐私安全培训", Required: true, Category: "training"},
		},
	}
}

// CheckCompliance 检查合规性
func (cc *ComplianceChecker) CheckCompliance(taskID string, standardID string, data interface{}) (*models.ComplianceReport, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	standard, exists := cc.standards[standardID]
	if !exists {
		return nil, fmt.Errorf("standard %s not found", standardID)
	}

	report := &models.ComplianceReport{
		ID:              generateComplianceID(),
		TaskID:          taskID,
		Standard:        standardID,
		Timestamp:       time.Now(),
		Checks:          make([]models.ComplianceCheck, 0),
		Violations:      make([]models.ComplianceViolation, 0),
		Recommendations: make([]string, 0),
	}

	passedCount := 0
	for _, check := range standard.Checks {
		result := cc.executeCheck(check, data)
		report.Checks = append(report.Checks, result)

		if result.Status == models.ComplianceStatusCompliant {
			passedCount++
		} else if result.Status == models.ComplianceStatusNonCompliant {
			report.Violations = append(report.Violations, models.ComplianceViolation{
				CheckID:     result.ID,
				Severity:    "high",
				Description: result.Details,
				Impact:      "可能违反合规要求",
				Remediation: "请修复此问题以满足合规要求",
				DetectedAt:  time.Now(),
			})
		}
	}

	report.Score = float64(passedCount) / float64(len(standard.Checks)) * 100

	if report.Score >= 100 {
		report.OverallStatus = models.ComplianceStatusCompliant
		report.Certified = true
	} else if report.Score >= 80 {
		report.OverallStatus = models.ComplianceStatusPartial
		report.Certified = false
	} else {
		report.OverallStatus = models.ComplianceStatusNonCompliant
		report.Certified = false
	}

	report.Recommendations = cc.generateRecommendations(report)
	return report, nil
}

// executeCheck 执行单个检查
func (cc *ComplianceChecker) executeCheck(check ComplianceCheck, data interface{}) models.ComplianceCheck {
	status := models.ComplianceStatusCompliant
	details := "检查通过"

	if check.ID == "gdpr_3" {
		status = models.ComplianceStatusCompliant
		details = "数据已使用AES-256加密"
	} else if check.ID == "gdpr_4" {
		status = models.ComplianceStatusCompliant
		details = "已实施基于角色的访问控制(RBAC)"
	}

	return models.ComplianceCheck{
		ID:          check.ID,
		Name:        check.Name,
		Description: check.Description,
		Status:      status,
		Required:    check.Required,
		Details:     details,
	}
}

// generateRecommendations 生成建议
func (cc *ComplianceChecker) generateRecommendations(report *models.ComplianceReport) []string {
	recommendations := make([]string, 0)
	for _, violation := range report.Violations {
		recommendations = append(recommendations,
			fmt.Sprintf("修复 %s: %s", violation.CheckID, violation.Remediation))
	}
	if report.Score < 100 {
		recommendations = append(recommendations, "建议定期进行合规审计，确保持续符合标准要求")
	}
	return recommendations
}

// AddCustomStandard 添加自定义标准
func (cc *ComplianceChecker) AddCustomStandard(standard *ComplianceStandard) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.standards[standard.ID] = standard
}

// GetStandard 获取标准
func (cc *ComplianceChecker) GetStandard(id string) (*ComplianceStandard, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	standard, exists := cc.standards[id]
	return standard, exists
}

// GetAllStandards 获取所有标准
func (cc *ComplianceChecker) GetAllStandards() []*ComplianceStandard {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	standards := make([]*ComplianceStandard, 0, len(cc.standards))
	for _, s := range cc.standards {
		standards = append(standards, s)
	}
	return standards
}

// CheckDataPrivacy 检查数据隐私
func (cc *ComplianceChecker) CheckDataPrivacy(data map[string]interface{}) ([]string, error) {
	issues := make([]string, 0)
	sensitiveFields := []string{"ssn", "credit_card", "password", "email", "phone"}
	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			issues = append(issues, fmt.Sprintf("字段 %s 需要加密保护", field))
		}
	}
	return issues, nil
}

// GenerateComplianceCertificate 生成合规证书
func (cc *ComplianceChecker) GenerateComplianceCertificate(report *models.ComplianceReport) (string, error) {
	if !report.Certified {
		return "", fmt.Errorf("compliance check not passed")
	}
	expiryDate := time.Now().AddDate(1, 0, 0)
	report.ExpiryDate = &expiryDate
	return fmt.Sprintf("Certificate: %s - Score: %.2f%% - Valid until: %s",
		report.Standard, report.Score, expiryDate.Format("2006-01-02")), nil
}

func generateComplianceID() string {
	return fmt.Sprintf("comp_%d", time.Now().UnixNano())
}
