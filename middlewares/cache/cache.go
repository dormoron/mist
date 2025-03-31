// Package cache 提供了基于内存的HTTP响应缓存中间件
package cache

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
	lru "github.com/hashicorp/golang-lru"
)

// ResponseCache 用于缓存HTTP响应数据
type ResponseCache struct {
	// 缓存
	cache *lru.Cache
	// 互斥锁，保护缓存访问
	mu sync.RWMutex
	// 缓存项TTL
	ttl time.Duration
	// 缓存大小
	size int
}

// cachedResponse 表示缓存的响应数据
type cachedResponse struct {
	// 响应数据
	data []byte
	// 状态码
	statusCode int
	// 创建时间
	createdAt time.Time
	// HTTP头
	headers http.Header
}

// New 创建一个新的ResponseCache实例
// size: 缓存项数量
// ttl: 缓存项过期时间
func New(size int, ttl time.Duration) (*ResponseCache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}

	return &ResponseCache{
		cache: cache,
		ttl:   ttl,
		size:  size,
	}, nil
}

// Middleware 创建一个缓存中间件，使用给定的键生成器函数
// keyFunc 函数用于从请求上下文生成缓存键
func (rc *ResponseCache) Middleware(keyFunc func(*mist.Context) string) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 仅缓存GET请求
			if ctx.Request.Method != http.MethodGet {
				next(ctx)
				return
			}

			// 生成缓存键
			key := keyFunc(ctx)
			if key == "" {
				next(ctx)
				return
			}

			// 尝试从缓存获取
			rc.mu.RLock()
			value, ok := rc.cache.Get(key)
			rc.mu.RUnlock()

			if ok {
				cachedResp, ok := value.(*cachedResponse)
				if ok && !isExpired(cachedResp, rc.ttl) {
					// 使用缓存的响应
					for k, values := range cachedResp.headers {
						for _, v := range values {
							ctx.Header(k, v)
						}
					}
					ctx.RespData = cachedResp.data
					ctx.RespStatusCode = cachedResp.statusCode
					return
				}
			}

			// 缓存未命中，执行请求处理
			next(ctx)

			// 请求处理完毕，将响应添加到缓存
			if ctx.RespStatusCode >= 200 && ctx.RespStatusCode < 300 {
				rc.mu.Lock()
				headers := make(http.Header)
				// 获取原始响应头
				for k, v := range ctx.ResponseWriter.Header() {
					headers[k] = v
				}

				rc.cache.Add(key, &cachedResponse{
					data:       bytes.Clone(ctx.RespData),
					statusCode: ctx.RespStatusCode,
					createdAt:  time.Now(),
					headers:    headers,
				})
				rc.mu.Unlock()
			}
		}
	}
}

// 检查缓存项是否过期
func isExpired(resp *cachedResponse, ttl time.Duration) bool {
	return time.Since(resp.createdAt) > ttl
}

// URLKeyGenerator 返回一个基于请求URL的键生成器函数
func URLKeyGenerator() func(*mist.Context) string {
	return func(ctx *mist.Context) string {
		return ctx.Request.URL.String()
	}
}

// URLAndHeaderKeyGenerator 返回一个基于URL和指定头的键生成器函数
func URLAndHeaderKeyGenerator(headers ...string) func(*mist.Context) string {
	return func(ctx *mist.Context) string {
		key := ctx.Request.URL.String()
		for _, h := range headers {
			key += ":" + ctx.Request.Header.Get(h)
		}
		return key
	}
}

// Clear 清空缓存
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache.Purge()
}
