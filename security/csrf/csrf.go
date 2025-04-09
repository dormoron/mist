package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

var (
	// ErrTokenNotFound 表示请求中未找到CSRF令牌
	ErrTokenNotFound = errors.New("CSRF token not found in request")
	// ErrTokenInvalid 表示请求中的CSRF令牌无效
	ErrTokenInvalid = errors.New("CSRF token invalid")
)

// Config 配置CSRF保护
type Config struct {
	// TokenLength 令牌长度，默认32字节
	TokenLength int
	// CookieName CSRF cookie名称，默认为"_csrf"
	CookieName string
	// CookiePath cookie路径，默认为"/"
	CookiePath string
	// CookieDomain cookie的域，可选
	CookieDomain string
	// CookieMaxAge cookie最大存活时间，默认为24小时
	CookieMaxAge time.Duration
	// CookieSecure 是否仅通过HTTPS发送cookie，默认为false
	CookieSecure bool
	// CookieHTTPOnly 是否禁止JavaScript访问cookie，默认为true
	CookieHTTPOnly bool
	// CookieSameSite SameSite属性，默认为Lax
	CookieSameSite http.SameSite
	// HeaderName 请求中CSRF头名称，默认为"X-CSRF-Token"
	HeaderName string
	// FormField 表单中CSRF字段名称，默认为"csrf_token"
	FormField string
	// ErrorHandler 自定义错误处理
	ErrorHandler func(ctx *mist.Context, err error)
	// IgnoreMethods 忽略的HTTP方法（默认忽略GET, HEAD, OPTIONS, TRACE）
	IgnoreMethods []string
}

// csrfProtection 实现CSRF保护
type csrfProtection struct {
	config Config
	mutex  sync.RWMutex
}

// New 创建新的CSRF保护中间件
func New(options ...func(*Config)) mist.Middleware {
	// 设置默认配置
	config := Config{
		TokenLength:    32,
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieMaxAge:   24 * time.Hour,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		HeaderName:     "X-CSRF-Token",
		FormField:      "csrf_token",
		IgnoreMethods:  []string{"GET", "HEAD", "OPTIONS", "TRACE"},
		ErrorHandler: func(ctx *mist.Context, err error) {
			ctx.AbortWithStatus(http.StatusForbidden)
		},
	}

	// 应用自定义选项
	for _, option := range options {
		option(&config)
	}

	protection := &csrfProtection{
		config: config,
	}

	return protection.middleware
}

// 生成CSRF令牌
func (c *csrfProtection) generateToken() (string, error) {
	bytes := make([]byte, c.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// 设置CSRF cookie
func (c *csrfProtection) setCSRFCookie(ctx *mist.Context, token string) {
	cookie := http.Cookie{
		Name:     c.config.CookieName,
		Value:    token,
		Path:     c.config.CookiePath,
		Domain:   c.config.CookieDomain,
		MaxAge:   int(c.config.CookieMaxAge.Seconds()),
		Secure:   c.config.CookieSecure,
		HttpOnly: c.config.CookieHTTPOnly,
		SameSite: c.config.CookieSameSite,
	}
	ctx.SetCookie(&cookie)
}

// 从请求中获取令牌
func (c *csrfProtection) getTokenFromRequest(ctx *mist.Context) string {
	// 首先从Header中查找
	token := ctx.Request.Header.Get(c.config.HeaderName)
	if token != "" {
		return token
	}

	// 然后从POST表单查找
	if ctx.Request.Method == "POST" {
		token = ctx.Request.PostFormValue(c.config.FormField)
	}

	return token
}

// 检查方法是否在忽略列表中
func (c *csrfProtection) isMethodIgnored(method string) bool {
	for _, m := range c.config.IgnoreMethods {
		if m == method {
			return true
		}
	}
	return false
}

// CSRF中间件
func (c *csrfProtection) middleware(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// 如果方法在忽略列表中，则跳过保护
		if c.isMethodIgnored(ctx.Request.Method) {
			// 为GET请求（通常是页面加载）生成令牌
			if ctx.Request.Method == "GET" {
				token, err := c.generateToken()
				if err != nil {
					c.config.ErrorHandler(ctx, fmt.Errorf("failed to generate CSRF token: %w", err))
					return
				}
				c.setCSRFCookie(ctx, token)
				// 设置上下文中的CSRF令牌，以便视图可以访问
				ctx.Set("csrf_token", token)
			}
			next(ctx)
			return
		}

		// 获取cookie中的令牌
		cookie, err := ctx.Request.Cookie(c.config.CookieName)
		if err != nil {
			c.config.ErrorHandler(ctx, ErrTokenNotFound)
			return
		}
		cookieToken := cookie.Value

		// 获取请求中的令牌
		requestToken := c.getTokenFromRequest(ctx)
		if requestToken == "" {
			c.config.ErrorHandler(ctx, ErrTokenNotFound)
			return
		}

		// 验证令牌
		if requestToken != cookieToken {
			c.config.ErrorHandler(ctx, ErrTokenInvalid)
			return
		}

		// 验证通过，继续处理请求
		next(ctx)
	}
}

// 提供配置选项函数
// WithTokenLength 设置令牌长度
func WithTokenLength(length int) func(*Config) {
	return func(c *Config) {
		c.TokenLength = length
	}
}

// WithCookieName 设置cookie名称
func WithCookieName(name string) func(*Config) {
	return func(c *Config) {
		c.CookieName = name
	}
}

// WithCookiePath 设置cookie路径
func WithCookiePath(path string) func(*Config) {
	return func(c *Config) {
		c.CookiePath = path
	}
}

// WithCookieDomain 设置cookie域
func WithCookieDomain(domain string) func(*Config) {
	return func(c *Config) {
		c.CookieDomain = domain
	}
}

// WithCookieMaxAge 设置cookie最大存活时间
func WithCookieMaxAge(maxAge time.Duration) func(*Config) {
	return func(c *Config) {
		c.CookieMaxAge = maxAge
	}
}

// WithCookieSecure 设置cookie是否仅HTTPS
func WithCookieSecure(secure bool) func(*Config) {
	return func(c *Config) {
		c.CookieSecure = secure
	}
}

// WithHeaderName 设置CSRF头名称
func WithHeaderName(name string) func(*Config) {
	return func(c *Config) {
		c.HeaderName = name
	}
}

// WithFormField 设置表单字段名称
func WithFormField(field string) func(*Config) {
	return func(c *Config) {
		c.FormField = field
	}
}

// WithErrorHandler 设置自定义错误处理
func WithErrorHandler(handler func(ctx *mist.Context, err error)) func(*Config) {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

// WithIgnoreMethods 设置忽略的HTTP方法
func WithIgnoreMethods(methods []string) func(*Config) {
	return func(c *Config) {
		c.IgnoreMethods = methods
	}
}
