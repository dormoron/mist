package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dormoron/mist"
	"github.com/stretchr/testify/assert"
)

// 测试默认配置的安全头部
func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	// 创建中间件
	headersMiddleware := New()

	// 创建测试处理函数
	testHandler := func(ctx *mist.Context) {
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := headersMiddleware(testHandler)

	// 创建请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
	}

	// 执行处理函数
	handler(ctx)

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, w.Code, "应该返回200状态码")

	// 验证头部是否已设置，具体值依赖于DefaultConfig
	// 注意：根据DefaultConfig的实现，可能需要修改这些断言
	resp := w.Result()
	headers := resp.Header

	// 检查一些关键的安全头部
	assert.NotEmpty(t, headers.Get("X-Content-Type-Options"), "X-Content-Type-Options 头部应该已设置")
	assert.NotEmpty(t, headers.Get("X-XSS-Protection"), "X-XSS-Protection 头部应该已设置")
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"), "X-Content-Type-Options 应该是 nosniff")
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"), "X-XSS-Protection 应该是 1; mode=block")
}

// 测试自定义安全头部配置
func TestSecurityHeaders_CustomConfig(t *testing.T) {
	// 创建带自定义配置的中间件
	headersMiddleware := New(
		WithXFrameOptions("DENY"),
		WithReferrerPolicy("no-referrer"),
		WithHSTS(true, 31536000, true, true),
		WithContentSecurityPolicy("default-src 'self'"),
	)

	// 创建测试处理函数
	testHandler := func(ctx *mist.Context) {
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := headersMiddleware(testHandler)

	// 创建请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
	}

	// 执行处理函数
	handler(ctx)

	// 验证头部设置
	resp := w.Result()
	headers := resp.Header

	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"), "X-Frame-Options 应该是 DENY")
	assert.Equal(t, "no-referrer", headers.Get("Referrer-Policy"), "Referrer-Policy 应该是 no-referrer")

	hstsHeader := headers.Get("Strict-Transport-Security")
	assert.Contains(t, hstsHeader, "max-age=31536000", "HSTS 应该包含 max-age=31536000")
	assert.Contains(t, hstsHeader, "includeSubDomains", "HSTS 应该包含 includeSubDomains")
	assert.Contains(t, hstsHeader, "preload", "HSTS 应该包含 preload")

	assert.Equal(t, "default-src 'self'", headers.Get("Content-Security-Policy"), "CSP 应该是 default-src 'self'")
}

// 测试禁用所有安全头部
func TestSecurityHeaders_DisableAll(t *testing.T) {
	// 创建禁用所有头部的中间件
	headersMiddleware := New(
		WithXSSProtection(false),
		WithContentTypeNoSniff(false),
		WithXFrameOptions(""),
		WithHSTS(false, 0, false, false),
		WithContentSecurityPolicy(""),
		WithReferrerPolicy(""),
		WithPermissionsPolicy(""),
		WithExpectCT(false, 0, false),
		WithCrossOriginPolicies("", "", ""),
	)

	// 创建测试处理函数
	testHandler := func(ctx *mist.Context) {
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := headersMiddleware(testHandler)

	// 创建请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
	}

	// 执行处理函数
	handler(ctx)

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, w.Code, "应该返回200状态码")

	// 验证无安全头部
	resp := w.Result()
	headers := resp.Header

	assert.Empty(t, headers.Get("X-XSS-Protection"), "X-XSS-Protection 不应该设置")
	assert.Empty(t, headers.Get("X-Content-Type-Options"), "X-Content-Type-Options 不应该设置")
	assert.Empty(t, headers.Get("X-Frame-Options"), "X-Frame-Options 不应该设置")
	assert.Empty(t, headers.Get("Strict-Transport-Security"), "Strict-Transport-Security 不应该设置")
	assert.Empty(t, headers.Get("Content-Security-Policy"), "Content-Security-Policy 不应该设置")
	assert.Empty(t, headers.Get("Referrer-Policy"), "Referrer-Policy 不应该设置")
	assert.Empty(t, headers.Get("Permissions-Policy"), "Permissions-Policy 不应该设置")
	assert.Empty(t, headers.Get("Expect-CT"), "Expect-CT 不应该设置")
	assert.Empty(t, headers.Get("Cross-Origin-Embedder-Policy"), "Cross-Origin-Embedder-Policy 不应该设置")
	assert.Empty(t, headers.Get("Cross-Origin-Opener-Policy"), "Cross-Origin-Opener-Policy 不应该设置")
	assert.Empty(t, headers.Get("Cross-Origin-Resource-Policy"), "Cross-Origin-Resource-Policy 不应该设置")
}

// 测试CSPBuilder
func TestCSPBuilder(t *testing.T) {
	// 测试基本功能
	builder := NewCSPBuilder()
	builder.Add("default-src", "'self'")
	builder.Add("script-src", "'self'", "https://example.com")
	builder.Add("img-src", "*")

	csp := builder.String()
	assert.Contains(t, csp, "default-src 'self'", "CSP 应该包含 default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' https://example.com", "CSP 应该包含 script-src 'self' https://example.com")
	assert.Contains(t, csp, "img-src *", "CSP 应该包含 img-src *")

	// 测试空builder
	emptyBuilder := NewCSPBuilder()
	assert.Empty(t, emptyBuilder.String(), "空builder应该返回空字符串")

	// 测试添加空值
	emptyValBuilder := NewCSPBuilder()
	emptyValBuilder.Add("default-src")
	assert.Contains(t, emptyValBuilder.String(), "default-src", "应该包含指令名，即使没有值")
}

// 测试CSPStrict和CSPBasic工厂函数
func TestCSPFactoryFunctions(t *testing.T) {
	// 测试严格CSP
	strictCSP := CSPStrict()
	assert.NotEmpty(t, strictCSP, "严格CSP不应为空")
	assert.Contains(t, strictCSP, "default-src 'self'", "严格CSP应包含default-src 'self'")
	assert.Contains(t, strictCSP, "upgrade-insecure-requests", "严格CSP应包含upgrade-insecure-requests")

	// 测试基础CSP
	basicCSP := CSPBasic()
	assert.NotEmpty(t, basicCSP, "基础CSP不应为空")
	assert.Contains(t, basicCSP, "default-src", "基础CSP应包含default-src")
}

// 测试XFrameAllowFrom工厂函数
func TestXFrameAllowFrom(t *testing.T) {
	// 测试有效URI
	validURI := "https://example.com"
	xframeValue := XFrameAllowFrom(validURI)
	assert.Equal(t, "ALLOW-FROM "+validURI, xframeValue, "应该返回ALLOW-FROM加上URI")

	// 测试空URI - 当前实现返回"ALLOW-FROM "
	emptyValue := XFrameAllowFrom("")
	assert.Equal(t, "ALLOW-FROM ", emptyValue, "对于空URI应返回ALLOW-FROM加空字符串")
}

// 测试HSTSValue工厂函数
func TestHSTSValue(t *testing.T) {
	// 测试基本设置
	basic := HSTSValue(3600, false, false)
	assert.Equal(t, "max-age=3600", basic, "只应包含max-age")

	// 测试包含子域名
	withSub := HSTSValue(3600, true, false)
	assert.Equal(t, "max-age=3600; includeSubDomains", withSub, "应包含includeSubDomains")

	// 测试包含预加载
	withPreload := HSTSValue(3600, false, true)
	assert.Equal(t, "max-age=3600; preload", withPreload, "应包含preload")

	// 测试全部设置
	all := HSTSValue(3600, true, true)
	assert.Equal(t, "max-age=3600; includeSubDomains; preload", all, "应包含所有选项")
}
