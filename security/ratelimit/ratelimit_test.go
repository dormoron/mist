package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dormoron/mist"
	"github.com/stretchr/testify/assert"
)

// 测试令牌桶实现
func TestTokenBucket(t *testing.T) {
	// 创建令牌桶，每秒5个令牌，容量为10
	bucket := newTokenBucket(5, 10)

	// 首次创建应该有满额令牌
	assert.True(t, bucket.getToken(), "初始应该能获取令牌")

	// 连续获取9个令牌（已经消耗了1个）
	for i := 0; i < 9; i++ {
		assert.True(t, bucket.getToken(), "应该能获取令牌 #%d", i+1)
	}

	// 现在应该没有令牌了
	assert.False(t, bucket.getToken(), "应该无法获取更多令牌")

	// 等待200毫秒，应该生成1个新令牌
	// 200毫秒 * 5令牌/秒 = 1个令牌
	time.Sleep(200 * time.Millisecond)
	assert.True(t, bucket.getToken(), "等待后应该能获取新令牌")

	// 再次尝试应该失败
	assert.False(t, bucket.getToken(), "应该无法获取更多令牌")
}

// 测试内存限流器
func TestMemoryLimiter(t *testing.T) {
	limiter := NewMemoryLimiter(5, 10)

	// 测试单个键
	key := "test-key"

	// 前10个请求应该成功
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(key)
		assert.NoError(t, err, "第%d次请求不应有错误", i+1)
		assert.True(t, allowed, "第%d次请求应该被允许", i+1)
	}

	// 第11个请求应该被限流
	allowed, err := limiter.Allow(key)
	assert.NoError(t, err)
	assert.False(t, allowed, "超出限制的请求应该被拒绝")

	// 测试不同键独立限流
	otherKey := "other-key"
	allowed, err = limiter.Allow(otherKey)
	assert.NoError(t, err)
	assert.True(t, allowed, "不同键的请求应该被独立限流")

	// 测试重置功能
	err = limiter.Reset(key)
	assert.NoError(t, err, "重置不应有错误")

	// 重置后应该能再次获取令牌
	allowed, err = limiter.Allow(key)
	assert.NoError(t, err)
	assert.True(t, allowed, "重置后应该能获取令牌")
}

// 测试限流中间件
func TestRateLimitMiddleware(t *testing.T) {
	// 创建限流器和中间件
	limiter := NewMemoryLimiter(5, 2) // 每秒5个请求，最多突发2个
	middleware := New(limiter)

	// 创建一个测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := middleware(testHandler)

	// 测试前2个请求（应该通过）
	for i := 0; i < 2; i++ {
		// 重置标志
		handlerCalled = false

		// 创建请求
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345" // 设置远程IP
		w := httptest.NewRecorder()

		ctx := &mist.Context{
			Request:        req,
			ResponseWriter: w,
		}

		// 执行处理函数
		handler(ctx)

		// 验证结果
		assert.True(t, handlerCalled, "第%d个请求应该被处理", i+1)
		assert.Equal(t, http.StatusOK, w.Code, "应该返回200状态码")
	}

	// 第3个请求应该被限流
	handlerCalled = false
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
	}

	// 执行处理函数
	handler(ctx)

	// 验证结果
	assert.False(t, handlerCalled, "超出限制的请求不应该被处理")
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "应该返回429状态码")
}

// 测试白名单功能
func TestRateLimitWhitelist(t *testing.T) {
	// 创建限流器和带白名单的中间件
	limiter := NewMemoryLimiter(5, 1) // 每秒5个请求，最多突发1个
	middleware := New(limiter, WithWhitelist([]string{"192.168.1.100"}))

	// 创建一个测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := middleware(testHandler)

	// 测试在白名单中的IP - 应该不受限流影响
	for i := 0; i < 5; i++ { // 尝试5次，应该都通过
		handlerCalled = false

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		ctx := &mist.Context{
			Request:        req,
			ResponseWriter: w,
		}

		// 执行处理函数
		handler(ctx)

		// 验证结果
		assert.True(t, handlerCalled, "白名单IP的第%d个请求应该被处理", i+1)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// 测试自定义键提取器
func TestCustomKeyExtractor(t *testing.T) {
	// 创建限流器和带自定义键提取器的中间件
	limiter := NewMemoryLimiter(5, 2)
	middleware := New(limiter, WithKeyExtractor(PathKeyExtractor))

	// 创建一个测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := middleware(testHandler)

	// 测试不同路径
	paths := []string{"/api/users", "/api/products"}

	// 对每个路径都测试限流
	for _, path := range paths {
		// 每个路径可以发送2个请求
		for i := 0; i < 2; i++ {
			handlerCalled = false

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			ctx := &mist.Context{
				Request:        req,
				ResponseWriter: w,
			}

			// 执行处理函数
			handler(ctx)

			// 验证结果
			assert.True(t, handlerCalled, "路径 %s 的第 %d 个请求应该被处理", path, i+1)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 第3个请求应该被限流
		handlerCalled = false
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		ctx := &mist.Context{
			Request:        req,
			ResponseWriter: w,
		}

		// 执行处理函数
		handler(ctx)

		// 验证结果
		assert.False(t, handlerCalled, "路径 %s 超出限制的请求不应该被处理", path)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	}
}

// 测试自定义错误处理
func TestCustomErrorHandler(t *testing.T) {
	// 创建自定义错误处理函数
	customErrorHandled := false
	customErrorHandler := func(ctx *mist.Context, err error) {
		customErrorHandled = true
		ctx.ResponseWriter.WriteHeader(http.StatusServiceUnavailable) // 返回503而不是429
	}

	// 创建限流器和带自定义错误处理的中间件
	limiter := NewMemoryLimiter(5, 1)
	middleware := New(limiter, WithErrorHandler(customErrorHandler))

	// 创建一个测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := middleware(testHandler)

	// 发送一个请求消耗令牌
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	ctx1 := &mist.Context{
		Request:        req1,
		ResponseWriter: w1,
	}
	handler(ctx1)

	// 第二个请求应该触发自定义错误处理
	handlerCalled = false
	customErrorHandled = false

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	ctx2 := &mist.Context{
		Request:        req2,
		ResponseWriter: w2,
	}

	// 执行处理函数
	handler(ctx2)

	// 验证结果
	assert.False(t, handlerCalled, "超出限制的请求不应该被处理")
	assert.True(t, customErrorHandled, "应该调用自定义错误处理函数")
	assert.Equal(t, http.StatusServiceUnavailable, w2.Code, "应该返回503状态码")
}
