package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dormoron/mist/session/cookie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookiePropagator 测试Cookie会话传播器的基本功能
func TestCookiePropagator(t *testing.T) {
	// 创建Cookie传播器
	propagator := cookie.NewPropagator("test-session",
		cookie.WithMaxAge(3600),
		cookie.WithSecure(true),
		cookie.WithHTTPOnly(true),
	)
	require.NotNil(t, propagator)

	// 创建响应写入器记录者
	w := httptest.NewRecorder()

	// 测试注入会话ID
	sessionID := "test-session-id-12345"
	err := propagator.Inject(sessionID, w)
	require.NoError(t, err)

	// 验证Cookie已设置
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "test-session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, sessionID, sessionCookie.Value)
	assert.Equal(t, 3600, sessionCookie.MaxAge)
	assert.True(t, sessionCookie.Secure)
	assert.True(t, sessionCookie.HttpOnly)

	// 创建请求用于提取会话ID
	req := &http.Request{
		Header: http.Header{"Cookie": []string{sessionCookie.String()}},
	}

	// 测试提取会话ID
	extractedID, err := propagator.Extract(req)
	require.NoError(t, err)
	assert.Equal(t, sessionID, extractedID)

	// 测试移除会话Cookie
	w = httptest.NewRecorder()
	err = propagator.Remove(w)
	require.NoError(t, err)

	// 验证Cookie已被设置为过期
	resp = w.Result()
	cookies = resp.Cookies()
	require.NotEmpty(t, cookies)

	var removedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "test-session" {
			removedCookie = c
			break
		}
	}
	require.NotNil(t, removedCookie)
	assert.Equal(t, "", removedCookie.Value)
	assert.Equal(t, -1, removedCookie.MaxAge)
}

// TestCookiePropagatorOptions 测试Cookie传播器的配置选项
func TestCookiePropagatorOptions(t *testing.T) {
	// 创建带有自定义选项的Cookie传播器
	propagator := cookie.NewPropagator("custom-session",
		cookie.WithPath("/api"),
		cookie.WithDomain("example.com"),
		cookie.WithMaxAge(7200),
		cookie.WithSecure(true),
		cookie.WithHTTPOnly(true),
		cookie.WithSameSite(http.SameSiteStrictMode),
	)
	require.NotNil(t, propagator)

	// 创建响应写入器记录者
	w := httptest.NewRecorder()

	// 测试注入会话ID
	sessionID := "custom-session-id-67890"
	err := propagator.Inject(sessionID, w)
	require.NoError(t, err)

	// 验证Cookie已设置并带有自定义选项
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "custom-session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, sessionID, sessionCookie.Value)
	assert.Equal(t, "/api", sessionCookie.Path)
	assert.Equal(t, "example.com", sessionCookie.Domain)
	assert.Equal(t, 7200, sessionCookie.MaxAge)
	assert.True(t, sessionCookie.Secure)
	assert.True(t, sessionCookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, sessionCookie.SameSite)

	// 测试动态修改Cookie最大有效期
	propagator.SetMaxAge(1800)

	// 创建新的响应写入器记录者
	w = httptest.NewRecorder()

	// 重新注入会话ID
	err = propagator.Inject(sessionID, w)
	require.NoError(t, err)

	// 验证Cookie的MaxAge已被更新
	resp = w.Result()
	cookies = resp.Cookies()
	require.NotEmpty(t, cookies)

	sessionCookie = nil
	for _, c := range cookies {
		if c.Name == "custom-session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, 1800, sessionCookie.MaxAge)
}
