package mist

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMiddlewareOrder(t *testing.T) {
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
		t.Fatalf("中间件执行顺序错误, 期望: %v, 实际: %v", expected, logs)
	}
}
