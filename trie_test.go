package mist

import (
	"fmt"
	"net/http"
	"testing"
)

func TestPathTrie_Basic(t *testing.T) {
	trie := NewPathTrie()

	// 注册路由
	trie.Add(http.MethodGet, "/", func(ctx *Context) {
		ctx.RespData = []byte("Root")
	})

	trie.Add(http.MethodGet, "/users", func(ctx *Context) {
		ctx.RespData = []byte("Users")
	})

	trie.Add(http.MethodGet, "/users/:id", func(ctx *Context) {
		ctx.RespData = []byte("User Detail")
	})

	trie.Add(http.MethodGet, "/products/:category/:id", func(ctx *Context) {
		ctx.RespData = []byte("Product Detail")
	})

	// 测试匹配
	testCases := []struct {
		method      string
		path        string
		shouldMatch bool
		paramCount  int
		params      map[string]string
	}{
		{http.MethodGet, "/", true, 0, nil},
		{http.MethodGet, "/users", true, 0, nil},
		{http.MethodGet, "/users/123", true, 1, map[string]string{"id": "123"}},
		{http.MethodGet, "/products/electronics/456", true, 2, map[string]string{"category": "electronics", "id": "456"}},
		{http.MethodGet, "/not-exist", false, 0, nil},
		{http.MethodPost, "/users", false, 0, nil},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			mi, matched := trie.Match(tc.method, tc.path)

			if matched != tc.shouldMatch {
				t.Errorf("路径 %s 匹配结果错误, 期望 %v, 实际 %v", tc.path, tc.shouldMatch, matched)
				return
			}

			if !matched {
				return
			}

			if len(mi.pathParams) != tc.paramCount {
				t.Errorf("路径 %s 参数数量不匹配, 期望 %d, 实际 %d", tc.path, tc.paramCount, len(mi.pathParams))
				return
			}

			for k, expectedVal := range tc.params {
				if actualVal, ok := mi.pathParams[k]; !ok || actualVal != expectedVal {
					t.Errorf("路径 %s 参数不匹配, 参数 %s, 期望 %s, 实际 %s",
						tc.path, k, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestPathTrie_Regex(t *testing.T) {
	trie := NewPathTrie()

	// 注册带正则表达式的路由
	trie.Add(http.MethodGet, "/{id:[0-9]+}", func(ctx *Context) {
		ctx.RespData = []byte("ID")
	})

	trie.Add(http.MethodGet, "/users/{id:[0-9]+}/profile", func(ctx *Context) {
		ctx.RespData = []byte("User Profile")
	})

	trie.Add(http.MethodGet, "/articles/{year:[0-9]{4}}/{month:[0-9]{2}}/{slug:[a-z0-9-]+}", func(ctx *Context) {
		ctx.RespData = []byte("Article")
	})

	// 测试匹配
	testCases := []struct {
		path        string
		shouldMatch bool
		params      map[string]string
	}{
		{"/123", true, map[string]string{"id": "123"}},
		{"/abc", false, nil}, // 不匹配正则表达式
		{"/users/456/profile", true, map[string]string{"id": "456"}},
		{"/users/abc/profile", false, nil}, // 不匹配正则表达式
		{"/articles/2023/05/my-article-123", true, map[string]string{
			"year":  "2023",
			"month": "05",
			"slug":  "my-article-123",
		}},
		{"/articles/20/05/my-article", false, nil}, // 年份不匹配正则表达式
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			mi, matched := trie.Match(http.MethodGet, tc.path)

			if matched != tc.shouldMatch {
				t.Errorf("路径 %s 匹配结果错误, 期望 %v, 实际 %v", tc.path, tc.shouldMatch, matched)
				return
			}

			if !matched {
				return
			}

			if len(mi.pathParams) != len(tc.params) {
				t.Errorf("路径 %s 参数数量不匹配, 期望 %d, 实际 %d",
					tc.path, len(tc.params), len(mi.pathParams))
				return
			}

			for k, expectedVal := range tc.params {
				if actualVal, ok := mi.pathParams[k]; !ok || actualVal != expectedVal {
					t.Errorf("路径 %s 参数不匹配, 参数 %s, 期望 %s, 实际 %s",
						tc.path, k, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestPathTrie_Wildcard(t *testing.T) {
	trie := NewPathTrie()

	// 注册带通配符的路由
	trie.Add(http.MethodGet, "/files/*filepath", func(ctx *Context) {
		ctx.RespData = []byte("File")
	})

	trie.Add(http.MethodGet, "/static/*filename", func(ctx *Context) {
		ctx.RespData = []byte("Static File")
	})

	// 测试匹配
	testCases := []struct {
		path        string
		shouldMatch bool
		param       string
		paramValue  string
	}{
		{"/files/document.pdf", true, "filepath", "document.pdf"},
		{"/files/images/logo.png", true, "filepath", "images/logo.png"},
		{"/static/css/style.css", true, "filename", "css/style.css"},
		{"/static/js/app.js", true, "filename", "js/app.js"},
		{"/other/path", false, "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			mi, matched := trie.Match(http.MethodGet, tc.path)

			if matched != tc.shouldMatch {
				t.Errorf("路径 %s 匹配结果错误, 期望 %v, 实际 %v", tc.path, tc.shouldMatch, matched)
				return
			}

			if !matched {
				return
			}

			if val, ok := mi.pathParams[tc.param]; !ok || val != tc.paramValue {
				t.Errorf("路径 %s 通配符参数不匹配, 期望 %s=%s, 实际 %s",
					tc.path, tc.param, tc.paramValue, val)
			}
		})
	}
}

func TestPathTrie_Cache(t *testing.T) {
	trie := NewPathTrie()
	trie.EnableCache(100)

	// 注册一些路由
	trie.Add(http.MethodGet, "/users/:id", func(ctx *Context) {})
	trie.Add(http.MethodGet, "/products/:category/:id", func(ctx *Context) {})

	// 首次访问，缓存未命中
	paths := []string{
		"/users/123",
		"/users/456",
		"/products/electronics/789",
		"/products/books/101",
	}

	for _, path := range paths {
		_, _ = trie.Match(http.MethodGet, path)
	}

	// 检查缓存统计
	hits, misses, size := trie.CacheStats()
	if hits != 0 {
		t.Errorf("首次访问后，缓存命中数应为0，实际为 %d", hits)
	}
	if misses != 4 {
		t.Errorf("首次访问后，缓存未命中数应为4，实际为 %d", misses)
	}
	if size != 4 {
		t.Errorf("首次访问后，缓存大小应为4，实际为 %d", size)
	}

	// 再次访问相同路径，应该命中缓存
	for _, path := range paths {
		_, _ = trie.Match(http.MethodGet, path)
	}

	// 再次检查缓存统计
	hits, misses, size = trie.CacheStats()
	if hits != 4 {
		t.Errorf("二次访问后，缓存命中数应为4，实际为 %d", hits)
	}
	if misses != 4 {
		t.Errorf("二次访问后，缓存未命中数应为4，实际为 %d", misses)
	}

	// 禁用缓存
	trie.DisableCache()
	_, _, size = trie.CacheStats()
	if size != 0 {
		t.Errorf("禁用缓存后，缓存大小应为0，实际为 %d", size)
	}
}

func BenchmarkPathTrie_StaticRoutes(b *testing.B) {
	trie := NewPathTrie()

	// 注册100个静态路由
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/path/%d", i)
		trie.Add(http.MethodGet, path, func(ctx *Context) {})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx := i % 100
		path := fmt.Sprintf("/path/%d", idx)
		trie.Match(http.MethodGet, path)
	}
}

func BenchmarkPathTrie_ParamRoutes(b *testing.B) {
	trie := NewPathTrie()

	// 注册带参数的路由
	trie.Add(http.MethodGet, "/users/:id", func(ctx *Context) {})
	trie.Add(http.MethodGet, "/users/:id/profile", func(ctx *Context) {})
	trie.Add(http.MethodGet, "/products/:category/:id", func(ctx *Context) {})

	paths := []string{
		"/users/123",
		"/users/123/profile",
		"/products/electronics/456",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := paths[i%3]
		trie.Match(http.MethodGet, path)
	}
}

func BenchmarkPathTrie_RegexRoutes(b *testing.B) {
	trie := NewPathTrie()

	// 注册带正则表达式的路由
	trie.Add(http.MethodGet, "/{id:[0-9]+}", func(ctx *Context) {})
	trie.Add(http.MethodGet, "/articles/{year:[0-9]{4}}/{month:[0-9]{2}}/{slug:[a-z0-9-]+}", func(ctx *Context) {})

	paths := []string{
		"/123",
		"/articles/2023/05/my-article-123",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := paths[i%2]
		trie.Match(http.MethodGet, path)
	}
}

func BenchmarkPathTrie_Cache(b *testing.B) {
	testCases := []struct {
		name         string
		cacheEnabled bool
	}{
		{"WithCache", true},
		{"WithoutCache", false},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			trie := NewPathTrie()
			if tc.cacheEnabled {
				trie.EnableCache(1000)
			} else {
				trie.DisableCache()
			}

			// 注册一些路由
			trie.Add(http.MethodGet, "/users/:id", func(ctx *Context) {})
			trie.Add(http.MethodGet, "/products/:category/:id", func(ctx *Context) {})
			trie.Add(http.MethodGet, "/articles/:year/:month/:slug", func(ctx *Context) {})

			// 使用少量路径，以便在启用缓存时能充分利用缓存
			paths := []string{
				"/users/123",
				"/users/456",
				"/products/electronics/789",
				"/products/books/101",
				"/articles/2023/05/my-article",
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				path := paths[i%5]
				trie.Match(http.MethodGet, path)
			}
		})
	}
}
