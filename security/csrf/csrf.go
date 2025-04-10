package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

var (
	// ErrTokenNotFound 表示请求中未找到CSRF令牌
	ErrTokenNotFound = errors.New("CSRF token not found in request")
	// ErrTokenInvalid 表示请求中的CSRF令牌无效
	ErrTokenInvalid = errors.New("CSRF token invalid")
	// ErrTokenExpired 表示CSRF令牌已过期
	ErrTokenExpired = errors.New("CSRF token expired")
)

// TokenMode 定义CSRF令牌验证模式
type TokenMode int

const (
	// StandardMode 标准模式 - 令牌存储在Cookie中并与请求中的令牌比较
	StandardMode TokenMode = iota
	// DoubleSubmitMode 双重提交模式 - 令牌存储在Cookie中并用于签名请求中的令牌
	DoubleSubmitMode
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
	// TokenMode 令牌验证模式，默认为StandardMode
	TokenMode TokenMode
	// TokenTTL 令牌有效时间，默认为2小时，设置为0则永不过期
	TokenTTL time.Duration
	// TokenRefreshInterval 令牌刷新间隔，默认为30分钟，设置为0则不刷新
	TokenRefreshInterval time.Duration
}

// TokenData 存储令牌相关数据
type TokenData struct {
	Token      string    // 令牌值
	Created    time.Time // 创建时间
	LastAccess time.Time // 最后访问时间
}

// csrfProtection 实现CSRF保护
type csrfProtection struct {
	config    Config
	tokenData map[string]*TokenData // 用于跟踪令牌状态
	mutex     sync.RWMutex
}

// New 创建新的CSRF保护中间件
func New(options ...func(*Config)) mist.Middleware {
	// 设置默认配置
	config := Config{
		TokenLength:          32,
		CookieName:           "_csrf",
		CookiePath:           "/",
		CookieMaxAge:         24 * time.Hour,
		CookieHTTPOnly:       true,
		CookieSameSite:       http.SameSiteLaxMode,
		HeaderName:           "X-CSRF-Token",
		FormField:            "csrf_token",
		IgnoreMethods:        []string{"GET", "HEAD", "OPTIONS", "TRACE"},
		TokenMode:            StandardMode,
		TokenTTL:             2 * time.Hour,
		TokenRefreshInterval: 30 * time.Minute,
		ErrorHandler: func(ctx *mist.Context, err error) {
			ctx.AbortWithStatus(http.StatusForbidden)
		},
	}

	// 应用自定义选项
	for _, option := range options {
		option(&config)
	}

	protection := &csrfProtection{
		config:    config,
		tokenData: make(map[string]*TokenData),
	}

	// 启动清理过期令牌的goroutine
	go protection.cleanupExpiredTokens()

	return protection.middleware
}

// 清理过期令牌
func (c *csrfProtection) cleanupExpiredTokens() {
	// 如果TokenTTL设置为0，则令牌不会过期，无需清理
	if c.config.TokenTTL == 0 {
		return
	}

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for token, data := range c.tokenData {
			// 如果令牌创建时间超过TTL，则删除
			if c.config.TokenTTL > 0 && data.Created.Add(c.config.TokenTTL).Before(now) {
				delete(c.tokenData, token)
			}
		}
		c.mutex.Unlock()
	}
}

// 生成CSRF令牌
func (c *csrfProtection) generateToken() (string, error) {
	bytes := make([]byte, c.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	token := base64.StdEncoding.EncodeToString(bytes)

	// 记录令牌数据
	c.mutex.Lock()
	c.tokenData[token] = &TokenData{
		Token:      token,
		Created:    time.Now(),
		LastAccess: time.Now(),
	}
	c.mutex.Unlock()

	return token, nil
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

// 检查令牌是否有效
func (c *csrfProtection) isTokenValid(token string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	data, exists := c.tokenData[token]
	if !exists {
		return false
	}

	// 如果设置了令牌TTL并已过期
	if c.config.TokenTTL > 0 && data.Created.Add(c.config.TokenTTL).Before(time.Now()) {
		return false
	}

	return true
}

// 更新令牌最后访问时间
func (c *csrfProtection) updateTokenAccess(token string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if data, exists := c.tokenData[token]; exists {
		data.LastAccess = time.Now()
	}
}

// 检查令牌是否需要刷新
func (c *csrfProtection) needsRefresh(token string) bool {
	// 如果刷新间隔为0，则不需要刷新
	if c.config.TokenRefreshInterval == 0 {
		return false
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	data, exists := c.tokenData[token]
	if !exists {
		return true
	}

	// 如果自上次访问以来已经过了刷新间隔，则需要刷新
	return data.LastAccess.Add(c.config.TokenRefreshInterval).Before(time.Now())
}

// CSRF中间件
func (c *csrfProtection) middleware(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// 如果方法在忽略列表中，则跳过保护
		if c.isMethodIgnored(ctx.Request.Method) {
			// 为GET请求（通常是页面加载）生成令牌
			if ctx.Request.Method == "GET" {
				// 尝试从cookie获取现有令牌
				cookie, err := ctx.Request.Cookie(c.config.CookieName)
				tokenExists := err == nil && cookie.Value != ""

				// 决定是否需要生成新令牌
				needNewToken := !tokenExists

				// 如果令牌存在，检查是否需要刷新
				if tokenExists && c.needsRefresh(cookie.Value) {
					needNewToken = true
				}

				// 如果令牌存在但已过期，也需要新令牌
				if tokenExists && !c.isTokenValid(cookie.Value) {
					needNewToken = true
				}

				if needNewToken {
					token, err := c.generateToken()
					if err != nil {
						c.config.ErrorHandler(ctx, fmt.Errorf("failed to generate CSRF token: %w", err))
						return
					}
					c.setCSRFCookie(ctx, token)
					// 设置上下文中的CSRF令牌，以便视图可以访问
					ctx.Set("csrf_token", token)
				} else {
					// 更新令牌的最后访问时间
					c.updateTokenAccess(cookie.Value)
					// 设置上下文中的CSRF令牌为已有的令牌
					ctx.Set("csrf_token", cookie.Value)
				}
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

		// 检查令牌是否有效（未过期）
		if !c.isTokenValid(cookieToken) {
			c.config.ErrorHandler(ctx, ErrTokenExpired)
			return
		}

		// 获取请求中的令牌
		requestToken := c.getTokenFromRequest(ctx)
		if requestToken == "" {
			c.config.ErrorHandler(ctx, ErrTokenNotFound)
			return
		}

		// 根据不同的验证模式进行验证
		var isValid bool

		switch c.config.TokenMode {
		case StandardMode:
			// 标准模式: 直接比较令牌
			isValid = requestToken == cookieToken
		case DoubleSubmitMode:
			// 双重提交模式: 带签名的验证
			// 这里是一个简化的实现，实际生产环境中可能需要更复杂的签名机制
			parts := strings.Split(requestToken, ".")
			if len(parts) != 2 {
				isValid = false
			} else {
				// 使用cookie令牌作为密钥进行签名验证
				signature := generateSignature(parts[0], cookieToken)
				isValid = signature == parts[1]
			}
		default:
			isValid = requestToken == cookieToken
		}

		if !isValid {
			c.config.ErrorHandler(ctx, ErrTokenInvalid)
			return
		}

		// 更新令牌的最后访问时间
		c.updateTokenAccess(cookieToken)

		// 验证通过，继续处理请求
		next(ctx)
	}
}

// 为双重提交模式生成签名
func generateSignature(token, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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

// WithCookieSameSite 设置cookie的SameSite属性
func WithCookieSameSite(sameSite http.SameSite) func(*Config) {
	return func(c *Config) {
		c.CookieSameSite = sameSite
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

// WithTokenMode 设置令牌验证模式
func WithTokenMode(mode TokenMode) func(*Config) {
	return func(c *Config) {
		c.TokenMode = mode
	}
}

// WithTokenTTL 设置令牌有效时间
func WithTokenTTL(ttl time.Duration) func(*Config) {
	return func(c *Config) {
		c.TokenTTL = ttl
	}
}

// WithTokenRefreshInterval 设置令牌刷新间隔
func WithTokenRefreshInterval(interval time.Duration) func(*Config) {
	return func(c *Config) {
		c.TokenRefreshInterval = interval
	}
}
