package checker

import (
	"fmt"
	"strings"
	"time"

	"github.com/dormoron/mist/security"
)

// CheckSeverity 检查严重性级别
type CheckSeverity int

const (
	SeverityInfo CheckSeverity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String returns string representation of severity
func (s CheckSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "信息"
	case SeverityLow:
		return "低"
	case SeverityMedium:
		return "中"
	case SeverityHigh:
		return "高"
	case SeverityCritical:
		return "严重"
	default:
		return "未知"
	}
}

// CheckResult 检查结果
type CheckResult struct {
	Name        string        // 检查名称
	Description string        // 检查描述
	Severity    CheckSeverity // 严重性级别
	Passed      bool          // 是否通过
	Details     string        // 详细信息
	Suggestion  string        // 建议
}

// SecurityChecker 安全检查器
type SecurityChecker struct {
	config security.SecurityConfig
	checks []func() CheckResult
}

// NewSecurityChecker 创建新的安全检查器
func NewSecurityChecker() *SecurityChecker {
	checker := &SecurityChecker{
		config: security.GetSecurityConfig(),
	}

	// 注册所有检查项
	checker.registerChecks()

	return checker
}

// 注册所有检查
func (c *SecurityChecker) registerChecks() {
	c.checks = []func() CheckResult{
		c.checkSecurityLevel,
		c.checkSessionSecure,
		c.checkCSRFProtection,
		c.checkHSTS,
		c.checkXSSProtection,
		c.checkContentTypeNoSniff,
		c.checkXFrameOptions,
		c.checkCSP,
		c.checkRateLimit,
		c.checkPasswordPolicy,
	}
}

// RunChecks 运行所有安全检查
func (c *SecurityChecker) RunChecks() []CheckResult {
	results := make([]CheckResult, 0, len(c.checks))

	for _, check := range c.checks {
		results = append(results, check())
	}

	return results
}

// RunCheck 运行特定名称的检查
func (c *SecurityChecker) RunCheck(name string) (CheckResult, bool) {
	for _, check := range c.checks {
		result := check()
		if strings.EqualFold(result.Name, name) {
			return result, true
		}
	}
	return CheckResult{}, false
}

// GetScore 获取安全评分（0-100）
func (c *SecurityChecker) GetScore() int {
	results := c.RunChecks()
	if len(results) == 0 {
		return 0
	}

	totalWeight := 0
	weightedScore := 0

	for _, result := range results {
		weight := getCheckWeight(result.Severity)
		totalWeight += weight

		if result.Passed {
			weightedScore += weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return (weightedScore * 100) / totalWeight
}

// GetLevel 获取安全级别描述
func (c *SecurityChecker) GetLevel() string {
	score := c.GetScore()

	if score >= 90 {
		return "优秀"
	} else if score >= 80 {
		return "良好"
	} else if score >= 70 {
		return "合格"
	} else if score >= 50 {
		return "需要改进"
	} else {
		return "不安全"
	}
}

// GetSummary 获取安全检查摘要
func (c *SecurityChecker) GetSummary() string {
	results := c.RunChecks()
	score := c.GetScore()
	level := c.GetLevel()

	passCount := 0
	failCount := 0
	var criticalIssues, highIssues, mediumIssues []string

	for _, result := range results {
		if result.Passed {
			passCount++
		} else {
			failCount++
			detail := result.Name
			switch result.Severity {
			case SeverityCritical:
				criticalIssues = append(criticalIssues, detail)
			case SeverityHigh:
				highIssues = append(highIssues, detail)
			case SeverityMedium:
				mediumIssues = append(mediumIssues, detail)
			}
		}
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("安全评分: %d/100 (%s)\n", score, level))
	summary.WriteString(fmt.Sprintf("检查总数: %d, 通过: %d, 失败: %d\n", len(results), passCount, failCount))

	if len(criticalIssues) > 0 {
		summary.WriteString(fmt.Sprintf("严重问题 (%d 项): %s\n", len(criticalIssues), strings.Join(criticalIssues, ", ")))
	}
	if len(highIssues) > 0 {
		summary.WriteString(fmt.Sprintf("高危问题 (%d 项): %s\n", len(highIssues), strings.Join(highIssues, ", ")))
	}
	if len(mediumIssues) > 0 {
		summary.WriteString(fmt.Sprintf("中危问题 (%d 项): %s\n", len(mediumIssues), strings.Join(mediumIssues, ", ")))
	}

	return summary.String()
}

// 获取检查权重
func getCheckWeight(severity CheckSeverity) int {
	switch severity {
	case SeverityCritical:
		return 10
	case SeverityHigh:
		return 8
	case SeverityMedium:
		return 5
	case SeverityLow:
		return 3
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// 具体检查函数实现

// 检查安全级别
func (c *SecurityChecker) checkSecurityLevel() CheckResult {
	level := c.config.Level
	passed := level >= security.LevelIntermediate

	var details, suggestion string
	switch level {
	case security.LevelBasic:
		details = "当前安全级别为'基础'，这在生产环境中通常不够安全"
		suggestion = "建议将安全级别提高至'中级'或'严格'"
	case security.LevelIntermediate:
		details = "当前安全级别为'中级'，适合一般的Web应用"
		suggestion = "如果应用处理敏感数据，建议提高至'严格'级别"
	case security.LevelStrict:
		details = "当前安全级别为'严格'，这是最高的预设安全级别"
		suggestion = "您已经使用了最高级别的安全配置"
	default:
		details = "使用了自定义安全级别"
		suggestion = "确保自定义安全配置满足您的需求"
	}

	return CheckResult{
		Name:        "安全级别配置",
		Description: "检查系统的整体安全级别配置",
		Severity:    SeverityHigh,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查会话安全
func (c *SecurityChecker) checkSessionSecure() CheckResult {
	session := c.config.Session
	passed := session.Secure && session.HttpOnly

	details := fmt.Sprintf("Cookie安全设置: Secure=%v, HttpOnly=%v, SameSite=%v",
		session.Secure, session.HttpOnly, session.SameSite)
	suggestion := ""

	if !session.Secure {
		suggestion += "启用Cookie的Secure标志以确保仅通过HTTPS发送; "
	}
	if !session.HttpOnly {
		suggestion += "启用Cookie的HttpOnly标志以防止JavaScript访问; "
	}
	if session.SameSite != 2 { // 2 = SameSiteStrictMode
		suggestion += "考虑使用SameSite=Strict增强跨域保护; "
	}
	if session.AccessTokenExpiry > 30*60*1000000000 { // 30分钟 (纳秒)
		suggestion += "减少访问令牌过期时间以降低风险; "
	}

	if suggestion == "" {
		suggestion = "当前会话配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "会话安全配置",
		Description: "检查会话Cookie和令牌的安全设置",
		Severity:    SeverityHigh,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查CSRF保护
func (c *SecurityChecker) checkCSRFProtection() CheckResult {
	csrf := c.config.CSRF
	passed := csrf.Enabled && csrf.TokenLength >= 32

	details := fmt.Sprintf("CSRF保护: 启用=%v, 令牌长度=%d",
		csrf.Enabled, csrf.TokenLength)
	suggestion := ""

	if !csrf.Enabled {
		suggestion += "启用CSRF保护以防止跨站请求伪造攻击; "
	}
	if csrf.TokenLength < 32 {
		suggestion += "增加CSRF令牌长度至少32字节以增强安全性; "
	}

	if suggestion == "" {
		suggestion = "当前CSRF配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "CSRF保护",
		Description: "检查跨站请求伪造保护设置",
		Severity:    SeverityHigh,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查HSTS
func (c *SecurityChecker) checkHSTS() CheckResult {
	headers := c.config.Headers
	// 严格模式下预期HSTS应该启用，包含子域名，预加载，最大年龄至少180天
	passed := headers.EnableHSTS &&
		headers.HSTSIncludeSubdomains &&
		headers.HSTSMaxAge >= 15552000*time.Second // 180天

	details := fmt.Sprintf("HSTS: 启用=%v, 包含子域名=%v, 预加载=%v, 最大年龄=%v",
		headers.EnableHSTS, headers.HSTSIncludeSubdomains, headers.HSTSPreload, headers.HSTSMaxAge)
	suggestion := ""

	if !headers.EnableHSTS {
		suggestion += "启用HSTS以强制客户端使用HTTPS连接; "
	}
	if !headers.HSTSIncludeSubdomains {
		suggestion += "配置HSTS包含子域名以保护所有子域; "
	}
	if !headers.HSTSPreload {
		suggestion += "考虑启用HSTS预加载以增强安全性; "
	}
	if headers.HSTSMaxAge < 15552000*time.Second {
		suggestion += "增加HSTS最大年龄至少180天; "
	}

	if suggestion == "" {
		suggestion = "当前HSTS配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "HTTP严格传输安全",
		Description: "检查HSTS配置",
		Severity:    SeverityMedium,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查XSS保护
func (c *SecurityChecker) checkXSSProtection() CheckResult {
	headers := c.config.Headers
	passed := headers.EnableXSSProtection

	details := fmt.Sprintf("XSS保护: 启用=%v", headers.EnableXSSProtection)
	suggestion := ""

	if !headers.EnableXSSProtection {
		suggestion = "启用X-XSS-Protection头以增强XSS防护（虽然现代浏览器更依赖CSP）"
	} else {
		suggestion = "当前XSS保护配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "XSS保护",
		Description: "检查跨站脚本保护设置",
		Severity:    SeverityMedium,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查内容类型嗅探保护
func (c *SecurityChecker) checkContentTypeNoSniff() CheckResult {
	headers := c.config.Headers
	passed := headers.EnableContentTypeNosniff

	details := fmt.Sprintf("内容类型嗅探保护: 启用=%v", headers.EnableContentTypeNosniff)
	suggestion := ""

	if !headers.EnableContentTypeNosniff {
		suggestion = "启用X-Content-Type-Options: nosniff以防止MIME类型嗅探"
	} else {
		suggestion = "当前内容类型嗅探保护配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "内容类型嗅探保护",
		Description: "检查X-Content-Type-Options设置",
		Severity:    SeverityMedium,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查X-Frame-Options
func (c *SecurityChecker) checkXFrameOptions() CheckResult {
	headers := c.config.Headers
	passed := headers.EnableXFrameOptions && (headers.XFrameOptionsValue == "DENY" || headers.XFrameOptionsValue == "SAMEORIGIN")

	details := fmt.Sprintf("X-Frame-Options: 启用=%v, 值=%s", headers.EnableXFrameOptions, headers.XFrameOptionsValue)
	suggestion := ""

	if !headers.EnableXFrameOptions {
		suggestion = "启用X-Frame-Options以防止点击劫持攻击"
	} else if headers.XFrameOptionsValue != "DENY" && headers.XFrameOptionsValue != "SAMEORIGIN" {
		suggestion = "将X-Frame-Options设置为DENY或SAMEORIGIN以增强安全性"
	} else {
		suggestion = "当前X-Frame-Options配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "X-Frame-Options",
		Description: "检查点击劫持保护设置",
		Severity:    SeverityMedium,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查内容安全策略
func (c *SecurityChecker) checkCSP() CheckResult {
	headers := c.config.Headers
	csp := headers.ContentSecurityPolicy

	// CSP应该存在且不应该包含unsafe-inline/unsafe-eval
	passed := csp != "" &&
		!strings.Contains(csp, "'unsafe-inline'") &&
		!strings.Contains(csp, "'unsafe-eval'") &&
		strings.Contains(csp, "default-src")

	details := fmt.Sprintf("内容安全策略: %s", csp)
	suggestion := ""

	if csp == "" {
		suggestion = "配置内容安全策略以防止XSS和数据注入攻击"
	} else if strings.Contains(csp, "'unsafe-inline'") || strings.Contains(csp, "'unsafe-eval'") {
		suggestion = "移除CSP中的'unsafe-inline'和'unsafe-eval'指令，使用nonce或hash替代"
	} else if !strings.Contains(csp, "default-src") {
		suggestion = "确保CSP包含default-src指令作为基础保护"
	} else if !strings.Contains(csp, "upgrade-insecure-requests") {
		suggestion += "考虑添加upgrade-insecure-requests指令以自动升级HTTP请求; "
		suggestion += "CSP基本配置良好，考虑更精细的指令控制"
	} else {
		suggestion = "当前CSP配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "内容安全策略",
		Description: "检查内容安全策略配置",
		Severity:    SeverityHigh,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查请求限流
func (c *SecurityChecker) checkRateLimit() CheckResult {
	rateLimit := c.config.RateLimit
	passed := rateLimit.Enabled && rateLimit.EnableIPRateLimit

	details := fmt.Sprintf("请求限流: 启用=%v, IP限流=%v, 速率=%v, 突发=%v",
		rateLimit.Enabled, rateLimit.EnableIPRateLimit, rateLimit.Rate, rateLimit.Burst)
	suggestion := ""

	if !rateLimit.Enabled {
		suggestion = "启用请求限流以防止DoS攻击和API滥用"
	} else if !rateLimit.EnableIPRateLimit {
		suggestion = "启用基于IP的限流以更有效地防止攻击"
	} else if rateLimit.Rate > 100 {
		suggestion = "考虑降低全局限流速率以增强安全性"
	} else {
		suggestion = "当前限流配置已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "请求限流",
		Description: "检查API请求限流配置",
		Severity:    SeverityMedium,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}

// 检查密码策略
func (c *SecurityChecker) checkPasswordPolicy() CheckResult {
	pwd := c.config.Password
	// 严格标准：长度>=12、要求大小写字母、数字和特殊字符，且有最大使用期限
	passed := pwd.MinLength >= 12 &&
		pwd.RequireUppercase &&
		pwd.RequireLowercase &&
		pwd.RequireDigits &&
		pwd.RequireSpecialChars &&
		pwd.MaxAge > 0

	details := fmt.Sprintf("密码策略: 最小长度=%d, 要求大写=%v, 要求小写=%v, 要求数字=%v, 要求特殊字符=%v, 最长使用期=%v",
		pwd.MinLength, pwd.RequireUppercase, pwd.RequireLowercase, pwd.RequireDigits, pwd.RequireSpecialChars, pwd.MaxAge)
	suggestion := ""

	if pwd.MinLength < 12 {
		suggestion += "增加密码最小长度至12个字符; "
	}
	if !pwd.RequireUppercase {
		suggestion += "要求密码包含大写字母; "
	}
	if !pwd.RequireLowercase {
		suggestion += "要求密码包含小写字母; "
	}
	if !pwd.RequireDigits {
		suggestion += "要求密码包含数字; "
	}
	if !pwd.RequireSpecialChars {
		suggestion += "要求密码包含特殊字符; "
	}
	if pwd.MaxAge == 0 {
		suggestion += "设置密码最长使用期限; "
	}
	if pwd.PreventReuseCount < 5 {
		suggestion += "增加密码重用防止计数以防止使用以前的密码; "
	}

	if suggestion == "" {
		suggestion = "当前密码策略已符合安全最佳实践"
	}

	return CheckResult{
		Name:        "密码策略",
		Description: "检查密码复杂度和生命周期策略",
		Severity:    SeverityHigh,
		Passed:      passed,
		Details:     details,
		Suggestion:  suggestion,
	}
}
