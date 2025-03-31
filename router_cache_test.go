package mist

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

// 测试路由缓存功能
func TestRouter_Cache(t *testing.T) {
	r := initRouter()
	r.EnableCache(100)

	// 注册一些路由
	r.registerRoute(http.MethodGet, "/", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/users", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/users/:id", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/products", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/products/:id", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/products/:id/details", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/products/:id/reviews", func(ctx *Context) {})

	paths := []string{
		"/",
		"/users",
		"/users/123",
		"/products",
		"/products/456",
		"/products/456/details",
		"/products/456/reviews",
	}

	// 首次访问，缓存未命中
	for _, path := range paths {
		_, found := r.findRoute(http.MethodGet, path)
		if !found {
			t.Errorf("路由未找到: %s", path)
		}
	}

	// 检查缓存统计
	hits, misses, size := r.CacheStats()
	if hits != 0 {
		t.Errorf("首次访问后，缓存命中数应为0，实际为 %d", hits)
	}
	if misses != 7 {
		t.Errorf("首次访问后，缓存未命中数应为7，实际为 %d", misses)
	}
	if size != 7 {
		t.Errorf("首次访问后，缓存大小应为7，实际为 %d", size)
	}

	// 二次访问，应该命中缓存
	for _, path := range paths {
		_, found := r.findRoute(http.MethodGet, path)
		if !found {
			t.Errorf("路由未找到: %s", path)
		}
	}

	// 再次检查缓存统计
	hits, misses, size = r.CacheStats()
	if hits != 7 {
		t.Errorf("二次访问后，缓存命中数应为7，实际为 %d", hits)
	}
	if misses != 7 {
		t.Errorf("二次访问后，缓存未命中数应为7，实际为 %d", misses)
	}
	if size != 7 {
		t.Errorf("二次访问后，缓存大小应为7，实际为 %d", size)
	}

	// 禁用缓存
	r.DisableCache()
	hits, misses, size = r.CacheStats()
	if size != 0 {
		t.Errorf("禁用缓存后，缓存大小应为0，实际为 %d", size)
	}

	// 禁用缓存后，缓存不应该被使用
	for _, path := range paths {
		_, found := r.findRoute(http.MethodGet, path)
		if !found {
			t.Errorf("路由未找到: %s", path)
		}
	}

	// 验证缓存没有增加
	hits, misses, size = r.CacheStats()
	if size != 0 {
		t.Errorf("禁用缓存后访问，缓存大小应为0，实际为 %d", size)
	}
}

// 测试静态路由快速匹配
func TestRouter_FastMatch(t *testing.T) {
	r := initRouter()

	// 注册静态路由
	r.registerRoute(http.MethodGet, "/static/path", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/static/another/path", func(ctx *Context) {})

	// 测试快速匹配
	_, found := r.findRoute(http.MethodGet, "/static/path")
	if !found {
		t.Error("静态路由未找到: /static/path")
	}

	_, found = r.findRoute(http.MethodGet, "/static/another/path")
	if !found {
		t.Error("静态路由未找到: /static/another/path")
	}

	// 测试不存在的路由
	_, found = r.findRoute(http.MethodGet, "/static/not/exist")
	if found {
		t.Error("不应该找到不存在的路由: /static/not/exist")
	}
}

// 测试路由缓存超过最大大小的情况
func TestRouter_CacheMaxSize(t *testing.T) {
	r := initRouter()
	r.EnableCache(5) // 设置较小的缓存大小，便于测试

	// 注册一些路由
	r.registerRoute(http.MethodGet, "/", func(ctx *Context) {})

	// 第一次访问一些路径，超过缓存大小
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("/path/%d", i)
		_, found := r.findRoute(http.MethodGet, path)
		if found {
			t.Errorf("不应该找到不存在的路由: %s", path)
		}
	}

	// 检查缓存大小不超过最大限制
	_, _, size := r.CacheStats()
	if size > 5 {
		t.Errorf("缓存大小超过最大值，应为5，实际为 %d", size)
	}
}

// 缓存性能基准测试
func BenchmarkRouter_WithoutCache(b *testing.B) {
	r := initRouter()
	r.DisableCache() // 确保缓存被禁用

	// 注册一些路由
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/path/%d", i)
		r.registerRoute(http.MethodGet, path, func(ctx *Context) {})
	}

	// 参数路由
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("/users/%d/:id", i)
		r.registerRoute(http.MethodGet, path, func(ctx *Context) {})
	}

	b.ResetTimer()

	// 随机访问路由，模拟真实请求
	for i := 0; i < b.N; i++ {
		idx := i % 100
		path := fmt.Sprintf("/path/%d", idx)
		r.findRoute(http.MethodGet, path)
	}
}

func BenchmarkRouter_WithCache(b *testing.B) {
	r := initRouter()
	r.EnableCache(1000) // 确保缓存被启用

	// 注册一些路由
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/path/%d", i)
		r.registerRoute(http.MethodGet, path, func(ctx *Context) {})
	}

	// 参数路由
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("/users/%d/:id", i)
		r.registerRoute(http.MethodGet, path, func(ctx *Context) {})
	}

	b.ResetTimer()

	// 随机访问路由，模拟真实请求
	for i := 0; i < b.N; i++ {
		idx := i % 100
		path := fmt.Sprintf("/path/%d", idx)
		r.findRoute(http.MethodGet, path)
	}
}

// 测试路由缓存与参数路由
func BenchmarkRouter_WithParams(b *testing.B) {
	testCases := []struct {
		name         string
		cacheEnabled bool
	}{
		{"WithCache", true},
		{"WithoutCache", false},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			r := initRouter()
			if tc.cacheEnabled {
				r.EnableCache(1000)
			} else {
				r.DisableCache()
			}

			// 注册带参数的路由
			r.registerRoute(http.MethodGet, "/users/:id", func(ctx *Context) {})
			r.registerRoute(http.MethodGet, "/products/:category/:id", func(ctx *Context) {})
			r.registerRoute(http.MethodGet, "/articles/:year/:month/:slug", func(ctx *Context) {})

			b.ResetTimer()

			// 测试参数路由匹配
			for i := 0; i < b.N; i++ {
				userID := strconv.Itoa(i % 100)
				r.findRoute(http.MethodGet, "/users/"+userID)

				category := "cat" + strconv.Itoa(i%5)
				productID := strconv.Itoa(i % 20)
				r.findRoute(http.MethodGet, "/products/"+category+"/"+productID)

				year := strconv.Itoa(2020 + (i % 5))
				month := strconv.Itoa(1 + (i % 12))
				slug := "article-" + strconv.Itoa(i%10)
				r.findRoute(http.MethodGet, "/articles/"+year+"/"+month+"/"+slug)
			}
		})
	}
}
