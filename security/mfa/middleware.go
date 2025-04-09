package mfa

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

const (
	// MFACookieName 用于标记MFA验证状态的Cookie名
	MFACookieName = "_mfa_validated"

	// MFASessionKey 用于在Session中存储MFA状态的键
	MFASessionKey = "_mfa_status"

	// DefaultValidationDuration MFA验证状态默认有效期
	DefaultValidationDuration = 12 * time.Hour
)

var (
	// ErrMFARequired 表示需要多因素验证
	ErrMFARequired = errors.New("需要多因素验证")

	// ErrInvalidMFACode 表示MFA验证码无效
	ErrInvalidMFACode = errors.New("无效的多因素验证码")
)

// ValidationStore 存储MFA验证状态的接口
type ValidationStore interface {
	// Validate 验证指定用户ID是否已完成MFA验证
	Validate(userID string) (bool, error)

	// Set 设置用户MFA验证状态
	Set(userID string, expiry time.Duration) error

	// Clear 清除用户MFA验证状态
	Clear(userID string) error
}

// MemoryStore 内存实现的MFA验证状态存储
type MemoryStore struct {
	validations map[string]int64
	mu          sync.RWMutex
}

// NewMemoryStore 创建新的内存验证状态存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		validations: make(map[string]int64),
	}
}

// Validate 验证用户MFA状态
func (s *MemoryStore) Validate(userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expiryTime, exists := s.validations[userID]
	if !exists {
		return false, nil
	}

	// 如果过期时间已到，则验证失败
	if expiryTime < time.Now().Unix() {
		return false, nil
	}

	return true, nil
}

// Set 设置用户MFA验证状态
func (s *MemoryStore) Set(userID string, expiry time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validations[userID] = time.Now().Add(expiry).Unix()
	return nil
}

// Clear 清除用户MFA验证状态
func (s *MemoryStore) Clear(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.validations, userID)
	return nil
}

// Config MFA中间件配置
type MiddlewareConfig struct {
	// Store MFA验证状态存储
	Store ValidationStore

	// GetUserID 从请求上下文中获取用户ID的函数
	GetUserID func(*mist.Context) (string, error)

	// ValidationDuration MFA验证有效期
	ValidationDuration time.Duration

	// RedirectURL 未验证时重定向的URL
	RedirectURL string

	// OnUnauthorized 未验证时的处理函数
	OnUnauthorized func(*mist.Context)
}

// New 创建新的MFA中间件
func NewMiddleware(options ...func(*MiddlewareConfig)) mist.Middleware {
	// 默认配置
	config := MiddlewareConfig{
		Store: NewMemoryStore(),
		GetUserID: func(ctx *mist.Context) (string, error) {
			// 默认从上下文中获取user_id
			if id, exists := ctx.Get("user_id"); exists {
				if userID, ok := id.(string); ok {
					return userID, nil
				}
			}
			return "", errors.New("无法获取用户ID")
		},
		ValidationDuration: DefaultValidationDuration,
		RedirectURL:        "/mfa/validate",
		OnUnauthorized: func(ctx *mist.Context) {
			// 默认重定向到MFA验证页面
			ctx.Header("Location", "/mfa/validate")
			ctx.AbortWithStatus(http.StatusFound)
		},
	}

	// 应用自定义选项
	for _, option := range options {
		option(&config)
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 获取用户ID
			userID, err := config.GetUserID(ctx)
			if err != nil {
				// 如果无法获取用户ID，视为未验证
				config.OnUnauthorized(ctx)
				return
			}

			// 检查是否已验证
			validated, err := config.Store.Validate(userID)
			if err != nil || !validated {
				config.OnUnauthorized(ctx)
				return
			}

			// 已验证，继续处理请求
			next(ctx)
		}
	}
}

// Validate 验证MFA代码
func Validate(ctx *mist.Context, userID, code string, totp *TOTP, store ValidationStore, duration time.Duration) error {
	// 验证TOTP代码
	if !totp.Validate(code) {
		return ErrInvalidMFACode
	}

	// 设置验证状态
	return store.Set(userID, duration)
}

// ClearValidation 清除MFA验证状态
func ClearValidation(userID string, store ValidationStore) error {
	return store.Clear(userID)
}

// 选项函数

// WithStore 设置验证状态存储
func WithStore(store ValidationStore) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.Store = store
	}
}

// WithGetUserID 设置获取用户ID的函数
func WithGetUserID(fn func(*mist.Context) (string, error)) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.GetUserID = fn
	}
}

// WithValidationDuration 设置验证有效期
func WithValidationDuration(duration time.Duration) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.ValidationDuration = duration
	}
}

// WithRedirectURL 设置重定向URL
func WithRedirectURL(url string) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.RedirectURL = url
	}
}

// WithUnauthorizedHandler 设置未授权处理函数
func WithUnauthorizedHandler(handler func(*mist.Context)) func(*MiddlewareConfig) {
	return func(c *MiddlewareConfig) {
		c.OnUnauthorized = handler
	}
}
