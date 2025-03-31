package mist

import (
	"net/http"
	"testing"
)

func TestRouter_RegisterRoute(t *testing.T) {
	testCases := []struct {
		name      string
		method    string
		path      string
		wantPanic bool
	}{
		{
			name:      "正常路径",
			method:    http.MethodGet,
			path:      "/user",
			wantPanic: false,
		},
		{
			name:      "正常带参数路径",
			method:    http.MethodGet,
			path:      "/user/:id",
			wantPanic: false,
		},
		{
			name:      "正常多级路径",
			method:    http.MethodGet,
			path:      "/user/:id/profile",
			wantPanic: false,
		},
		{
			name:      "空路径-应该触发panic",
			method:    http.MethodGet,
			path:      "",
			wantPanic: true,
		},
		{
			name:      "不以/开头-应该触发panic",
			method:    http.MethodGet,
			path:      "user",
			wantPanic: true,
		},
		{
			name:      "以/结尾-应该触发panic",
			method:    http.MethodGet,
			path:      "/user/",
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tc.wantPanic {
						t.Errorf("注册路由时不应该panic，但是panic了: %v", r)
					}
				} else {
					if tc.wantPanic {
						t.Errorf("注册路由时应该panic，但是没有panic")
					}
				}
			}()

			r := initRouter()
			r.registerRoute(tc.method, tc.path, func(ctx *Context) {})
		})
	}
}

func TestRouter_FindRoute(t *testing.T) {
	r := initRouter()

	// 注册路由
	r.registerRoute(http.MethodGet, "/", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/user", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/user/:id", func(ctx *Context) {})
	r.registerRoute(http.MethodGet, "/user/:id/profile", func(ctx *Context) {})
	r.registerRoute(http.MethodPost, "/order", func(ctx *Context) {})

	testCases := []struct {
		name   string
		method string
		path   string
		found  bool
		params map[string]string
	}{
		{
			name:   "根路径",
			method: http.MethodGet,
			path:   "/",
			found:  true,
			params: map[string]string{},
		},
		{
			name:   "普通路径",
			method: http.MethodGet,
			path:   "/user",
			found:  true,
			params: map[string]string{},
		},
		{
			name:   "带参数路径",
			method: http.MethodGet,
			path:   "/user/123",
			found:  true,
			params: map[string]string{"id": "123"},
		},
		{
			name:   "多段带参数路径",
			method: http.MethodGet,
			path:   "/user/123/profile",
			found:  true,
			params: map[string]string{"id": "123"},
		},
		{
			name:   "不存在的路径",
			method: http.MethodGet,
			path:   "/not-exist",
			found:  false,
		},
		{
			name:   "不支持的HTTP方法",
			method: http.MethodDelete,
			path:   "/user",
			found:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, found := r.findRoute(tc.method, tc.path)
			if found != tc.found {
				t.Errorf("路由查找结果错误, 期望 %v, 实际 %v", tc.found, found)
				return
			}

			if !found {
				return
			}

			// 验证参数
			if len(info.pathParams) != len(tc.params) {
				t.Errorf("路径参数数量不匹配, 期望 %d, 实际 %d", len(tc.params), len(info.pathParams))
				return
			}

			for k, v := range tc.params {
				if info.pathParams[k] != v {
					t.Errorf("路径参数不匹配, 键 %s, 期望 %s, 实际 %s", k, v, info.pathParams[k])
				}
			}
		})
	}
}
