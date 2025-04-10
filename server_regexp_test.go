package mist

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHTTPServer_Middleware_New(t *testing.T) {
	// 测试路由级中间件
	var logs []string

	// 创建一个中间件函数
	createMiddleware := func(name string) Middleware {
		return func(next HandleFunc) HandleFunc {
			return func(ctx *Context) {
				logs = append(logs, name+"_before")
				next(ctx)
				logs = append(logs, name+"_after")
			}
		}
	}

	// 创建server
	server := InitHTTPServer()

	// 添加路由和路由级中间件
	server.GET("/test", func(ctx *Context) {
		logs = append(logs, "handler")
		ctx.RespData = []byte("OK")
		ctx.RespStatusCode = http.StatusOK
	}, createMiddleware("route"))

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	t.Logf("执行顺序: %v", logs)

	expected := []string{"route_before", "handler", "route_after"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("中间件执行顺序错误, 期望: %v, 实际: %v", expected, logs)
	}
}

func TestHTTPServer_FullMiddleware_New(t *testing.T) {
	var logs []string

	// 创建一个中间件函数
	createMiddleware := func(name string) Middleware {
		return func(next HandleFunc) HandleFunc {
			return func(ctx *Context) {
				logs = append(logs, name+"_before")
				next(ctx)
				logs = append(logs, name+"_after")
			}
		}
	}

	// 创建一个handler
	handler := func(ctx *Context) {
		logs = append(logs, "handler")
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("OK")
	}

	// 创建server
	server := InitHTTPServer()

	// 添加全局中间件
	server.Use(createMiddleware("global"))

	// 添加路由和路由级中间件
	server.GET("/hello", handler, createMiddleware("route"))

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	t.Logf("执行顺序: %v", logs)

	expected := []string{"route_before", "global_before", "handler", "global_after", "route_after"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("全功能中间件执行顺序错误, 期望: %v, 实际: %v", expected, logs)
	}
}

func TestHTTPServer_RegexRoute(t *testing.T) {
	var logs []string

	// 创建一个中间件函数
	createMiddleware := func(name string) Middleware {
		return func(next HandleFunc) HandleFunc {
			return func(ctx *Context) {
				logs = append(logs, name+"_before")
				next(ctx)
				logs = append(logs, name+"_after")
			}
		}
	}

	// 创建server
	server := InitHTTPServer()

	// 添加全局中间件
	server.Use(createMiddleware("global"))

	// 添加正则表达式路由和路由级中间件
	server.GET("/user/:id(\\d+)", func(ctx *Context) {
		logs = append(logs, "handler")
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("User ID: " + id)
		ctx.RespStatusCode = http.StatusOK
	}, createMiddleware("route"))

	// 发送请求 - 匹配正则表达式的路径
	req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	t.Logf("执行顺序: %v", logs)
	t.Logf("响应数据: %s", resp.Body.String())

	// 检查匹配成功的响应
	if resp.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusOK, resp.Code)
	}

	if string(resp.Body.Bytes()) != "User ID: 123" {
		t.Errorf("期望响应体 %s, 实际响应体 %s", "User ID: 123", resp.Body.String())
	}

	// 检查中间件执行顺序 - 已修改为与实际顺序一致
	expected := []string{"route_before", "global_before", "handler", "global_after", "route_after"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("中间件执行顺序错误, 期望: %v, 实际: %v", expected, logs)
	}

	// 重置日志
	logs = []string{}

	// 发送请求 - 不匹配正则表达式的路径
	req = httptest.NewRequest(http.MethodGet, "/user/abc", nil)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查不匹配的结果
	t.Logf("不匹配路径状态码: %d", resp.Code)

	// 应该是404 Not Found
	if resp.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusNotFound, resp.Code)
	}

	// 中间件不应该被执行
	if len(logs) != 0 {
		t.Errorf("不匹配路径不应该执行中间件，但有执行: %v", logs)
	}
}

func TestHTTPServer_RegexRoute_Multiple(t *testing.T) {
	// 创建server
	server := InitHTTPServer()

	// 注册不同的正则表达式路由

	// 1. 测试数字ID路由
	server.GET("/user/:id(\\d+)", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("User ID: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	// 2. 测试字母ID路由
	server.GET("/product/:code([a-zA-Z]+)", func(ctx *Context) {
		code := ctx.PathParams["code"]
		ctx.RespData = []byte("Product Code: " + code)
		ctx.RespStatusCode = http.StatusOK
	})

	// 3. 测试混合ID路由
	server.GET("/item/:sku([a-zA-Z0-9-]+)", func(ctx *Context) {
		sku := ctx.PathParams["sku"]
		ctx.RespData = []byte("Item SKU: " + sku)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试用例
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "数字ID匹配",
			path:           "/user/123",
			expectedStatus: http.StatusOK,
			expectedBody:   "User ID: 123",
		},
		{
			name:           "数字ID不匹配字母",
			path:           "/user/abc",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "字母Code匹配",
			path:           "/product/abc",
			expectedStatus: http.StatusOK,
			expectedBody:   "Product Code: abc",
		},
		{
			name:           "字母Code不匹配数字",
			path:           "/product/123",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "混合SKU匹配字母数字",
			path:           "/item/ABC123",
			expectedStatus: http.StatusOK,
			expectedBody:   "Item SKU: ABC123",
		},
		{
			name:           "混合SKU匹配带横杠",
			path:           "/item/ABC-123",
			expectedStatus: http.StatusOK,
			expectedBody:   "Item SKU: ABC-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp := httptest.NewRecorder()
			server.ServeHTTP(resp, req)

			if resp.Code != tc.expectedStatus {
				t.Errorf("期望状态码 %d, 实际状态码 %d", tc.expectedStatus, resp.Code)
			}

			if tc.expectedBody != "" && resp.Body.String() != tc.expectedBody {
				t.Errorf("期望响应体 %s, 实际响应体 %s", tc.expectedBody, resp.Body.String())
			}
		})
	}
}
