package secureheader

import (
	"github.com/dormoron/mist"
)

// Options 定义了安全头部中间件的配置选项
type Options struct {
	// XSSProtection 提供XSS保护
	XSSProtection string

	// ContentTypeNosniff 防止MIME类型嗅探
	ContentTypeNosniff string

	// XFrameOptions 控制iframe嵌入
	XFrameOptions string

	// HSTSMaxAge HSTS最大有效期（秒）
	HSTSMaxAge int

	// HSTSExcludeSubdomains 是否排除子域名
	HSTSExcludeSubdomains bool

	// ContentSecurityPolicy 内容安全策略
	ContentSecurityPolicy string

	// ReferrerPolicy 引用策略
	ReferrerPolicy string

	// PermissionsPolicy 权限策略
	PermissionsPolicy string

	// CrossOriginOpenerPolicy 跨源打开者策略
	CrossOriginOpenerPolicy string

	// CrossOriginEmbedderPolicy 跨源嵌入策略
	CrossOriginEmbedderPolicy string

	// CrossOriginResourcePolicy 跨源资源策略
	CrossOriginResourcePolicy string
}

// DefaultOptions 返回默认的安全头部选项
func DefaultOptions() Options {
	return Options{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		HSTSMaxAge:                31536000,
		HSTSExcludeSubdomains:     false,
		ContentSecurityPolicy:     "",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// NewMiddleware 创建一个新的安全头部中间件
func NewMiddleware(options Options) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 添加安全相关的HTTP头

			// 防止XSS攻击
			if options.XSSProtection != "" {
				ctx.Header("X-XSS-Protection", options.XSSProtection)
			}

			// 防止MIME类型嗅探
			if options.ContentTypeNosniff != "" {
				ctx.Header("X-Content-Type-Options", options.ContentTypeNosniff)
			}

			// 控制iframe嵌入
			if options.XFrameOptions != "" {
				ctx.Header("X-Frame-Options", options.XFrameOptions)
			}

			// HTTP严格传输安全
			if options.HSTSMaxAge > 0 && ctx.Request.TLS != nil {
				hstsValue := "max-age=" + string(options.HSTSMaxAge)
				if !options.HSTSExcludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				ctx.Header("Strict-Transport-Security", hstsValue)
			}

			// 内容安全策略
			if options.ContentSecurityPolicy != "" {
				ctx.Header("Content-Security-Policy", options.ContentSecurityPolicy)
			}

			// 引用策略
			if options.ReferrerPolicy != "" {
				ctx.Header("Referrer-Policy", options.ReferrerPolicy)
			}

			// 权限策略
			if options.PermissionsPolicy != "" {
				ctx.Header("Permissions-Policy", options.PermissionsPolicy)
			}

			// 跨源打开者策略
			if options.CrossOriginOpenerPolicy != "" {
				ctx.Header("Cross-Origin-Opener-Policy", options.CrossOriginOpenerPolicy)
			}

			// 跨源嵌入策略
			if options.CrossOriginEmbedderPolicy != "" {
				ctx.Header("Cross-Origin-Embedder-Policy", options.CrossOriginEmbedderPolicy)
			}

			// 跨源资源策略
			if options.CrossOriginResourcePolicy != "" {
				ctx.Header("Cross-Origin-Resource-Policy", options.CrossOriginResourcePolicy)
			}

			// 添加请求ID
			requestID := ctx.RequestID()
			if requestID != "" {
				ctx.Header("X-Request-ID", requestID)
			}

			// 继续处理请求
			next(ctx)
		}
	}
}

// WithSecureHeaders 创建一个使用默认选项的安全头部中间件
func WithSecureHeaders() mist.Middleware {
	return NewMiddleware(DefaultOptions())
}

// WithCustomSecureHeaders 创建一个使用自定义选项的安全头部中间件
func WithCustomSecureHeaders(configurator func(*Options)) mist.Middleware {
	options := DefaultOptions()
	configurator(&options)
	return NewMiddleware(options)
}
