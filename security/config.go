package security

import (
	"net/http"
	"time"
)

// SecurityLevel 表示应用程序的安全级别
type SecurityLevel int

const (
	// LevelBasic 基本安全级别，适用于开发或不太敏感的应用
	LevelBasic SecurityLevel = iota
	// LevelIntermediate 中级安全级别，适用于一般Web应用
	LevelIntermediate
	// LevelStrict 严格安全级别，适用于处理敏感数据的应用
	LevelStrict
	// LevelCustom 自定义安全级别，使用用户指定的设置
	LevelCustom
)

// SecurityConfig 是安全模块的主配置结构
type SecurityConfig struct {
	// Level 安全级别
	Level SecurityLevel

	// Session 会话配置
	Session SessionConfig

	// CSRF 防跨站请求伪造配置
	CSRF CSRFConfig

	// RateLimit 请求限流配置
	RateLimit RateLimitConfig

	// Headers HTTP安全头配置
	Headers HeadersConfig

	// Auth 身份验证配置
	Auth AuthConfig

	// Password 密码策略配置
	Password PasswordConfig
}

// SessionConfig 会话配置
type SessionConfig struct {
	// Enabled 是否启用会话管理
	Enabled bool

	// Domain Cookie域名
	Domain string

	// Path Cookie路径
	Path string

	// MaxAge 会话最大存活时间
	MaxAge time.Duration

	// Secure 是否仅通过HTTPS发送Cookie
	Secure bool

	// HttpOnly 是否禁止JavaScript访问Cookie
	HttpOnly bool

	// SameSite Cookie的SameSite属性
	SameSite http.SameSite

	// AccessTokenExpiry 访问令牌过期时间
	AccessTokenExpiry time.Duration

	// RefreshTokenExpiry 刷新令牌过期时间
	RefreshTokenExpiry time.Duration

	// TokenHeader 令牌的HTTP头
	TokenHeader string

	// AccessTokenHeader 访问令牌的HTTP头
	AccessTokenHeader string

	// RefreshTokenHeader 刷新令牌的HTTP头
	RefreshTokenHeader string

	// IdleTimeout 会话闲置超时时间
	IdleTimeout time.Duration

	// AbsoluteTimeout 会话绝对过期时间（无论活动与否）
	AbsoluteTimeout time.Duration

	// EnableFingerprinting 启用会话指纹绑定
	EnableFingerprinting bool

	// RotateTokenOnValidation 每次认证成功后轮换会话令牌
	RotateTokenOnValidation bool

	// RequireReauthForSensitive 敏感操作需要重新验证
	RequireReauthForSensitive bool

	// ReauthTimeout 重新验证超时时间
	ReauthTimeout time.Duration
}

// CSRFConfig CSRF配置
type CSRFConfig struct {
	// Enabled 是否启用CSRF保护
	Enabled bool

	// TokenLength CSRF令牌长度
	TokenLength int

	// CookieName CSRF Cookie名称
	CookieName string

	// HeaderName CSRF HTTP头名称
	HeaderName string

	// FormField CSRF表单字段名称
	FormField string

	// IgnoreMethods 忽略的HTTP方法
	IgnoreMethods []string
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// Enabled 是否启用限流
	Enabled bool

	// Rate 每秒请求限制
	Rate float64

	// Burst 突发请求限制
	Burst int

	// EnableIPRateLimit 是否启用IP限流
	EnableIPRateLimit bool

	// UseRedisBackend 是否使用Redis后端
	UseRedisBackend bool
}

// HeadersConfig 安全HTTP头配置
type HeadersConfig struct {
	// EnableXSSProtection 是否启用XSS防护
	EnableXSSProtection bool

	// EnableContentTypeNosniff 是否启用内容类型嗅探保护
	EnableContentTypeNosniff bool

	// EnableXFrameOptions 是否启用X-Frame-Options
	EnableXFrameOptions bool

	// XFrameOptionsValue X-Frame-Options的值
	XFrameOptionsValue string

	// EnableHSTS 是否启用HSTS
	EnableHSTS bool

	// HSTSMaxAge HSTS最大存活时间
	HSTSMaxAge time.Duration

	// HSTSIncludeSubdomains 是否包含子域名
	HSTSIncludeSubdomains bool

	// HSTSPreload 是否启用预加载
	HSTSPreload bool

	// ContentSecurityPolicy 内容安全策略
	ContentSecurityPolicy string

	// EnablePermissionsPolicy 是否启用权限策略
	EnablePermissionsPolicy bool

	// PermissionsPolicy 权限策略内容
	PermissionsPolicy string

	// EnableCrossOriginPolicies 是否启用跨域策略
	EnableCrossOriginPolicies bool

	// CrossOriginEmbedderPolicy 跨域嵌入者策略
	CrossOriginEmbedderPolicy string

	// CrossOriginOpenerPolicy 跨域打开者策略
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy 跨域资源策略
	CrossOriginResourcePolicy string

	// EnableCacheControl 是否启用缓存控制
	EnableCacheControl bool

	// CacheControl 缓存控制内容
	CacheControl string

	// EnableReferrerPolicy 是否启用引用策略
	EnableReferrerPolicy bool

	// ReferrerPolicy 引用策略内容
	ReferrerPolicy string
}

// AuthConfig 身份验证配置
type AuthConfig struct {
	// JWTSecret JWT密钥
	JWTSecret string

	// EnableRefreshToken 是否启用刷新令牌
	EnableRefreshToken bool

	// MaxLoginAttempts 最大登录尝试次数
	MaxLoginAttempts int

	// LockoutDuration 锁定时长
	LockoutDuration time.Duration

	// AccountLockout 账户锁定策略
	AccountLockout AccountLockoutConfig
}

// AccountLockoutConfig 账户锁定配置
type AccountLockoutConfig struct {
	// Enabled 是否启用账户锁定
	Enabled bool

	// MaxAttempts 允许的最大失败尝试次数
	MaxAttempts int

	// LockoutDuration 锁定持续时间
	LockoutDuration time.Duration

	// ResetDuration 失败尝试记录重置时间
	ResetDuration time.Duration

	// IncludeIPInKey 是否在锁定键中包含IP地址
	IncludeIPInKey bool

	// CleanupInterval 清理过期锁定记录的间隔
	CleanupInterval time.Duration
}

// PasswordConfig 密码配置
type PasswordConfig struct {
	// MinLength 最小长度
	MinLength int

	// RequireUppercase 是否要求大写字母
	RequireUppercase bool

	// RequireLowercase 是否要求小写字母
	RequireLowercase bool

	// RequireDigits 是否要求数字
	RequireDigits bool

	// RequireSpecialChars 是否要求特殊字符
	RequireSpecialChars bool

	// MaxAge 密码最长使用时间
	MaxAge time.Duration

	// PreventReuseCount 禁止重复使用最近密码的数量
	PreventReuseCount int
}

// 使用工厂函数生成不同安全级别的默认配置

// DefaultSecurityConfig 返回基于指定安全级别的默认配置
func DefaultSecurityConfig(level SecurityLevel) SecurityConfig {
	switch level {
	case LevelBasic:
		return basicSecurityConfig()
	case LevelIntermediate:
		return intermediateSecurityConfig()
	case LevelStrict:
		return strictSecurityConfig()
	default:
		return basicSecurityConfig()
	}
}

// basicSecurityConfig 返回基本安全级别的配置
func basicSecurityConfig() SecurityConfig {
	return SecurityConfig{
		Level: LevelBasic,
		Session: SessionConfig{
			Enabled:                   true,
			Domain:                    "",
			Path:                      "/",
			MaxAge:                    24 * time.Hour,
			Secure:                    false,
			HttpOnly:                  true,
			SameSite:                  http.SameSiteLaxMode,
			AccessTokenExpiry:         15 * time.Minute,
			RefreshTokenExpiry:        7 * 24 * time.Hour,
			TokenHeader:               "Authorization",
			AccessTokenHeader:         "X-Access-Token",
			RefreshTokenHeader:        "X-Refresh-Token",
			IdleTimeout:               30 * time.Minute,
			AbsoluteTimeout:           24 * time.Hour,
			EnableFingerprinting:      false,
			RotateTokenOnValidation:   false,
			RequireReauthForSensitive: false,
			ReauthTimeout:             10 * time.Minute,
		},
		CSRF: CSRFConfig{
			Enabled:       false,
			TokenLength:   32,
			CookieName:    "csrf_token",
			HeaderName:    "X-CSRF-Token",
			FormField:     "csrf_token",
			IgnoreMethods: []string{"GET", "HEAD", "OPTIONS"},
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			Rate:              100,
			Burst:             200,
			EnableIPRateLimit: false,
			UseRedisBackend:   false,
		},
		Headers: HeadersConfig{
			EnableXSSProtection:       true,
			EnableContentTypeNosniff:  true,
			EnableXFrameOptions:       true,
			XFrameOptionsValue:        "SAMEORIGIN",
			EnableHSTS:                false,
			HSTSMaxAge:                0,
			HSTSIncludeSubdomains:     false,
			HSTSPreload:               false,
			ContentSecurityPolicy:     "",
			EnablePermissionsPolicy:   false,
			PermissionsPolicy:         "",
			EnableCrossOriginPolicies: false,
			CrossOriginEmbedderPolicy: "",
			CrossOriginOpenerPolicy:   "",
			CrossOriginResourcePolicy: "",
			EnableCacheControl:        false,
			CacheControl:              "",
			EnableReferrerPolicy:      false,
			ReferrerPolicy:            "",
		},
		Auth: AuthConfig{
			JWTSecret:          "default_jwt_secret_change_me_in_production",
			EnableRefreshToken: true,
			MaxLoginAttempts:   0,
			LockoutDuration:    0,
			AccountLockout: AccountLockoutConfig{
				Enabled:         false,
				MaxAttempts:     5,
				LockoutDuration: 15 * time.Minute,
				ResetDuration:   24 * time.Hour,
				IncludeIPInKey:  true,
				CleanupInterval: 30 * time.Minute,
			},
		},
		Password: PasswordConfig{
			MinLength:           8,
			RequireUppercase:    false,
			RequireLowercase:    false,
			RequireDigits:       false,
			RequireSpecialChars: false,
			MaxAge:              0,
			PreventReuseCount:   0,
		},
	}
}

// intermediateSecurityConfig 返回中级安全配置
func intermediateSecurityConfig() SecurityConfig {
	config := basicSecurityConfig()
	config.Level = LevelIntermediate

	// 会话配置增强
	config.Session.Secure = true
	config.Session.IdleTimeout = 15 * time.Minute
	config.Session.AbsoluteTimeout = 12 * time.Hour
	config.Session.EnableFingerprinting = true
	config.Session.RotateTokenOnValidation = true

	// 启用CSRF保护
	config.CSRF.Enabled = true

	// 启用适度的限流
	config.RateLimit.Enabled = true
	config.RateLimit.EnableIPRateLimit = true

	// 增强HTTP头
	config.Headers.EnableHSTS = true
	config.Headers.HSTSMaxAge = 180 * 24 * time.Hour
	config.Headers.HSTSIncludeSubdomains = true
	config.Headers.ContentSecurityPolicy = "default-src 'self'"
	config.Headers.EnablePermissionsPolicy = true
	config.Headers.PermissionsPolicy = "geolocation=self"
	config.Headers.EnableReferrerPolicy = true
	config.Headers.ReferrerPolicy = "strict-origin-when-cross-origin"

	// 启用账户锁定
	config.Auth.AccountLockout.Enabled = true

	// 增强密码安全性
	config.Password.RequireUppercase = true
	config.Password.RequireLowercase = true
	config.Password.RequireDigits = true
	config.Password.MaxAge = 90 * 24 * time.Hour
	config.Password.PreventReuseCount = 3

	return config
}

// strictSecurityConfig 返回严格安全配置
func strictSecurityConfig() SecurityConfig {
	config := intermediateSecurityConfig()
	config.Level = LevelStrict

	// 会话安全强化
	config.Session.MaxAge = 8 * time.Hour
	config.Session.AccessTokenExpiry = 5 * time.Minute
	config.Session.IdleTimeout = 10 * time.Minute
	config.Session.RequireReauthForSensitive = true
	config.Session.ReauthTimeout = 5 * time.Minute

	// 限流强化
	config.RateLimit.Rate = 30
	config.RateLimit.Burst = 60

	// 强化HTTP头
	config.Headers.HSTSPreload = true
	config.Headers.ContentSecurityPolicy = "default-src 'self'; script-src 'self'; object-src 'none'; img-src 'self' data:; style-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'"
	config.Headers.EnableCrossOriginPolicies = true
	config.Headers.CrossOriginEmbedderPolicy = "require-corp"
	config.Headers.CrossOriginOpenerPolicy = "same-origin"
	config.Headers.CrossOriginResourcePolicy = "same-origin"
	config.Headers.EnableCacheControl = true
	config.Headers.CacheControl = "no-store, max-age=0"

	// 强化账户锁定策略
	config.Auth.AccountLockout.MaxAttempts = 3
	config.Auth.AccountLockout.LockoutDuration = 30 * time.Minute

	// 强化密码策略
	config.Password.MinLength = 12
	config.Password.RequireSpecialChars = true
	config.Password.MaxAge = 60 * 24 * time.Hour
	config.Password.PreventReuseCount = 10

	return config
}

// 全局配置实例和操作函数

var (
	// 全局配置实例
	globalConfig = DefaultSecurityConfig(LevelIntermediate)
)

// SetSecurityConfig 设置全局安全配置
func SetSecurityConfig(config SecurityConfig) {
	globalConfig = config
}

// GetSecurityConfig 获取当前全局安全配置
func GetSecurityConfig() SecurityConfig {
	return globalConfig
}

// SetSecurityLevel 设置全局安全级别
func SetSecurityLevel(level SecurityLevel) {
	globalConfig = DefaultSecurityConfig(level)
}

// GetSecurityLevel 获取当前安全级别
func GetSecurityLevel() SecurityLevel {
	return globalConfig.Level
}
