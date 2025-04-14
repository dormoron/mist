package headers

import (
	"bytes"
	"net/http"
	"strconv"
	"time"
)

// SecurityHeaders 包含所有可配置的安全HTTP头
type SecurityHeaders struct {
	// XFrameOptions 控制页面是否可以被嵌入到iframe中
	// 可选值: DENY, SAMEORIGIN, ALLOW-FROM uri
	XFrameOptions string

	// XContentTypeOptions 防止MIME类型嗅探
	// 可选值: nosniff
	XContentTypeOptions string

	// XSSProtection 启用跨站脚本过滤
	// 可选值: 0, 1, 1; mode=block
	XSSProtection string

	// ContentSecurityPolicy 内容安全策略
	ContentSecurityPolicy string

	// ReferrerPolicy 控制Referer头的发送
	ReferrerPolicy string

	// StrictTransportSecurity HTTP严格传输安全
	// includeSubDomains: 是否包含子域名
	// preload: 是否加入HSTS预加载列表
	// maxAge: 有效期（秒）
	HSTS struct {
		Enable            bool
		MaxAge            time.Duration
		IncludeSubDomains bool
		Preload           bool
	}

	// PermissionsPolicy 权限策略
	PermissionsPolicy string

	// CacheControl 缓存控制
	CacheControl string

	// ExpectCT 证书透明度期望
	ExpectCT struct {
		Enable    bool
		MaxAge    time.Duration
		Enforce   bool
		ReportURI string
	}

	// CrossOriginEmbedderPolicy 跨域嵌入者策略
	CrossOriginEmbedderPolicy string

	// CrossOriginOpenerPolicy 跨域打开者策略
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy 跨域资源策略
	CrossOriginResourcePolicy string

	// ReportTo 报告机制
	ReportTo string
}

// DefaultSecurityHeaders 返回推荐的默认安全头设置
func DefaultSecurityHeaders() *SecurityHeaders {
	headers := &SecurityHeaders{
		XFrameOptions:             "SAMEORIGIN",
		XContentTypeOptions:       "nosniff",
		XSSProtection:             "1; mode=block",
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self'; object-src 'none'; img-src 'self' data:; style-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; require-trusted-types-for 'script';",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CacheControl:              "no-store, max-age=0",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		PermissionsPolicy:         "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
	}

	// 设置HSTS
	headers.HSTS.Enable = true
	headers.HSTS.MaxAge = 63072000 * time.Second // 2年
	headers.HSTS.IncludeSubDomains = true
	headers.HSTS.Preload = true

	// 设置Expect-CT
	headers.ExpectCT.Enable = true
	headers.ExpectCT.MaxAge = 86400 * time.Second // 1天
	headers.ExpectCT.Enforce = true

	return headers
}

// Middleware 创建一个添加安全头的HTTP中间件
func (sh *SecurityHeaders) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 添加X-Frame-Options
			if sh.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", sh.XFrameOptions)
			}

			// 添加X-Content-Type-Options
			if sh.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", sh.XContentTypeOptions)
			}

			// 添加X-XSS-Protection
			if sh.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", sh.XSSProtection)
			}

			// 添加Content-Security-Policy
			if sh.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", sh.ContentSecurityPolicy)
			}

			// 添加Referrer-Policy
			if sh.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", sh.ReferrerPolicy)
			}

			// 添加Strict-Transport-Security
			if sh.HSTS.Enable {
				value := "max-age=" + strconv.FormatInt(int64(sh.HSTS.MaxAge.Seconds()), 10)
				if sh.HSTS.IncludeSubDomains {
					value += "; includeSubDomains"
				}
				if sh.HSTS.Preload {
					value += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", value)
			}

			// 添加Permissions-Policy
			if sh.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", sh.PermissionsPolicy)
			}

			// 添加Cache-Control
			if sh.CacheControl != "" {
				w.Header().Set("Cache-Control", sh.CacheControl)
			}

			// 添加Expect-CT
			if sh.ExpectCT.Enable {
				value := "max-age=" + strconv.FormatInt(int64(sh.ExpectCT.MaxAge.Seconds()), 10)
				if sh.ExpectCT.Enforce {
					value += ", enforce"
				}
				if sh.ExpectCT.ReportURI != "" {
					value += ", report-uri=\"" + sh.ExpectCT.ReportURI + "\""
				}
				w.Header().Set("Expect-CT", value)
			}

			// 添加Cross-Origin-Embedder-Policy
			if sh.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", sh.CrossOriginEmbedderPolicy)
			}

			// 添加Cross-Origin-Opener-Policy
			if sh.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", sh.CrossOriginOpenerPolicy)
			}

			// 添加Cross-Origin-Resource-Policy
			if sh.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", sh.CrossOriginResourcePolicy)
			}

			// 添加Report-To
			if sh.ReportTo != "" {
				w.Header().Set("Report-To", sh.ReportTo)
			}

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// NewSecurityHeaders 创建安全头设置
func NewSecurityHeaders(options ...func(*SecurityHeaders)) *SecurityHeaders {
	sh := DefaultSecurityHeaders()

	for _, option := range options {
		option(sh)
	}

	return sh
}

// SetXFrameOptions 设置X-Frame-Options选项
func SetXFrameOptions(value string) func(*SecurityHeaders) {
	return func(sh *SecurityHeaders) {
		sh.XFrameOptions = value
	}
}

// SetContentSecurityPolicy 设置内容安全策略
func SetContentSecurityPolicy(value string) func(*SecurityHeaders) {
	return func(sh *SecurityHeaders) {
		sh.ContentSecurityPolicy = value
	}
}

// SetHSTS 设置HSTS策略
func SetHSTS(enable bool, maxAge time.Duration, includeSubDomains, preload bool) func(*SecurityHeaders) {
	return func(sh *SecurityHeaders) {
		sh.HSTS.Enable = enable
		sh.HSTS.MaxAge = maxAge
		sh.HSTS.IncludeSubDomains = includeSubDomains
		sh.HSTS.Preload = preload
	}
}

// RemoveServerHeaders 移除X-Powered-By和Server头的中间件
func RemoveServerHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Del("X-Powered-By")
			w.Header().Del("Server")
			next.ServeHTTP(w, r)
		})
	}
}

// AutoContentType 设置正确的Content-Type头的中间件
func AutoContentType() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			// 如果没有设置Content-Type且有内容，自动检测
			if rw.Header().Get("Content-Type") == "" && rw.buffer.Len() > 0 {
				contentType := http.DetectContentType(rw.buffer.Bytes())
				rw.Header().Set("Content-Type", contentType)
			}
		})
	}
}

// responseWriter 是一个自定义的响应写入器，用于捕获响应体
type responseWriter struct {
	http.ResponseWriter
	buffer  bytes.Buffer
	status  int
	written bool
}

// Write 实现http.ResponseWriter接口
func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
		if w.status == 0 {
			w.status = http.StatusOK
		}
	}
	w.buffer.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader 实现http.ResponseWriter接口
func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

// GetSecurityHeadersMiddleware 提供默认安全头中间件
func GetSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return NewSecurityHeaders().Middleware()
}
