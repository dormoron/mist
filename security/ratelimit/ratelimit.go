package ratelimit

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

var (
	// ErrTooManyRequests 表示请求超过了限流阈值
	ErrTooManyRequests = errors.New("too many requests")
)

// Limiter 接口定义了限流器的基本操作
type Limiter interface {
	// Allow 检查特定键是否允许通过限流
	Allow(key string) (bool, error)
	// Reset 重置特定键的限流状态
	Reset(key string) error
}

// Config 限流配置
type Config struct {
	// 每秒允许的请求数
	Rate float64
	// 允许的突发请求数
	Burst int
	// 键提取器，从Context中提取限流键
	KeyExtractor func(*mist.Context) string
	// 自定义错误处理
	ErrorHandler func(*mist.Context, error)
	// 启用客户端IP限流
	EnableIPRateLimit bool
	// 使用真实IP（X-Forwarded-For或X-Real-IP）
	UseRealIP bool
	// 限流白名单，这些键不会被限流
	Whitelist []string
}

// TokenBucket 令牌桶实现
type tokenBucket struct {
	rate       float64   // 每秒填充的令牌数
	capacity   int       // 桶容量
	tokens     float64   // 当前令牌数
	lastAccess time.Time // 最后访问时间
}

// 创建新的令牌桶
func newTokenBucket(rate float64, capacity int) *tokenBucket {
	return &tokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastAccess: time.Now(),
	}
}

// 尝试获取令牌
func (tb *tokenBucket) getToken() bool {
	now := time.Now()
	// 计算从上次访问到现在应该添加的令牌数
	elapsed := now.Sub(tb.lastAccess).Seconds()
	tb.lastAccess = now

	// 添加新令牌（不超过桶容量）
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.rate)

	if tb.tokens < 1.0 {
		return false
	}

	// 消耗一个令牌
	tb.tokens--
	return true
}

// 令牌桶限流器
type memoryLimiter struct {
	limiters map[string]*tokenBucket
	mu       sync.RWMutex
	rate     float64
	burst    int
}

// NewMemoryLimiter 创建一个基于内存的限流器
func NewMemoryLimiter(r float64, b int) Limiter {
	return &memoryLimiter{
		limiters: make(map[string]*tokenBucket),
		rate:     r,
		burst:    b,
	}
}

// Allow 检查请求是否允许通过
func (m *memoryLimiter) Allow(key string) (bool, error) {
	m.mu.RLock()
	limiter, exists := m.limiters[key]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		// 创建新的限流器
		if limiter, exists = m.limiters[key]; !exists {
			limiter = newTokenBucket(m.rate, m.burst)
			m.limiters[key] = limiter
		}
		m.mu.Unlock()
	}

	return limiter.getToken(), nil
}

// Reset 重置限流器
func (m *memoryLimiter) Reset(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.limiters, key)
	return nil
}

// 限流中间件
type middleware struct {
	limiter      Limiter
	config       Config
	whitelistMap map[string]struct{}
}

// New 创建一个新的限流中间件
func New(limiter Limiter, options ...func(*Config)) mist.Middleware {
	config := Config{
		Rate:              10, // 默认每秒10个请求
		Burst:             20, // 默认突发20个请求
		EnableIPRateLimit: true,
		KeyExtractor: func(ctx *mist.Context) string {
			// 默认使用IP作为键
			ip := ctx.ClientIP()
			if ip == "" {
				ip = "unknown"
			}
			return ip
		},
		ErrorHandler: func(ctx *mist.Context, err error) {
			ctx.AbortWithStatus(http.StatusTooManyRequests)
		},
	}

	for _, option := range options {
		option(&config)
	}

	// 构建白名单映射
	whitelistMap := make(map[string]struct{}, len(config.Whitelist))
	for _, key := range config.Whitelist {
		whitelistMap[key] = struct{}{}
	}

	return (&middleware{
		limiter:      limiter,
		config:       config,
		whitelistMap: whitelistMap,
	}).Handle
}

// Handle 实现限流中间件
func (m *middleware) Handle(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// 提取限流键
		key := m.config.KeyExtractor(ctx)

		// 检查白名单
		if _, exists := m.whitelistMap[key]; exists {
			next(ctx)
			return
		}

		// 检查是否允许请求
		allowed, err := m.limiter.Allow(key)
		if err != nil {
			m.config.ErrorHandler(ctx, err)
			return
		}

		if !allowed {
			m.config.ErrorHandler(ctx, ErrTooManyRequests)
			return
		}

		// 允许请求通过
		next(ctx)
	}
}

// helper function that works similar to math.Min for float64
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// 选项函数

// WithRate 设置限流速率
func WithRate(r float64) func(*Config) {
	return func(c *Config) {
		c.Rate = r
	}
}

// WithBurst 设置突发请求数量
func WithBurst(b int) func(*Config) {
	return func(c *Config) {
		c.Burst = b
	}
}

// WithKeyExtractor 设置键提取器
func WithKeyExtractor(fn func(*mist.Context) string) func(*Config) {
	return func(c *Config) {
		c.KeyExtractor = fn
	}
}

// WithErrorHandler 设置错误处理器
func WithErrorHandler(fn func(*mist.Context, error)) func(*Config) {
	return func(c *Config) {
		c.ErrorHandler = fn
	}
}

// WithWhitelist 设置白名单
func WithWhitelist(whitelist []string) func(*Config) {
	return func(c *Config) {
		c.Whitelist = whitelist
	}
}

// 提供一些常用的键提取器

// IPKeyExtractor 基于客户端IP的键提取器
func IPKeyExtractor(ctx *mist.Context) string {
	return ctx.ClientIP()
}

// PathKeyExtractor 基于请求路径的键提取器
func PathKeyExtractor(ctx *mist.Context) string {
	return ctx.Request.URL.Path
}

// UserIDKeyExtractor 基于用户ID的键提取器
func UserIDKeyExtractor(ctx *mist.Context) string {
	// 尝试从上下文中获取用户ID
	if userID, exists := ctx.Get("user_id"); exists {
		return userID.(string)
	}
	return "anonymous"
}

// CombinedKeyExtractor 组合多个键提取器的结果
func CombinedKeyExtractor(extractors ...func(*mist.Context) string) func(*mist.Context) string {
	return func(ctx *mist.Context) string {
		var key string
		for _, extractor := range extractors {
			key += extractor(ctx) + ":"
		}
		return key
	}
}

// RedisLimiter 使用Redis作为存储后端的限流器接口
// 这仅是接口定义，实际实现可以根据具体Redis客户端库实现
type RedisLimiter interface {
	Limiter
	// 设置键的过期时间
	SetExpiry(key string, ttl time.Duration) error
	// 使用的Redis键前缀
	Prefix() string
}

// 提供更多帮助工具

// ParseIPNetwork 将CIDR字符串解析为IP网络
func ParseIPNetwork(cidr string) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(cidr)
	return network, err
}

// IPInRangeKeyExtractor 基于IP范围的键提取器
func IPInRangeKeyExtractor(networks []*net.IPNet) func(*mist.Context) string {
	return func(ctx *mist.Context) string {
		ip := net.ParseIP(ctx.ClientIP())
		if ip == nil {
			return "unknown"
		}

		for _, network := range networks {
			if network.Contains(ip) {
				// 如果IP在某个网络范围内，使用网络作为键
				return network.String()
			}
		}

		// 如果IP不在任何网络范围内，返回IP本身
		return ip.String()
	}
}
