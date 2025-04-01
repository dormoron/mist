package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

const (
	// 默认令牌长度
	defaultTokenLength = 32

	// 默认令牌有效期
	defaultTokenExpiry = 1 * time.Hour

	// CSRF令牌cookie名称
	defaultCookieName = "csrf_token"

	// CSRF令牌头名称
	defaultHeaderName = "X-CSRF-Token"

	// CSRF令牌表单字段名称
	defaultFormField = "_csrf"
)

var (
	ErrNoToken      = errors.New("没有提供CSRF令牌")
	ErrInvalidToken = errors.New("无效的CSRF令牌")
)

// Options CSRF中间件选项
type Options struct {
	// TokenLength 令牌长度
	TokenLength int

	// TokenExpiry 令牌有效期
	TokenExpiry time.Duration

	// CookieName CSRF令牌cookie名称
	CookieName string

	// HeaderName CSRF令牌头名称
	HeaderName string

	// FormField CSRF令牌表单字段名称
	FormField string

	// CookiePath Cookie路径
	CookiePath string

	// CookieDomain Cookie域
	CookieDomain string

	// CookieSecure 是否只通过HTTPS发送Cookie
	CookieSecure bool

	// CookieHTTPOnly 是否禁止JavaScript访问Cookie
	CookieHTTPOnly bool

	// CookieSameSite Cookie的SameSite属性
	CookieSameSite http.SameSite

	// TrustedOrigins 可信来源列表
	TrustedOrigins []string

	// IgnoreMethods 忽略的HTTP方法列表
	IgnoreMethods []string

	// ErrorHandler 错误处理函数
	ErrorHandler func(*mist.Context, error)
}

// DefaultOptions 返回默认CSRF中间件选项
func DefaultOptions() Options {
	return Options{
		TokenLength:    defaultTokenLength,
		TokenExpiry:    defaultTokenExpiry,
		CookieName:     defaultCookieName,
		HeaderName:     defaultHeaderName,
		FormField:      defaultFormField,
		CookiePath:     "/",
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteStrictMode,
		IgnoreMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace},
		ErrorHandler:   defaultErrorHandler,
	}
}

// 默认错误处理函数
func defaultErrorHandler(ctx *mist.Context, err error) {
	ctx.AbortWithStatus(http.StatusForbidden)
	ctx.RespData = []byte("CSRF令牌验证失败")
}

// CSRF中间件实现
type csrfMiddleware struct {
	options Options
	tokens  sync.Map // 用于存储和验证令牌
}

// NewMiddleware 创建一个新的CSRF中间件
func NewMiddleware(opts Options) mist.Middleware {
	if opts.TokenLength == 0 {
		opts.TokenLength = defaultTokenLength
	}
	if opts.TokenExpiry == 0 {
		opts.TokenExpiry = defaultTokenExpiry
	}
	if opts.CookieName == "" {
		opts.CookieName = defaultCookieName
	}
	if opts.HeaderName == "" {
		opts.HeaderName = defaultHeaderName
	}
	if opts.FormField == "" {
		opts.FormField = defaultFormField
	}
	if opts.CookiePath == "" {
		opts.CookiePath = "/"
	}
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultErrorHandler
	}
	if len(opts.IgnoreMethods) == 0 {
		opts.IgnoreMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	}

	middleware := &csrfMiddleware{
		options: opts,
	}

	return middleware.middleware
}

// WithDefaultCSRF 使用默认选项创建CSRF中间件
func WithDefaultCSRF() mist.Middleware {
	return NewMiddleware(DefaultOptions())
}

// middleware 中间件函数
func (m *csrfMiddleware) middleware(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// 检查请求是否应该被跳过CSRF验证
		if m.shouldSkip(ctx) {
			next(ctx)
			return
		}

		// 获取令牌，如果不存在则创建一个新的
		token := m.getTokenFromCookie(ctx)
		if token == "" {
			token = m.generateToken()
			m.saveToken(token)
			m.setTokenCookie(ctx, token)
		}

		// 对于非GET等方法，验证令牌
		if !m.isIgnoredMethod(ctx.Request.Method) {
			clientToken := m.getTokenFromRequest(ctx)
			if clientToken == "" {
				m.options.ErrorHandler(ctx, ErrNoToken)
				return
			}

			if !m.validateToken(clientToken) {
				m.options.ErrorHandler(ctx, ErrInvalidToken)
				return
			}
		}

		// 将令牌设置在上下文中，以便模板使用
		ctx.Set("csrf_token", token)

		// 设置CSRF头，允许客户端获取令牌
		ctx.Header(m.options.HeaderName, token)

		// 继续处理请求
		next(ctx)
	}
}

// shouldSkip 检查是否应该跳过CSRF验证
func (m *csrfMiddleware) shouldSkip(ctx *mist.Context) bool {
	// 如果是被忽略的方法，仍然生成和设置令牌，但不验证
	if m.isIgnoredMethod(ctx.Request.Method) {
		return false
	}

	// 检查来源是否受信任
	if len(m.options.TrustedOrigins) > 0 {
		origin := ctx.Request.Header.Get("Origin")
		referer := ctx.Request.Header.Get("Referer")

		// 如果有来源头，检查是否在信任列表中
		if origin != "" {
			for _, trusted := range m.options.TrustedOrigins {
				if strings.HasPrefix(origin, trusted) {
					return true
				}
			}
		}

		// 如果有referer头，检查是否在信任列表中
		if referer != "" {
			for _, trusted := range m.options.TrustedOrigins {
				if strings.HasPrefix(referer, trusted) {
					return true
				}
			}
		}
	}

	return false
}

// isIgnoredMethod 检查方法是否在忽略列表中
func (m *csrfMiddleware) isIgnoredMethod(method string) bool {
	for _, ignored := range m.options.IgnoreMethods {
		if method == ignored {
			return true
		}
	}
	return false
}

// generateToken 生成一个新的CSRF令牌
func (m *csrfMiddleware) generateToken() string {
	tokenBytes := make([]byte, m.options.TokenLength)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		// 如果随机数生成失败，使用时间戳和一些随机字节作为备选
		tokenBytes = []byte(time.Now().String())
	}
	return base64.StdEncoding.EncodeToString(tokenBytes)
}

// saveToken 保存令牌到存储中
func (m *csrfMiddleware) saveToken(token string) {
	m.tokens.Store(token, time.Now().Add(m.options.TokenExpiry))
}

// validateToken 验证令牌是否有效
func (m *csrfMiddleware) validateToken(token string) bool {
	if expiry, exists := m.tokens.Load(token); exists {
		if expiry.(time.Time).After(time.Now()) {
			return true
		}
		// 令牌已过期，删除它
		m.tokens.Delete(token)
	}
	return false
}

// getTokenFromCookie 从cookie中获取令牌
func (m *csrfMiddleware) getTokenFromCookie(ctx *mist.Context) string {
	cookie, err := ctx.Request.Cookie(m.options.CookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

// setTokenCookie 设置令牌到cookie
func (m *csrfMiddleware) setTokenCookie(ctx *mist.Context, token string) {
	cookie := &http.Cookie{
		Name:     m.options.CookieName,
		Value:    token,
		Path:     m.options.CookiePath,
		Domain:   m.options.CookieDomain,
		Expires:  time.Now().Add(m.options.TokenExpiry),
		Secure:   m.options.CookieSecure,
		HttpOnly: m.options.CookieHTTPOnly,
		SameSite: m.options.CookieSameSite,
	}
	ctx.SetCookie(cookie)
}

// getTokenFromRequest 从请求中获取令牌
func (m *csrfMiddleware) getTokenFromRequest(ctx *mist.Context) string {
	// 首先从头部获取
	token := ctx.Request.Header.Get(m.options.HeaderName)
	if token != "" {
		return token
	}

	// 然后从表单字段获取
	if ctx.Request.Method == http.MethodPost {
		if err := ctx.Request.ParseForm(); err == nil {
			token = ctx.Request.PostForm.Get(m.options.FormField)
			if token != "" {
				return token
			}
		}
	}

	// 最后从查询参数获取
	token = ctx.Request.URL.Query().Get(m.options.FormField)
	return token
}
