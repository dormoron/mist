package middleware

import (
	"net/http"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/blocklist"
)

// BlocklistConfig 中间件配置
type BlocklistConfig struct {
	// Manager IP黑名单管理器
	Manager *blocklist.Manager
	// OnBlocked 当IP被封禁时的处理函数
	OnBlocked func(*mist.Context)
}

// DefaultConfig 返回默认配置
func DefaultConfig(manager *blocklist.Manager) BlocklistConfig {
	return BlocklistConfig{
		Manager: manager,
		OnBlocked: func(ctx *mist.Context) {
			ctx.AbortWithStatus(http.StatusForbidden)
		},
	}
}

// Option 配置选项函数
type Option func(*BlocklistConfig)

// WithOnBlocked 设置IP被封禁时的处理函数
func WithOnBlocked(handler func(*mist.Context)) Option {
	return func(c *BlocklistConfig) {
		c.OnBlocked = handler
	}
}

// New 创建IP黑名单中间件
func New(manager *blocklist.Manager, opts ...Option) mist.Middleware {
	// 使用默认配置
	config := DefaultConfig(manager)

	// 应用自定义选项
	for _, opt := range opts {
		opt(&config)
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			ip := ctx.ClientIP()

			// 如果IP被封禁，中断请求
			if config.Manager.IsBlocked(ip) {
				// 调用封禁处理函数
				if config.OnBlocked != nil {
					config.OnBlocked(ctx)
				} else {
					ctx.AbortWithStatus(http.StatusForbidden)
				}
				return
			}

			// 继续处理请求
			next(ctx)
		}
	}
}
