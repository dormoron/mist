package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/session"
	"github.com/dormoron/mist/session/cookie"
	"github.com/dormoron/mist/session/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManagerLifecycle 测试会话管理器的完整生命周期
func TestManagerLifecycle(t *testing.T) {
	// 创建内存存储
	store, err := memory.NewStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// 创建自定义cookie传播器
	cookieProp := cookie.NewPropagator("mist_session",
		cookie.WithMaxAge(3600),
		cookie.WithPath("/"),
		cookie.WithHTTPOnly(true),
	)

	// 直接创建管理器
	manager := &session.Manager{
		Store:         store,
		Propagator:    cookieProp,
		CtxSessionKey: "session",
	}

	// 创建HTTP请求和响应
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()

	// 创建mist上下文
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
		UserValues:     make(map[string]any),
	}

	// 测试初始化会话
	sess, err := manager.InitSession(ctx)
	require.NoError(t, err)
	require.NotNil(t, sess)

	// 获取会话ID
	sessionID := sess.ID()
	require.NotEmpty(t, sessionID)

	// 设置会话数据
	err = sess.Set(req.Context(), "user_id", 12345)
	require.NoError(t, err)
	err = sess.Set(req.Context(), "username", "testuser")
	require.NoError(t, err)
	err = sess.Save()
	require.NoError(t, err)

	// 检查ResponseWriter收到的cookie
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "mist_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, sessionID, sessionCookie.Value)
	assert.Equal(t, 3600, sessionCookie.MaxAge)

	// 测试读取现有会话
	req2, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	ctx2 := &mist.Context{
		Request:        req2,
		ResponseWriter: w2,
		UserValues:     make(map[string]any),
	}

	sess2, err := manager.GetSession(ctx2)
	require.NoError(t, err)
	require.NotNil(t, sess2)
	assert.Equal(t, sessionID, sess2.ID())

	// 验证会话数据
	val, err := sess2.Get(req2.Context(), "user_id")
	require.NoError(t, err)
	assert.Equal(t, 12345, val)

	val, err = sess2.Get(req2.Context(), "username")
	require.NoError(t, err)
	assert.Equal(t, "testuser", val)

	// 测试直接从存储中删除会话
	err = store.Remove(context.Background(), sessionID)
	require.NoError(t, err)

	// 测试尝试获取已删除的会话
	req3, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req3.AddCookie(sessionCookie)
	w3 := httptest.NewRecorder()

	ctx3 := &mist.Context{
		Request:        req3,
		ResponseWriter: w3,
		UserValues:     make(map[string]any),
	}

	_, err = manager.GetSession(ctx3)
	assert.Error(t, err)
}

// TestSessionRemoval 测试删除会话
func TestSessionRemoval(t *testing.T) {
	// 创建内存存储
	store, err := memory.NewStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// 创建自定义cookie传播器
	cookieProp := cookie.NewPropagator("test_session",
		cookie.WithMaxAge(3600),
		cookie.WithPath("/"),
		cookie.WithHTTPOnly(true),
	)

	// 测试cookie传播器的Remove方法
	w := httptest.NewRecorder()
	err = cookieProp.Remove(w)
	require.NoError(t, err)

	// 验证cookie已被设置为删除
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var removedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "test_session" {
			removedCookie = c
			break
		}
	}
	require.NotNil(t, removedCookie)
	assert.Equal(t, "", removedCookie.Value)
	assert.Equal(t, -1, removedCookie.MaxAge)
}

// TestManagerAutoGC 测试会话管理器的自动垃圾回收功能
func TestManagerAutoGC(t *testing.T) {
	// 创建具有短期过期时间的内存存储
	store := memory.InitStore(100 * time.Millisecond)
	require.NotNil(t, store)

	// 创建会话管理器
	manager, err := session.NewManager(store, 1)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// 启用自动垃圾回收
	err = manager.EnableAutoGC(50 * time.Millisecond)
	require.NoError(t, err)

	// 创建一些会话
	for i := 0; i < 5; i++ {
		_, err := manager.Create()
		require.NoError(t, err)
	}

	// 等待垃圾回收运行
	time.Sleep(200 * time.Millisecond)

	// 禁用自动垃圾回收
	err = manager.DisableAutoGC()
	require.NoError(t, err)

	// 手动运行垃圾回收
	err = manager.RunGC()
	require.NoError(t, err)
}

// TestManagerSetMaxAge 测试设置会话Cookie的最大有效期
func TestManagerSetMaxAge(t *testing.T) {
	// 创建内存存储
	store, err := memory.NewStore()
	require.NoError(t, err)

	// 创建自定义cookie传播器
	cookieProp := cookie.NewPropagator("test_session",
		cookie.WithMaxAge(3600),
		cookie.WithPath("/"),
		cookie.WithHTTPOnly(true),
	)

	// 直接创建管理器
	manager := &session.Manager{
		Store:         store,
		Propagator:    cookieProp,
		CtxSessionKey: "session",
	}

	// 修改最大有效期
	manager.SetMaxAge(7200)

	// 创建HTTP请求和响应
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()

	// 创建mist上下文
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
		UserValues:     make(map[string]any),
	}

	// 初始化会话
	_, err = manager.InitSession(ctx)
	require.NoError(t, err)

	// 验证Cookie的MaxAge已设置为新值
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "test_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, 7200, sessionCookie.MaxAge)
}

// TestManagerRemoveSession 测试会话管理器的RemoveSession方法
func TestManagerRemoveSession(t *testing.T) {
	// 创建内存存储
	store, err := memory.NewStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// 创建自定义cookie传播器
	cookieProp := cookie.NewPropagator("remove_test_session",
		cookie.WithMaxAge(3600),
		cookie.WithPath("/"),
		cookie.WithHTTPOnly(true),
	)

	// 直接创建管理器
	manager := &session.Manager{
		Store:         store,
		Propagator:    cookieProp,
		CtxSessionKey: "session",
	}

	// 创建HTTP请求和响应
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()

	// 创建mist上下文
	ctx := &mist.Context{
		Request:        req,
		ResponseWriter: w,
		UserValues:     make(map[string]any),
	}

	// 初始化会话
	sess, err := manager.InitSession(ctx)
	require.NoError(t, err)
	require.NotNil(t, sess)

	// 获取会话ID
	sessionID := sess.ID()
	require.NotEmpty(t, sessionID)

	// 设置会话数据
	err = sess.Set(req.Context(), "test_key", "test_value")
	require.NoError(t, err)
	err = sess.Save()
	require.NoError(t, err)

	// 获取cookie
	resp := w.Result()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "remove_test_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// 创建新的请求，带上会话cookie
	req2, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	ctx2 := &mist.Context{
		Request:        req2,
		ResponseWriter: w2,
		UserValues:     make(map[string]any),
	}

	// 调用RemoveSession方法
	err = manager.RemoveSession(ctx2)
	require.NoError(t, err)

	// 验证cookie被标记为删除
	resp2 := w2.Result()
	cookies2 := resp2.Cookies()
	require.NotEmpty(t, cookies2)
	var removedCookie *http.Cookie
	for _, c := range cookies2 {
		if c.Name == "remove_test_session" {
			removedCookie = c
			break
		}
	}
	require.NotNil(t, removedCookie)
	assert.Equal(t, "", removedCookie.Value)
	assert.Less(t, removedCookie.MaxAge, 0)

	// 验证会话已从存储中删除
	_, err = store.Get(context.Background(), sessionID)
	assert.Error(t, err)

	// 确认UserValues中的会话已被删除
	_, ok := ctx2.UserValues[manager.CtxSessionKey]
	assert.False(t, ok)
}
