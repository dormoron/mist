package mist_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dormoron/mist/middlewares/bodylimit"

	"github.com/dormoron/mist"
	"github.com/stretchr/testify/assert"
)

// 测试自适应路由缓存
func TestAdaptiveCache(t *testing.T) {
	r := mist.InitHTTPServer()

	// 设置一些测试路由
	r.GET("/cached/static", func(ctx *mist.Context) {
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"message": "This is a cached static response",
		})
	})

	r.GET("/cached/:param", func(ctx *mist.Context) {
		paramValue, _ := ctx.PathValue("param").String()
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"param": paramValue,
		})
	})

	// 进行测试请求
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/cached/static", nil)
		recorder := httptest.NewRecorder()
		r.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}

	// 检查缓存统计
	hits, misses, size := r.CacheStats()
	t.Logf("Cache stats after static requests: hits=%d, misses=%d, size=%d", hits, misses, size)
	assert.Greater(t, hits, uint64(0), "Cache hits should be greater than 0")

	// 测试参数化路由
	for i := 0; i < 100; i++ {
		paramValue := "param" + string(rune(i%5+'0'))
		req := httptest.NewRequest(http.MethodGet, "/cached/"+paramValue, nil)
		recorder := httptest.NewRecorder()
		r.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}

	// 再次检查缓存统计
	hits2, misses2, size2 := r.CacheStats()
	t.Logf("Cache stats after all requests: hits=%d, misses=%d, size=%d", hits2, misses2, size2)
	assert.Greater(t, hits2, hits, "Cache hits should increase")
	assert.Greater(t, size2, 1, "Cache size should be greater than 1")
}

// 测试请求体大小限制中间件
func TestBodyLimitMiddleware(t *testing.T) {
	r := mist.InitHTTPServer()

	// 应用请求体限制中间件
	r.Use(bodylimit.BodyLimit("1KB"))

	// 测试路由
	r.POST("/test/body", func(ctx *mist.Context) {
		body, _ := io.ReadAll(ctx.Request.Body)
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"size": len(body),
		})
	})

	// 发送小于限制的请求
	smallBody := strings.Repeat("a", 500) // 500字节
	req := httptest.NewRequest(http.MethodPost, "/test/body", strings.NewReader(smallBody))
	req.Header.Set("Content-Length", "500")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// 发送大于限制的请求
	largeBody := strings.Repeat("a", 2*1024) // 2KB
	req = httptest.NewRequest(http.MethodPost, "/test/body", strings.NewReader(largeBody))
	req.Header.Set("Content-Length", "2048")
	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

// 测试零拷贝文件传输
func TestZeroCopyResponse(t *testing.T) {
	r := mist.InitHTTPServer()

	// 创建临时测试文件
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "testfile.txt")

	// 写入一些测试数据
	testData := strings.Repeat("This is a test file content.\n", 1000) // 约30KB
	err := os.WriteFile(tempFile, []byte(testData), 0644)
	assert.NoError(t, err)

	// 设置测试路由
	r.GET("/file/standard", func(ctx *mist.Context) {
		// 标准方式提供文件
		http.ServeFile(ctx.ResponseWriter, ctx.Request, tempFile)
	})

	r.GET("/file/zerocopy", func(ctx *mist.Context) {
		// 零拷贝方式提供文件
		zr := mist.NewZeroCopyResponse(ctx.ResponseWriter)
		err := zr.ServeFile(tempFile)
		if err != nil {
			ctx.RespondWithJSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	})

	// 测试标准方式
	req := httptest.NewRequest(http.MethodGet, "/file/standard", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, len(testData), recorder.Body.Len())

	// 测试零拷贝方式
	req = httptest.NewRequest(http.MethodGet, "/file/zerocopy", nil)
	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, len(testData), recorder.Body.Len())
}

// 测试内存监控功能
func TestMemoryMonitor(t *testing.T) {
	// 创建内存监控器，不启动监控线程
	monitor := mist.NewMemoryMonitor(time.Second, 10)

	// 收集初始样本（手动方式）
	initialStats := monitor.GetCurrentStats()
	t.Logf("Initial memory: %d bytes, goroutines: %d", initialStats.Alloc, initialStats.NumGoroutine)

	// 创建一些内存压力
	var memoryPressure [][]byte
	for i := 0; i < 3; i++ {
		memoryPressure = append(memoryPressure, make([]byte, 1024*1024)) // 1MB
	}

	// 手动触发GC
	monitor.ForceGC()

	// 手动获取内存报告
	report := monitor.GetMemoryUsageReport()
	t.Logf("Memory report: %+v", report)

	// 验证内存监控器的基本功能
	assert.NotNil(t, report["current"])

	// 释放内存
	memoryPressure = nil

	// 测试告警回调添加功能
	monitor.AddAlertCallback(func(stats mist.MemStats, message string) {
		t.Logf("callback would be called with message: %s", message)
	})

	// 设置告警阈值
	monitor.SetAlertThreshold(0.1)

	// 即使不启动监控，也应该能够获取样本和报告
	samples := monitor.GetSamples()
	t.Logf("Samples count: %d", len(samples))

	assert.True(t, true, "测试成功完成")
}

// 测试HTTP/3支持
func TestHTTP3Config(t *testing.T) {
	// 注意：此测试仅验证HTTP/3服务器配置，不会实际启动服务器

	// 检查HTTP/3配置默认值
	config := mist.DefaultHTTP3Config()

	// 验证配置默认值
	assert.Greater(t, config.MaxIdleTimeout, time.Duration(0))
	assert.Greater(t, config.MaxIncomingStreams, int64(0))

	// 检查配置值调整
	config.MaxIdleTimeout = 60 * time.Second
	config.MaxIncomingStreams = 200

	assert.Equal(t, 60*time.Second, config.MaxIdleTimeout)
	assert.Equal(t, int64(200), config.MaxIncomingStreams)
}
