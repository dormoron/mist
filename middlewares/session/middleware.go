package session

import (
	"net/http"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/session"
	"github.com/dormoron/mist/session/cookie"
	"github.com/dormoron/mist/session/memory"
	"github.com/dormoron/mist/session/redis"
)

// MiddlewareBuilder 会话中间件构建器
type MiddlewareBuilder struct {
	manager        *session.Manager
	cookieName     string
	cookiePath     string
	cookieDomain   string
	cookieSecure   bool
	cookieHTTPOnly bool
	cookieSameSite http.SameSite
	maxAge         int
	sessionKey     string
}

// NewMemoryStore 创建基于内存的会话中间件
func NewMemoryStore(opts ...Option) (*MiddlewareBuilder, error) {
	store, err := memory.NewStore()
	if err != nil {
		return nil, err
	}

	builder := &MiddlewareBuilder{
		cookieName:     "mist_session",
		cookiePath:     "/",
		cookieSecure:   true,
		cookieHTTPOnly: true,
		cookieSameSite: http.SameSiteStrictMode,
		maxAge:         3600, // 1小时
		sessionKey:     "session",
	}

	manager, err := session.NewManager(store, builder.maxAge)
	if err != nil {
		return nil, err
	}

	// 创建Cookie传播器
	propagator := cookie.NewPropagator(builder.cookieName,
		cookie.WithPath(builder.cookiePath),
		cookie.WithMaxAge(builder.maxAge),
		cookie.WithSecure(builder.cookieSecure),
		cookie.WithHTTPOnly(builder.cookieHTTPOnly),
		cookie.WithSameSite(builder.cookieSameSite),
	)
	manager.Propagator = propagator

	builder.manager = manager

	// 应用选项
	for _, opt := range opts {
		opt(builder)
	}

	return builder, nil
}

// NewRedisStore 创建基于Redis的会话中间件
func NewRedisStore(addr string, password string, db int, keyPrefix string, opts ...Option) (*MiddlewareBuilder, error) {
	options := &redis.Options{
		Addr:      addr,
		Password:  password,
		DB:        db,
		KeyPrefix: keyPrefix,
	}

	store, err := redis.NewStore(options)
	if err != nil {
		return nil, err
	}

	builder := &MiddlewareBuilder{
		cookieName:     "mist_session",
		cookiePath:     "/",
		cookieSecure:   true,
		cookieHTTPOnly: true,
		cookieSameSite: http.SameSiteStrictMode,
		maxAge:         3600, // 1小时
		sessionKey:     "session",
	}

	manager, err := session.NewManager(store, builder.maxAge)
	if err != nil {
		return nil, err
	}

	// 创建Cookie传播器
	propagator := cookie.NewPropagator(builder.cookieName,
		cookie.WithPath(builder.cookiePath),
		cookie.WithMaxAge(builder.maxAge),
		cookie.WithSecure(builder.cookieSecure),
		cookie.WithHTTPOnly(builder.cookieHTTPOnly),
		cookie.WithSameSite(builder.cookieSameSite),
	)
	manager.Propagator = propagator

	builder.manager = manager

	// 应用选项
	for _, opt := range opts {
		opt(builder)
	}

	return builder, nil
}

// Option 会话中间件选项
type Option func(*MiddlewareBuilder)

// WithCookieName 设置cookie名称
func WithCookieName(name string) Option {
	return func(b *MiddlewareBuilder) {
		b.cookieName = name
	}
}

// WithCookiePath 设置cookie路径
func WithCookiePath(path string) Option {
	return func(b *MiddlewareBuilder) {
		b.cookiePath = path
	}
}

// WithCookieDomain 设置cookie域
func WithCookieDomain(domain string) Option {
	return func(b *MiddlewareBuilder) {
		b.cookieDomain = domain
	}
}

// WithCookieSecure 设置cookie是否只通过HTTPS发送
func WithCookieSecure(secure bool) Option {
	return func(b *MiddlewareBuilder) {
		b.cookieSecure = secure
	}
}

// WithCookieHTTPOnly 设置cookie是否禁止JavaScript访问
func WithCookieHTTPOnly(httpOnly bool) Option {
	return func(b *MiddlewareBuilder) {
		b.cookieHTTPOnly = httpOnly
	}
}

// WithCookieSameSite 设置cookie的SameSite属性
func WithCookieSameSite(sameSite http.SameSite) Option {
	return func(b *MiddlewareBuilder) {
		b.cookieSameSite = sameSite
	}
}

// WithMaxAge 设置会话最大有效期（秒）
func WithMaxAge(maxAge int) Option {
	return func(b *MiddlewareBuilder) {
		b.maxAge = maxAge
		if b.manager != nil {
			b.manager.SetMaxAge(maxAge)
		}
	}
}

// WithSessionKey 设置会话在上下文中的键名
func WithSessionKey(key string) Option {
	return func(b *MiddlewareBuilder) {
		b.sessionKey = key
	}
}

// Build 构建会话中间件
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			var sess session.Session
			var err error

			// 尝试从请求中获取会话ID
			sessID, err := b.manager.Propagator.Extract(ctx.Request)
			if err == nil && sessID != "" {
				// 尝试加载已有会话
				sess, err = b.manager.Get(ctx.Request.Context(), sessID)
				if err != nil {
					// 会话无效或过期，初始化新会话
					sess, err = b.manager.InitSession(ctx)
					if err != nil {
						mist.Error("初始化会话失败: %v", err)
						next(ctx)
						return
					}
				}
			} else {
				// 没有会话ID，初始化新会话
				sess, err = b.manager.InitSession(ctx)
				if err != nil {
					mist.Error("初始化会话失败: %v", err)
					next(ctx)
					return
				}
			}

			// 将会话存储在上下文中
			ctx.Set(b.sessionKey, sess)

			// 继续处理请求
			next(ctx)

			// 保存会话更改
			if err := sess.Save(); err != nil {
				mist.Error("保存会话失败: %v", err)
			}
		}
	}
}

// GetSession 从上下文中获取会话
func GetSession(ctx *mist.Context, key string) (session.Session, bool) {
	val, exists := ctx.Get(key)
	if !exists {
		return nil, false
	}

	sess, ok := val.(session.Session)
	return sess, ok
}

// DefaultGetSession 使用默认键名获取会话
func DefaultGetSession(ctx *mist.Context) (session.Session, bool) {
	return GetSession(ctx, "session")
}

// SessionCleanupTask 返回一个定期清理过期会话的函数
func SessionCleanupTask(manager *session.Manager, interval time.Duration) func() {
	return func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := manager.GC(); err != nil {
				mist.Error("清理过期会话失败: %v", err)
			}
		}
	}
}
