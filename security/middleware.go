package security

import (
	"context"
	"net/http"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/auth"
	"github.com/dormoron/mist/session"
)

// Middleware 创建全局安全中间件
func Middleware() mist.HandleFunc {
	return func(ctx *mist.Context) {
		config := GetSecurityConfig()

		// 添加安全HTTP头
		if config.Headers.EnableXSSProtection {
			ctx.Header("X-XSS-Protection", "1; mode=block")
		}

		if config.Headers.EnableContentTypeNosniff {
			ctx.Header("X-Content-Type-Options", "nosniff")
		}

		if config.Headers.EnableXFrameOptions {
			ctx.Header("X-Frame-Options", config.Headers.XFrameOptionsValue)
		}

		if config.Headers.EnableHSTS {
			value := "max-age=" + config.Headers.HSTSMaxAge.String()
			if config.Headers.HSTSIncludeSubdomains {
				value += "; includeSubdomains"
			}
			if config.Headers.HSTSPreload {
				value += "; preload"
			}
			ctx.Header("Strict-Transport-Security", value)
		}

		if config.Headers.ContentSecurityPolicy != "" {
			ctx.Header("Content-Security-Policy", config.Headers.ContentSecurityPolicy)
		}

		// 添加其他增强的安全头
		if config.Headers.EnablePermissionsPolicy && config.Headers.PermissionsPolicy != "" {
			ctx.Header("Permissions-Policy", config.Headers.PermissionsPolicy)
		}

		if config.Headers.EnableReferrerPolicy && config.Headers.ReferrerPolicy != "" {
			ctx.Header("Referrer-Policy", config.Headers.ReferrerPolicy)
		}

		if config.Headers.EnableCacheControl && config.Headers.CacheControl != "" {
			ctx.Header("Cache-Control", config.Headers.CacheControl)
		}

		if config.Headers.EnableCrossOriginPolicies {
			if config.Headers.CrossOriginEmbedderPolicy != "" {
				ctx.Header("Cross-Origin-Embedder-Policy", config.Headers.CrossOriginEmbedderPolicy)
			}
			if config.Headers.CrossOriginOpenerPolicy != "" {
				ctx.Header("Cross-Origin-Opener-Policy", config.Headers.CrossOriginOpenerPolicy)
			}
			if config.Headers.CrossOriginResourcePolicy != "" {
				ctx.Header("Cross-Origin-Resource-Policy", config.Headers.CrossOriginResourcePolicy)
			}
		}
	}
}

// SessionWithSecurityMiddleware 创建增强的安全会话中间件
func SessionWithSecurityMiddleware(manager *session.Manager) mist.HandleFunc {
	return func(ctx *mist.Context) {
		config := GetSecurityConfig()

		// 创建会话安全选项
		options := &session.SessionSecurityOptions{
			EnableSameSite:            config.Session.SameSite != http.SameSiteNoneMode,
			SameSiteMode:              config.Session.SameSite,
			SecureOnly:                config.Session.Secure,
			HttpOnly:                  config.Session.HttpOnly,
			EnableFingerprinting:      config.Session.EnableFingerprinting,
			RotateTokenOnValidation:   config.Session.RotateTokenOnValidation,
			AbsoluteTimeout:           config.Session.AbsoluteTimeout,
			IdleTimeout:               config.Session.IdleTimeout,
			RenewTimeout:              config.Session.MaxAge / 6, // 默认为最大时间的1/6
			RequireReauthForSensitive: config.Session.RequireReauthForSensitive,
			ReauthTimeout:             config.Session.ReauthTimeout,
		}

		// 应用会话安全设置
		manager.SetSecurityOptions(options)

		// 尝试获取现有会话
		_, err := manager.GetSession(ctx)

		// 如果会话不存在或已过期，创建新会话
		if err != nil {
			_, err = manager.InitSessionWithSecurity(ctx, options)
			if err != nil {
				ctx.RespStatusCode = http.StatusInternalServerError
				ctx.RespData = []byte("无法初始化会话")
				return
			}
		} else {
			// 刷新现有会话，应用安全设置
			if err := manager.RefreshSessionWithSecurity(ctx, options); err != nil {
				// 可能是会话被劫持或过期
				ctx.RespStatusCode = http.StatusUnauthorized
				ctx.RespData = []byte("会话无效")
				return
			}
		}
	}
}

// AccountLockoutMiddleware 创建账户锁定中间件
func AccountLockoutMiddleware() mist.HandleFunc {
	// 初始化账户锁定管理器
	config := GetSecurityConfig()
	lockoutConfig := config.Auth.AccountLockout

	if !lockoutConfig.Enabled {
		// 如果未启用账户锁定，返回一个无操作中间件
		return func(ctx *mist.Context) {
			// 无操作
		}
	}

	lockoutPolicy := &auth.LockoutPolicy{
		MaxAttempts:     lockoutConfig.MaxAttempts,
		LockoutDuration: lockoutConfig.LockoutDuration,
		ResetDuration:   lockoutConfig.ResetDuration,
		IncludeIPInKey:  lockoutConfig.IncludeIPInKey,
	}

	accountLockout := auth.NewAccountLockout(lockoutPolicy)

	// 启动自动清理过期锁定记录的后台任务
	cleanupCtx, cancel := context.WithCancel(context.Background())
	go accountLockout.RunLockoutCleanup(cleanupCtx, lockoutConfig.CleanupInterval)

	// 确保在应用关闭时取消清理任务
	securityCleanupFunctions = append(securityCleanupFunctions, cancel)

	return func(ctx *mist.Context) {
		// 从请求中获取用户ID
		// 这个逻辑需要根据应用程序的认证机制进行调整
		userIDValue, ok := ctx.UserValues["user_id"]
		if !ok || userIDValue == nil {
			// 如果没有找到用户ID，跳过锁定检查
			return
		}

		userIDStr, ok := userIDValue.(string)
		if !ok {
			// 如果用户ID不是字符串，跳过锁定检查
			return
		}

		// 获取客户端IP
		clientIP := getClientIP(ctx.Request)

		// 检查是否已被锁定
		isLocked, unlockTime := accountLockout.IsLocked(userIDStr, clientIP)
		if isLocked {
			waitTime := time.Until(unlockTime).Round(time.Minute)
			ctx.RespStatusCode = http.StatusTooManyRequests
			ctx.RespData = []byte("账户已被锁定，请在" + waitTime.String() + "后重试")
			return
		}

		// 记录请求结果
		// 这个钩子将检测身份验证失败并记录失败尝试
		if ctx.UserValues == nil {
			ctx.UserValues = make(map[string]any)
		}
		ctx.UserValues["account_lockout"] = accountLockout
		ctx.UserValues["lockout_user_id"] = userIDStr
		ctx.UserValues["lockout_ip"] = clientIP
	}
}

// 全局清理函数列表，用于在应用关闭时执行
var securityCleanupFunctions []func()

// CleanupSecurity 清理安全相关资源
func CleanupSecurity() {
	for _, cleanup := range securityCleanupFunctions {
		cleanup()
	}
	securityCleanupFunctions = nil
}

// getClientIP 从请求中获取客户端IP
func getClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// 可能有多个IP，取第一个
		return ip
	}

	// 尝试从X-Real-IP获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	return r.RemoteAddr
}

// RecordFailedLoginAttempt 记录失败的登录尝试
func RecordFailedLoginAttempt(ctx *mist.Context) bool {
	if ctx.UserValues == nil {
		return false
	}

	lockoutObj, ok := ctx.UserValues["account_lockout"]
	if !ok || lockoutObj == nil {
		return false
	}

	accountLockout, ok := lockoutObj.(*auth.AccountLockout)
	if !ok {
		return false
	}

	userID, _ := ctx.UserValues["lockout_user_id"].(string)
	clientIP, _ := ctx.UserValues["lockout_ip"].(string)

	if userID == "" {
		return false
	}

	return accountLockout.RecordFailedAttempt(userID, clientIP)
}

// ResetLockout 重置账户锁定
func ResetLockout(ctx *mist.Context) {
	if ctx.UserValues == nil {
		return
	}

	lockoutObj, ok := ctx.UserValues["account_lockout"]
	if !ok || lockoutObj == nil {
		return
	}

	accountLockout, ok := lockoutObj.(*auth.AccountLockout)
	if !ok {
		return
	}

	userID, _ := ctx.UserValues["lockout_user_id"].(string)
	clientIP, _ := ctx.UserValues["lockout_ip"].(string)

	if userID == "" {
		return
	}

	accountLockout.ResetLockout(userID, clientIP)
}
