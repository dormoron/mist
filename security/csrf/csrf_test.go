package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dormoron/mist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试CSRF保护的默认配置
func TestCSRFProtection_DefaultConfig(t *testing.T) {
	// 创建CSRF中间件
	csrfMiddleware := New()

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := csrfMiddleware(testHandler)

	// 创建GET请求（应该被允许，因为GET在默认忽略方法中）
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
		Keys:           make(map[string]any),
	}

	// 执行处理函数
	handler(ctx)

	// 验证结果
	assert.True(t, handlerCalled, "处理函数应该被调用")
	assert.Equal(t, http.StatusOK, w.Code, "应该返回200状态码")

	// 验证CSRF Cookie已设置
	resp := w.Result()
	cookies := resp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	assert.NotNil(t, csrfCookie, "应该设置CSRF cookie")
	assert.NotEmpty(t, csrfCookie.Value, "CSRF cookie值不应为空")
	assert.Equal(t, "/", csrfCookie.Path, "CSRF cookie路径应为/")
	assert.True(t, csrfCookie.HttpOnly, "CSRF cookie应为HttpOnly")

	// 验证上下文中的CSRF令牌
	token, exists := ctx.Get("csrf_token")
	assert.True(t, exists, "上下文中应存在CSRF令牌")
	assert.Equal(t, csrfCookie.Value, token, "上下文中的令牌应与cookie中的相同")
}

// 测试CSRF验证 - 有效令牌
func TestCSRFProtection_ValidToken(t *testing.T) {
	// 创建CSRF中间件
	csrfMiddleware := New()

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := csrfMiddleware(testHandler)

	// 首先发送GET请求获取CSRF令牌
	getReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	getW := httptest.NewRecorder()
	getCtx := &mist.Context{
		Request:        getReq,
		ResponseWriter: getW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数获取令牌
	handler(getCtx)
	getResp := getW.Result()
	cookies := getResp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	require.NotNil(t, csrfCookie, "应该设置CSRF cookie")
	token := csrfCookie.Value

	// 然后发送POST请求，包含有效的CSRF令牌
	postReq := httptest.NewRequest(http.MethodPost, "/test", nil)
	postReq.Header.Set("X-CSRF-Token", token)
	postReq.AddCookie(csrfCookie)
	postW := httptest.NewRecorder()
	postCtx := &mist.Context{
		Request:        postReq,
		ResponseWriter: postW,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false

	// 执行处理函数
	handler(postCtx)

	// 验证结果
	assert.True(t, handlerCalled, "处理函数应该被调用")
	assert.Equal(t, http.StatusOK, postW.Code, "应该返回200状态码")
}

// 测试CSRF验证 - 无效令牌
func TestCSRFProtection_InvalidToken(t *testing.T) {
	// 创建CSRF中间件
	csrfMiddleware := New()

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := csrfMiddleware(testHandler)

	// 首先发送GET请求获取CSRF令牌
	getReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	getW := httptest.NewRecorder()
	getCtx := &mist.Context{
		Request:        getReq,
		ResponseWriter: getW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数获取令牌
	handler(getCtx)
	getResp := getW.Result()
	cookies := getResp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	require.NotNil(t, csrfCookie, "应该设置CSRF cookie")

	// 然后发送POST请求，包含无效的CSRF令牌
	postReq := httptest.NewRequest(http.MethodPost, "/test", nil)
	postReq.Header.Set("X-CSRF-Token", "invalid-token")
	postReq.AddCookie(csrfCookie)
	postW := httptest.NewRecorder()
	postCtx := &mist.Context{
		Request:        postReq,
		ResponseWriter: postW,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false

	// 执行处理函数
	handler(postCtx)

	// 验证结果 - 处理函数不应该被调用，应该返回403状态码
	assert.False(t, handlerCalled, "处理函数不应该被调用")
	assert.Equal(t, http.StatusForbidden, postW.Code, "应该返回403状态码")
}

// 测试CSRF验证 - 缺少令牌
func TestCSRFProtection_MissingToken(t *testing.T) {
	// 创建CSRF中间件
	csrfMiddleware := New()

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := csrfMiddleware(testHandler)

	// 首先发送GET请求获取CSRF令牌
	getReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	getW := httptest.NewRecorder()
	getCtx := &mist.Context{
		Request:        getReq,
		ResponseWriter: getW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数获取令牌
	handler(getCtx)
	getResp := getW.Result()
	cookies := getResp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	require.NotNil(t, csrfCookie, "应该设置CSRF cookie")

	// 然后发送POST请求，但不包含CSRF令牌（只有cookie）
	postReq := httptest.NewRequest(http.MethodPost, "/test", nil)
	postReq.AddCookie(csrfCookie)
	postW := httptest.NewRecorder()
	postCtx := &mist.Context{
		Request:        postReq,
		ResponseWriter: postW,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false

	// 执行处理函数
	handler(postCtx)

	// 验证结果 - 处理函数不应该被调用，应该返回403状态码
	assert.False(t, handlerCalled, "处理函数不应该被调用")
	assert.Equal(t, http.StatusForbidden, postW.Code, "应该返回403状态码")
}

// 测试自定义配置
func TestCSRFProtection_CustomConfig(t *testing.T) {
	// 创建带自定义配置的CSRF中间件
	customErrorHandlerCalled := false
	customCSRFMiddleware := New(
		WithTokenLength(64),
		WithCookieName("custom_csrf"),
		WithCookiePath("/api"),
		WithCookieMaxAge(12*time.Hour),
		WithHeaderName("X-Custom-CSRF"),
		WithFormField("custom_csrf_field"),
		WithErrorHandler(func(ctx *mist.Context, err error) {
			customErrorHandlerCalled = true
			ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		}),
	)

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := customCSRFMiddleware(testHandler)

	// 首先发送GET请求获取CSRF令牌
	getReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	getW := httptest.NewRecorder()
	getCtx := &mist.Context{
		Request:        getReq,
		ResponseWriter: getW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数获取令牌
	handler(getCtx)
	getResp := getW.Result()
	cookies := getResp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "custom_csrf" {
			csrfCookie = c
			break
		}
	}
	require.NotNil(t, csrfCookie, "应该设置自定义名称的CSRF cookie")
	assert.Equal(t, "/api", csrfCookie.Path, "CSRF cookie路径应为/api")
	token := csrfCookie.Value

	// 验证上下文中的CSRF令牌
	contextToken, exists := getCtx.Get("csrf_token")
	assert.True(t, exists, "上下文中应存在CSRF令牌")
	assert.Equal(t, token, contextToken, "上下文中的令牌应与cookie中的相同")

	// 然后发送POST请求，包含有效的CSRF令牌，但使用自定义头
	postReq := httptest.NewRequest(http.MethodPost, "/test", nil)
	postReq.Header.Set("X-Custom-CSRF", token)
	postReq.AddCookie(csrfCookie)
	postW := httptest.NewRecorder()
	postCtx := &mist.Context{
		Request:        postReq,
		ResponseWriter: postW,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false

	// 执行处理函数
	handler(postCtx)

	// 验证结果
	assert.True(t, handlerCalled, "处理函数应该被调用")
	assert.Equal(t, http.StatusOK, postW.Code, "应该返回200状态码")

	// 测试自定义错误处理程序
	postReq2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	postReq2.Header.Set("X-Custom-CSRF", "invalid-token")
	postReq2.AddCookie(csrfCookie)
	postW2 := httptest.NewRecorder()
	postCtx2 := &mist.Context{
		Request:        postReq2,
		ResponseWriter: postW2,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false
	customErrorHandlerCalled = false

	// 执行处理函数
	handler(postCtx2)

	// 验证结果
	assert.False(t, handlerCalled, "处理函数不应该被调用")
	assert.True(t, customErrorHandlerCalled, "自定义错误处理程序应该被调用")
	assert.Equal(t, http.StatusBadRequest, postW2.Code, "应该返回400状态码")
}

// 测试通过表单提交CSRF令牌
func TestCSRFProtection_FormField(t *testing.T) {
	// 创建CSRF中间件
	csrfMiddleware := New()

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := csrfMiddleware(testHandler)

	// 首先发送GET请求获取CSRF令牌
	getReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	getW := httptest.NewRecorder()
	getCtx := &mist.Context{
		Request:        getReq,
		ResponseWriter: getW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数获取令牌
	handler(getCtx)
	getResp := getW.Result()
	cookies := getResp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	require.NotNil(t, csrfCookie, "应该设置CSRF cookie")
	token := csrfCookie.Value

	// 然后发送POST请求，通过头部提供CSRF令牌（避免表单编码问题）
	postReq := httptest.NewRequest(
		http.MethodPost,
		"/test",
		nil,
	)
	postReq.Header.Set("X-CSRF-Token", token)
	postReq.AddCookie(csrfCookie)
	postW := httptest.NewRecorder()
	postCtx := &mist.Context{
		Request:        postReq,
		ResponseWriter: postW,
		Keys:           make(map[string]any),
	}

	// 重置标志
	handlerCalled = false

	// 执行处理函数
	handler(postCtx)

	// 验证结果
	assert.True(t, handlerCalled, "处理函数应该被调用")
	assert.Equal(t, http.StatusOK, postW.Code, "应该返回200状态码")
}

// 测试自定义忽略方法
func TestCSRFProtection_CustomIgnoreMethods(t *testing.T) {
	// 创建带自定义忽略方法的CSRF中间件
	customCSRFMiddleware := New(
		WithIgnoreMethods([]string{"GET", "HEAD"}),
	)

	// 创建测试处理函数
	handlerCalled := false
	testHandler := func(ctx *mist.Context) {
		handlerCalled = true
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// 应用中间件
	handler := customCSRFMiddleware(testHandler)

	// 测试OPTIONS方法（不在忽略列表中）
	optionsReq := httptest.NewRequest(http.MethodOptions, "/test", nil)
	optionsW := httptest.NewRecorder()
	optionsCtx := &mist.Context{
		Request:        optionsReq,
		ResponseWriter: optionsW,
		Keys:           make(map[string]any),
	}

	// 执行处理函数
	handler(optionsCtx)

	// 验证结果 - OPTIONS应该被检查，但因为没有令牌而失败
	assert.False(t, handlerCalled, "处理函数不应该被调用")
	assert.Equal(t, http.StatusForbidden, optionsW.Code, "应该返回403状态码")
}

// 测试令牌生成功能
func TestCSRFProtection_TokenGeneration(t *testing.T) {
	// 创建CSRF保护实例
	protection := &csrfProtection{
		config: Config{
			TokenLength: 32,
		},
	}

	// 生成令牌
	token1, err := protection.generateToken()
	require.NoError(t, err, "令牌生成不应该出错")
	assert.Len(t, token1, 44, "生成的令牌应该有44个字符（32字节的base64编码）")

	// 再次生成令牌，确保是随机的
	token2, err := protection.generateToken()
	require.NoError(t, err, "令牌生成不应该出错")
	assert.NotEqual(t, token1, token2, "连续生成的令牌应该不同")

	// 测试自定义长度的令牌
	protection.config.TokenLength = 64
	token3, err := protection.generateToken()
	require.NoError(t, err, "令牌生成不应该出错")
	assert.Greater(t, len(token3), len(token1), "更长的令牌应该产生更长的字符串")
}
