package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dormoron/mist/session/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryStore 测试内存会话存储的基本功能
func TestMemoryStore(t *testing.T) {
	// 创建内存存储
	store, err := memory.NewStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// 测试上下文
	ctx := context.Background()

	// 生成新会话
	sessionID := "test-memory-session-1"
	sess, err := store.Generate(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, sessionID, sess.ID())

	// 测试设置会话值
	err = sess.Set(ctx, "username", "testuser")
	require.NoError(t, err)
	assert.True(t, sess.IsModified())

	// 测试获取会话值
	val, err := sess.Get(ctx, "username")
	require.NoError(t, err)
	assert.Equal(t, "testuser", val)

	// 测试保存会话
	err = sess.Save()
	require.NoError(t, err)
	assert.False(t, sess.IsModified())

	// 测试刷新会话
	err = store.Refresh(ctx, sessionID)
	require.NoError(t, err)

	// 测试获取现有会话
	retrievedSess, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, retrievedSess)
	assert.Equal(t, sessionID, retrievedSess.ID())

	// 测试从获取的会话中获取值
	val, err = retrievedSess.Get(ctx, "username")
	require.NoError(t, err)
	assert.Equal(t, "testuser", val)

	// 测试删除会话键
	err = retrievedSess.Delete(ctx, "username")
	require.NoError(t, err)
	assert.True(t, retrievedSess.IsModified())

	// 验证键已被删除
	val, err = retrievedSess.Get(ctx, "username")
	require.NoError(t, err)
	assert.Nil(t, val)

	// 测试设置最大存活时间
	retrievedSess.SetMaxAge(3600) // 1小时
	err = retrievedSess.Save()
	require.NoError(t, err)
	assert.False(t, retrievedSess.IsModified())

	// 测试删除会话
	err = store.Remove(ctx, sessionID)
	require.NoError(t, err)

	// 验证会话已被删除
	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err)

	// 测试垃圾回收
	err = store.GC(ctx)
	require.NoError(t, err)
}

// TestMemoryStoreExpiration 测试内存会话存储的过期功能
func TestMemoryStoreExpiration(t *testing.T) {
	// 创建具有短期过期时间的内存存储
	store := memory.InitStore(100 * time.Millisecond)
	require.NotNil(t, store)

	// 测试上下文
	ctx := context.Background()

	// 生成新会话
	sessionID := "test-expiring-session"
	sess, err := store.Generate(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sess)

	// 设置会话值
	err = sess.Set(ctx, "temp", "value")
	require.NoError(t, err)

	// 等待会话过期
	time.Sleep(200 * time.Millisecond)

	// 验证会话已过期
	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err)

	// 运行垃圾回收确保资源被释放
	err = store.GC(ctx)
	require.NoError(t, err)
}

// TestConcurrentAccess 测试并发访问内存会话存储
func TestConcurrentAccess(t *testing.T) {
	store, err := memory.NewStore()
	require.NoError(t, err)
	require.NotNil(t, store)

	// 测试上下文
	ctx := context.Background()

	// 生成新会话
	sessionID := "test-concurrent-session"
	sess, err := store.Generate(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sess)

	// 并发读写测试
	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // 读和写操作的goroutines数量

	// 并发写入
	for i := 0; i < goroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", routineID, j)
				err := sess.Set(ctx, key, j)
				assert.NoError(t, err)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < goroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", routineID, j)
				_, err := sess.Get(ctx, key)
				// 不断言值，因为可能还没有被写入
				if err != nil {
					// 忽略错误，因为可能是键不存在
					_ = err
				}
			}
		}(i)
	}

	wg.Wait()

	// 保存会话
	err = sess.Save()
	require.NoError(t, err)

	// 清理
	err = store.Remove(ctx, sessionID)
	require.NoError(t, err)
}
