package headers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
)

// Config 安全头部配置
type Config struct {
	// XSSProtection 启用XSS保护
	XSSProtection bool
	// ContentTypeNoSniff 禁止内容类型嗅探
	ContentTypeNoSniff bool
	// XFrameOptions X-Frame-Options 设置
	XFrameOptions string
	// HSTS 是否启用HTTP严格传输安全
	HSTS bool
	// HSTSMaxAge HSTS最大存活时间（秒）
	HSTSMaxAge int
	// HSTSIncludeSubdomains 是否包含子域名
	HSTSIncludeSubdomains bool
	// HSTSPreload 是否启用预加载
	HSTSPreload bool
	// ContentSecurityPolicy 内容安全策略
	ContentSecurityPolicy string
	// ReferrerPolicy 引用来源政策
	ReferrerPolicy string
	// PermissionsPolicy 权限策略
	PermissionsPolicy string
	// XContentTypeOptions X-Content-Type-Options 头部
	XContentTypeOptions string
	// ExpectCT 证书透明度期望
	ExpectCT bool
	// ExpectCTMaxAge Expect-CT 最大存活时间（秒）
	ExpectCTMaxAge int
	// ExpectCTEnforce 是否强制执行Expect-CT
	ExpectCTEnforce bool
	// CrossOriginEmbedderPolicy 跨源嵌入者策略
	CrossOriginEmbedderPolicy string
	// CrossOriginOpenerPolicy 跨源打开者策略
	CrossOriginOpenerPolicy string
	// CrossOriginResourcePolicy 跨源资源策略
	CrossOriginResourcePolicy string
	// DocumentPolicy 文档策略
	DocumentPolicy string
	// ReportTo 违规报告配置
	ReportTo string
	// ReportURI CSP违规报告URI
	ReportURI string
	// EnableNonce 是否启用CSP nonce
	EnableNonce bool
	// EnableUpgradeInsecureRequests 是否启用升级不安全请求
	EnableUpgradeInsecureRequests bool
	// CSPReporting 是否启用CSP报告模式
	CSPReporting bool
}

// DefaultConfig 返回默认的安全头部配置
func DefaultConfig() Config {
	// 从全局安全配置中获取
	secConfig := security.GetSecurityConfig()

	config := Config{
		XSSProtection:                 secConfig.Headers.EnableXSSProtection,
		ContentTypeNoSniff:            secConfig.Headers.EnableContentTypeNosniff,
		XFrameOptions:                 secConfig.Headers.XFrameOptionsValue,
		HSTS:                          secConfig.Headers.EnableHSTS,
		HSTSMaxAge:                    int(secConfig.Headers.HSTSMaxAge.Seconds()),
		HSTSIncludeSubdomains:         secConfig.Headers.HSTSIncludeSubdomains,
		HSTSPreload:                   secConfig.Headers.HSTSPreload,
		ContentSecurityPolicy:         secConfig.Headers.ContentSecurityPolicy,
		ReferrerPolicy:                "strict-origin-when-cross-origin",
		XContentTypeOptions:           "nosniff",
		CrossOriginEmbedderPolicy:     "require-corp",
		CrossOriginOpenerPolicy:       "same-origin",
		CrossOriginResourcePolicy:     "same-origin",
		DocumentPolicy:                "",
		ReportTo:                      "",
		ReportURI:                     "/api/security/report",
		EnableNonce:                   true,
		EnableUpgradeInsecureRequests: true,
		CSPReporting:                  false,
	}

	return config
}

// New 创建一个新的安全头部中间件
func New(options ...func(*Config)) mist.Middleware {
	config := DefaultConfig()

	for _, option := range options {
		option(&config)
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 如果启用nonce，生成一个nonce值
			var nonce string
			if config.EnableNonce {
				nonce = generateNonce()
				ctx.Set("csp_nonce", nonce)
			}

			// X-XSS-Protection
			if config.XSSProtection {
				ctx.Header("X-XSS-Protection", "1; mode=block")
			}

			// X-Content-Type-Options
			if config.ContentTypeNoSniff {
				ctx.Header("X-Content-Type-Options", config.XContentTypeOptions)
			}

			// X-Frame-Options
			if config.XFrameOptions != "" {
				ctx.Header("X-Frame-Options", config.XFrameOptions)
			}

			// Strict-Transport-Security (HSTS)
			if config.HSTS {
				value := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
				if config.HSTSIncludeSubdomains {
					value += "; includeSubDomains"
				}
				if config.HSTSPreload {
					value += "; preload"
				}
				ctx.Header("Strict-Transport-Security", value)
			}

			// Content-Security-Policy
			if config.ContentSecurityPolicy != "" {
				cspValue := config.ContentSecurityPolicy

				// 如果启用nonce，添加到CSP中的script-src和style-src
				if config.EnableNonce && nonce != "" {
					cspValue = addNonceToCSP(cspValue, nonce)
				}

				// 如果启用升级不安全请求
				if config.EnableUpgradeInsecureRequests && !strings.Contains(cspValue, "upgrade-insecure-requests") {
					if !strings.HasSuffix(cspValue, "; ") && cspValue != "" {
						cspValue += "; "
					}
					cspValue += "upgrade-insecure-requests"
				}

				// 如果启用报告
				if config.CSPReporting && config.ReportURI != "" {
					if !strings.HasSuffix(cspValue, "; ") && cspValue != "" {
						cspValue += "; "
					}
					cspValue += "report-uri " + config.ReportURI
				}

				ctx.Header("Content-Security-Policy", cspValue)

				// 如果启用报告模式，同时设置报告模式头
				if config.CSPReporting {
					ctx.Header("Content-Security-Policy-Report-Only", cspValue)
				}
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				ctx.Header("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				ctx.Header("Permissions-Policy", config.PermissionsPolicy)
			}

			// Document-Policy
			if config.DocumentPolicy != "" {
				ctx.Header("Document-Policy", config.DocumentPolicy)
			}

			// Report-To
			if config.ReportTo != "" {
				ctx.Header("Report-To", config.ReportTo)
			}

			// Expect-CT
			if config.ExpectCT {
				value := fmt.Sprintf("max-age=%d", config.ExpectCTMaxAge)
				if config.ExpectCTEnforce {
					value += ", enforce"
				}
				if config.ReportURI != "" {
					value += fmt.Sprintf(`, report-uri="%s"`, config.ReportURI)
				}
				ctx.Header("Expect-CT", value)
			}

			// Cross-Origin-Embedder-Policy
			if config.CrossOriginEmbedderPolicy != "" {
				ctx.Header("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Cross-Origin-Opener-Policy
			if config.CrossOriginOpenerPolicy != "" {
				ctx.Header("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}

			// Cross-Origin-Resource-Policy
			if config.CrossOriginResourcePolicy != "" {
				ctx.Header("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}

			next(ctx)
		}
	}
}

// 生成随机nonce值
func generateNonce() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// 添加nonce到CSP
func addNonceToCSP(csp, nonce string) string {
	if csp == "" {
		return fmt.Sprintf("script-src 'self' 'nonce-%s'; style-src 'self' 'nonce-%s'", nonce, nonce)
	}

	directives := strings.Split(csp, ";")
	var result []string
	scriptSrcFound := false
	styleSrcFound := false

	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}

		if strings.HasPrefix(directive, "script-src") {
			scriptSrcFound = true
			// 如果不包含nonce，添加它
			if !strings.Contains(directive, "'nonce-") {
				directive += fmt.Sprintf(" 'nonce-%s'", nonce)
			}
		} else if strings.HasPrefix(directive, "style-src") {
			styleSrcFound = true
			// 如果不包含nonce，添加它
			if !strings.Contains(directive, "'nonce-") {
				directive += fmt.Sprintf(" 'nonce-%s'", nonce)
			}
		}
		result = append(result, directive)
	}

	// 如果没有找到script-src，添加它
	if !scriptSrcFound {
		result = append(result, fmt.Sprintf("script-src 'self' 'nonce-%s'", nonce))
	}

	// 如果没有找到style-src，添加它
	if !styleSrcFound {
		result = append(result, fmt.Sprintf("style-src 'self' 'nonce-%s'", nonce))
	}

	return strings.Join(result, "; ")
}

// 选项函数

// WithXSSProtection 设置XSS保护
func WithXSSProtection(enabled bool) func(*Config) {
	return func(c *Config) {
		c.XSSProtection = enabled
	}
}

// WithContentTypeNoSniff 设置内容类型嗅探保护
func WithContentTypeNoSniff(enabled bool) func(*Config) {
	return func(c *Config) {
		c.ContentTypeNoSniff = enabled
	}
}

// WithXFrameOptions 设置X-Frame-Options
func WithXFrameOptions(option string) func(*Config) {
	return func(c *Config) {
		c.XFrameOptions = option
	}
}

// WithHSTS 设置HSTS
func WithHSTS(enabled bool, maxAge int, includeSubdomains bool, preload bool) func(*Config) {
	return func(c *Config) {
		c.HSTS = enabled
		c.HSTSMaxAge = maxAge
		c.HSTSIncludeSubdomains = includeSubdomains
		c.HSTSPreload = preload
	}
}

// WithContentSecurityPolicy 设置内容安全策略
func WithContentSecurityPolicy(policy string) func(*Config) {
	return func(c *Config) {
		c.ContentSecurityPolicy = policy
	}
}

// WithReferrerPolicy 设置引用来源政策
func WithReferrerPolicy(policy string) func(*Config) {
	return func(c *Config) {
		c.ReferrerPolicy = policy
	}
}

// WithPermissionsPolicy 设置权限策略
func WithPermissionsPolicy(policy string) func(*Config) {
	return func(c *Config) {
		c.PermissionsPolicy = policy
	}
}

// WithDocumentPolicy 设置文档策略
func WithDocumentPolicy(policy string) func(*Config) {
	return func(c *Config) {
		c.DocumentPolicy = policy
	}
}

// WithReportingEndpoints 设置报告终端
func WithReportingEndpoints(endpoints string) func(*Config) {
	return func(c *Config) {
		c.ReportTo = endpoints
	}
}

// WithReportURI 设置报告URI
func WithReportURI(uri string) func(*Config) {
	return func(c *Config) {
		c.ReportURI = uri
	}
}

// WithCSPReporting 设置CSP报告模式
func WithCSPReporting(enabled bool) func(*Config) {
	return func(c *Config) {
		c.CSPReporting = enabled
	}
}

// WithNonce 设置是否启用Nonce
func WithNonce(enabled bool) func(*Config) {
	return func(c *Config) {
		c.EnableNonce = enabled
	}
}

// WithUpgradeInsecureRequests 设置是否启用升级不安全请求
func WithUpgradeInsecureRequests(enabled bool) func(*Config) {
	return func(c *Config) {
		c.EnableUpgradeInsecureRequests = enabled
	}
}

// WithExpectCT 设置Expect-CT
func WithExpectCT(enabled bool, maxAge int, enforce bool) func(*Config) {
	return func(c *Config) {
		c.ExpectCT = enabled
		c.ExpectCTMaxAge = maxAge
		c.ExpectCTEnforce = enforce
	}
}

// WithCrossOriginPolicies 设置跨源政策
func WithCrossOriginPolicies(embedder, opener, resource string) func(*Config) {
	return func(c *Config) {
		c.CrossOriginEmbedderPolicy = embedder
		c.CrossOriginOpenerPolicy = opener
		c.CrossOriginResourcePolicy = resource
	}
}

// 辅助函数

// CSPBuilder 用于构建内容安全策略的生成器
type CSPBuilder struct {
	directives map[string][]string
	requireSRI map[string]bool
}

// NewCSPBuilder 创建新的CSP生成器
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[string][]string),
		requireSRI: make(map[string]bool),
	}
}

// Add 添加内容安全策略指令
func (b *CSPBuilder) Add(directive string, values ...string) *CSPBuilder {
	b.directives[directive] = append(b.directives[directive], values...)
	return b
}

// RequireSRI 为特定指令要求使用SRI
func (b *CSPBuilder) RequireSRI(directive string, require bool) *CSPBuilder {
	b.requireSRI[directive] = require
	return b
}

// String 生成内容安全策略字符串
func (b *CSPBuilder) String() string {
	var policies []string

	for directive, values := range b.directives {
		if len(values) > 0 {
			policy := directive
			// 检查是否需要为此指令添加SRI要求
			if require, ok := b.requireSRI[directive]; ok && require {
				policy += " 'require-sri-for'"
			}
			policy += " " + strings.Join(values, " ")
			policies = append(policies, policy)
		} else {
			policies = append(policies, directive)
		}
	}

	return strings.Join(policies, "; ")
}

// 常用CSP预设

// CSPStrict 返回严格的CSP策略
func CSPStrict() string {
	return NewCSPBuilder().
		Add("default-src", "'self'").
		Add("script-src", "'self'").
		Add("object-src", "'none'").
		Add("style-src", "'self'").
		Add("img-src", "'self'").
		Add("media-src", "'self'").
		Add("frame-src", "'none'").
		Add("font-src", "'self'").
		Add("connect-src", "'self'").
		Add("base-uri", "'self'").
		Add("form-action", "'self'").
		Add("frame-ancestors", "'none'").
		Add("upgrade-insecure-requests").
		RequireSRI("script-src", true).
		RequireSRI("style-src", true).
		String()
}

// CSPModern 返回适合现代Web应用的CSP策略
func CSPModern() string {
	return NewCSPBuilder().
		Add("default-src", "'self'").
		Add("script-src", "'self'").
		Add("script-src-elem", "'self'").
		Add("script-src-attr", "'none'").
		Add("style-src", "'self'").
		Add("style-src-elem", "'self'").
		Add("style-src-attr", "'none'").
		Add("img-src", "'self' data:").
		Add("font-src", "'self'").
		Add("connect-src", "'self'").
		Add("media-src", "'self'").
		Add("object-src", "'none'").
		Add("child-src", "'none'").
		Add("frame-ancestors", "'none'").
		Add("form-action", "'self'").
		Add("base-uri", "'self'").
		Add("manifest-src", "'self'").
		Add("worker-src", "'self'").
		Add("upgrade-insecure-requests").
		RequireSRI("script-src", true).
		RequireSRI("style-src", true).
		String()
}

// CSPBasic 返回基本的CSP策略
func CSPBasic() string {
	return NewCSPBuilder().
		Add("default-src", "'self'").
		Add("img-src", "'self' data:").
		Add("script-src", "'self'").
		Add("style-src", "'self' 'unsafe-inline'").
		Add("font-src", "'self'").
		String()
}

// DefaultPermissionsPolicy 返回默认的权限策略
func DefaultPermissionsPolicy() string {
	permissions := []string{
		"accelerometer=()",
		"ambient-light-sensor=()",
		"autoplay=(self)",
		"battery=(self)",
		"camera=(self)",
		"display-capture=(self)",
		"document-domain=(self)",
		"encrypted-media=(self)",
		"execution-while-not-rendered=(self)",
		"execution-while-out-of-viewport=(self)",
		"fullscreen=(self)",
		"geolocation=(self)",
		"gyroscope=()",
		"magnetometer=()",
		"microphone=(self)",
		"midi=(self)",
		"navigation-override=(self)",
		"payment=(self)",
		"picture-in-picture=(self)",
		"publickey-credentials-get=(self)",
		"screen-wake-lock=(self)",
		"sync-xhr=(self)",
		"usb=(self)",
		"web-share=(self)",
		"xr-spatial-tracking=()",
	}
	return strings.Join(permissions, ", ")
}

// DefaultDocumentPolicy 返回默认的文档策略
func DefaultDocumentPolicy() string {
	policies := []string{
		"document-write=?0",
		"force-load-at-top=?0",
		"js-profiling=?0",
		"legacy-image-formats=?0",
		"sync-script=?0",
		"sync-xhr=?0",
		"unsized-media=?0",
	}
	return strings.Join(policies, ", ")
}

// DefaultReportingEndpoints 返回默认的报告终端配置
func DefaultReportingEndpoints(endpoint string) string {
	if endpoint == "" {
		endpoint = "/api/security/report"
	}
	return fmt.Sprintf(`{"endpoints":[{"url":"%s"}],"group":"default","max_age":10886400}`, endpoint)
}

// 预设X-Frame-Options

// XFrameDeny 拒绝所有iframe嵌入
const XFrameDeny = "DENY"

// XFrameSameOrigin 仅允许同源iframe嵌入
const XFrameSameOrigin = "SAMEORIGIN"

// XFrameAllowFrom 允许特定来源的iframe嵌入
func XFrameAllowFrom(uri string) string {
	return "ALLOW-FROM " + uri
}

// 预设Referrer-Policy

// ReferrerNoReferrer 不发送Referrer信息
const ReferrerNoReferrer = "no-referrer"

// ReferrerNoReferrerWhenDowngrade 仅在HTTPS到HTTP时不发送
const ReferrerNoReferrerWhenDowngrade = "no-referrer-when-downgrade"

// ReferrerSameOrigin 仅同源时发送
const ReferrerSameOrigin = "same-origin"

// ReferrerStrictOrigin 只发送源（严格）
const ReferrerStrictOrigin = "strict-origin"

// ReferrerStrictOriginWhenCrossOrigin 跨域时仅发送源（严格）
const ReferrerStrictOriginWhenCrossOrigin = "strict-origin-when-cross-origin"

// 其他辅助函数

// HSTSValue 生成HSTS头部值
func HSTSValue(maxAge int, includeSubdomains bool, preload bool) string {
	value := "max-age=" + strconv.Itoa(maxAge)
	if includeSubdomains {
		value += "; includeSubDomains"
	}
	if preload {
		value += "; preload"
	}
	return value
}
