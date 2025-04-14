package security

import (
	"github.com/dormoron/mist"
)

// MiddlewareBuilder 是用于构建登录检查中间件的构建器
type MiddlewareBuilder struct {
	provider Provider
	paths    []string
}

// InitMiddlewareBuilder 初始化一个新的中间件构建器
// Parameters:
// - provider: 会话提供者接口
// - paths: 需要检查登录状态的路径
// Returns:
// - *MiddlewareBuilder: 初始化后的中间件构建器
func InitMiddlewareBuilder(provider Provider, paths ...string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		provider: provider,
		paths:    paths,
	}
}

// Build 构建中间件
// Returns:
// - mist.Middleware: 构建的中间件函数
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// 创建路径映射集合，用于快速检查
	pathMap := make(map[string]bool)
	for _, path := range m.paths {
		pathMap[path] = true
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 检查当前路径是否需要验证登录
			if _, exists := pathMap[ctx.Request.URL.Path]; !exists {
				// 不需要验证登录，直接执行下一个处理函数
				next(ctx)
				return
			}

			// 尝试获取会话
			session, err := m.provider.Get(ctx)
			if err != nil || session == nil {
				// 未登录，返回未授权状态
				ctx.AbortWithStatus(401) // 未授权
				return
			}

			// 已登录，执行下一个处理函数
			next(ctx)
		}
	}
}
