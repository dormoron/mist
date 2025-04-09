package headers

import (
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
}

// DefaultConfig 返回默认的安全头部配置
func DefaultConfig() Config {
	// 从全局安全配置中获取
	secConfig := security.GetSecurityConfig()

	config := Config{
		XSSProtection:             secConfig.Headers.EnableXSSProtection,
		ContentTypeNoSniff:        secConfig.Headers.EnableContentTypeNosniff,
		XFrameOptions:             secConfig.Headers.XFrameOptionsValue,
		HSTS:                      secConfig.Headers.EnableHSTS,
		HSTSMaxAge:                int(secConfig.Headers.HSTSMaxAge.Seconds()),
		HSTSIncludeSubdomains:     secConfig.Headers.HSTSIncludeSubdomains,
		HSTSPreload:               secConfig.Headers.HSTSPreload,
		ContentSecurityPolicy:     secConfig.Headers.ContentSecurityPolicy,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		XContentTypeOptions:       "nosniff",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
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
				ctx.Header("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				ctx.Header("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				ctx.Header("Permissions-Policy", config.PermissionsPolicy)
			}

			// Expect-CT
			if config.ExpectCT {
				value := fmt.Sprintf("max-age=%d", config.ExpectCTMaxAge)
				if config.ExpectCTEnforce {
					value += ", enforce"
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
}

// NewCSPBuilder 创建新的CSP生成器
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[string][]string),
	}
}

// Add 添加内容安全策略指令
func (b *CSPBuilder) Add(directive string, values ...string) *CSPBuilder {
	b.directives[directive] = append(b.directives[directive], values...)
	return b
}

// String 生成内容安全策略字符串
func (b *CSPBuilder) String() string {
	var policies []string

	for directive, values := range b.directives {
		if len(values) > 0 {
			policy := directive + " " + strings.Join(values, " ")
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
