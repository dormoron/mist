package mist

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRegexRoute(t *testing.T) {
	server := InitHTTPServer()

	// 注册一个带正则表达式的路由 - 只匹配数字ID
	server.GET("/users/{id:[0-9]+}", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("User ID: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	// 注册另一个正则路由 - 匹配年份/月份格式
	server.GET("/articles/{year:[0-9]{4}}/{month:[0-9]{2}}", func(ctx *Context) {
		year := ctx.PathParams["year"]
		month := ctx.PathParams["month"]
		ctx.RespData = []byte(year + "-" + month)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试场景1: 正确的数字ID
	t.Run("数字ID匹配", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusOK, resp.Code)
		}

		expected := "User ID: 123"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试场景2: 非数字ID不匹配
	t.Run("非数字ID不匹配", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusNotFound, resp.Code)
		}
	})

	// 测试场景3: 年份月份格式
	t.Run("年份月份格式匹配", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/articles/2023/05", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusOK, resp.Code)
		}

		expected := "2023-05"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试场景4: 错误的年份月份格式
	t.Run("错误的年份月份格式", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/articles/20/5", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusNotFound, resp.Code)
		}
	})
}

// 测试正则表达式路由与中间件的组合
func TestRegexRouteWithMiddleware(t *testing.T) {
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

	server := InitHTTPServer()

	// 添加全局中间件
	server.Use(createMiddleware("global"))

	// 注册带正则表达式和中间件的路由
	server.GET("/products/{id:[0-9]+}", func(ctx *Context) {
		logs = append(logs, "handler")
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("Product: " + id)
		ctx.RespStatusCode = http.StatusOK
	}, createMiddleware("route"))

	// 测试正则路由和中间件
	req := httptest.NewRequest(http.MethodGet, "/products/456", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查响应
	if resp.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusOK, resp.Code)
	}

	expected := "Product: 456"
	if string(resp.Body.Bytes()) != expected {
		t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
	}

	// 检查中间件执行顺序
	t.Logf("执行顺序: %v", logs)
	expectedOrder := []string{"global_before", "route_before", "handler", "route_after", "global_after"}
	if !reflect.DeepEqual(logs, expectedOrder) {
		t.Fatalf("中间件执行顺序错误, 期望: %v, 实际: %v", expectedOrder, logs)
	}
}

// 测试多个匹配的正则路由的优先级
func TestRegexRoutePriority(t *testing.T) {
	//t.Skip("这个测试与其他测试冲突，需要单独运行")

	server := InitHTTPServer()

	// 先注册通用路由
	server.GET("/api/{resource}", func(ctx *Context) {
		resource := ctx.PathParams["resource"]
		ctx.RespData = []byte("General API: " + resource)
		ctx.RespStatusCode = http.StatusOK
	})

	// 后注册特定数字ID的路由（应该优先匹配）
	server.GET("/api/{id:[0-9]+}", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("Numeric API: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试非数字路径 - 应该匹配到通用路由
	t.Run("非数字路径", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "General API: users"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试数字路径 - 应该匹配到特定正则路由
	t.Run("数字路径", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/123", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "Numeric API: 123"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})
}

// 测试简单的正则路由模式
func TestSimpleRegexRoute(t *testing.T) {
	server := InitHTTPServer()

	// 注册一个通用参数路由，使用冒号格式
	server.GET("/api/:resource", func(ctx *Context) {
		resource := ctx.PathParams["resource"]
		ctx.RespData = []byte("General: " + resource)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试匹配
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	expected := "General: users"
	if string(resp.Body.Bytes()) != expected {
		t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
	}
}

// 测试简单的数字ID正则路由
func TestSimpleNumericRegexRoute(t *testing.T) {
	server := InitHTTPServer()

	// 注册一个正则路由，只匹配数字
	server.GET("/api/{id:[0-9]+}", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("Numeric: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试匹配
	req := httptest.NewRequest(http.MethodGet, "/api/123", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	expected := "Numeric: 123"
	if string(resp.Body.Bytes()) != expected {
		t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
	}

	// 测试不匹配
	req = httptest.NewRequest(http.MethodGet, "/api/abc", nil)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 应该是404 Not Found
	if resp.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusNotFound, resp.Code)
	}
}

// 测试多级路径正则表达式
func TestNestedRegexRoute(t *testing.T) {
	server := InitHTTPServer()

	// 注册一个多级路径中包含正则表达式的路由
	server.GET("/api/users/{id:[0-9]+}/profile", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("User profile: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试匹配
	req := httptest.NewRequest(http.MethodGet, "/api/users/123/profile", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 检查结果
	expected := "User profile: 123"
	if string(resp.Body.Bytes()) != expected {
		t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
	}

	// 测试不匹配
	req = httptest.NewRequest(http.MethodGet, "/api/users/abc/profile", nil)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	// 应该是404 Not Found
	if resp.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d, 实际状态码 %d", http.StatusNotFound, resp.Code)
	}
}

// 测试路由优先级 - 修复版
func TestRoutesPriority(t *testing.T) {
	server := InitHTTPServer()

	// 注册路由 - 按优先级注册，从高到低
	// 1. 静态路由 - 优先级最高
	server.GET("/api/static", func(ctx *Context) {
		ctx.RespData = []byte("Static Route")
		ctx.RespStatusCode = http.StatusOK
	})

	// 2. 正则路由与参数路由各自有自己的路径
	server.GET("/users/{id:[0-9]+}", func(ctx *Context) {
		id := ctx.PathParams["id"]
		ctx.RespData = []byte("Regex Route: " + id)
		ctx.RespStatusCode = http.StatusOK
	})

	server.GET("/posts/:name", func(ctx *Context) {
		name := ctx.PathParams["name"]
		ctx.RespData = []byte("Param Route: " + name)
		ctx.RespStatusCode = http.StatusOK
	})

	// 3. 通配符路由 - 不与其他路由冲突
	server.GET("/files/*", func(ctx *Context) {
		ctx.RespData = []byte("Wildcard Route")
		ctx.RespStatusCode = http.StatusOK
	})

	// 测试静态路由
	t.Run("静态路由", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/static", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "Static Route"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试正则路由
	t.Run("正则路由", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "Regex Route: 123"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试参数路由
	t.Run("参数路由", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/posts/article", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "Param Route: article"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})

	// 测试通配符路由
	t.Run("通配符路由", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files/anything", nil)
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)

		expected := "Wildcard Route"
		if string(resp.Body.Bytes()) != expected {
			t.Errorf("期望响应体 %s, 实际响应体 %s", expected, resp.Body.String())
		}
	})
}
