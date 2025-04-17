package bodylimit

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/dormoron/mist"
)

// BodyLimitConfig 配置请求体大小限制中间件
type BodyLimitConfig struct {
	// 最大允许大小(字节数)
	MaxSize int64

	// 响应状态码，默认413 Request Entity Too Large
	StatusCode int

	// 超过限制时的错误消息
	ErrorMessage string

	// 白名单路径 - 不受限制的路径前缀
	WhitelistPaths []string

	// 跳过OPTIONS/HEAD请求的检查
	SkipOptions bool
	SkipHead    bool

	// 自定义检查是否需要限制的函数
	SkipFunc func(ctx *mist.Context) bool
}

// DefaultBodyLimitConfig 返回默认配置
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		MaxSize:      1 * 1024 * 1024, // 默认1MB
		StatusCode:   http.StatusRequestEntityTooLarge,
		ErrorMessage: "请求体超过允许的大小限制",
		SkipOptions:  true,
		SkipHead:     true,
	}
}

// BodyLimit 创建请求体大小限制中间件
func BodyLimit(maxSize string) mist.Middleware {
	size, err := parseSize(maxSize)
	if err != nil {
		panic(fmt.Sprintf("无效的大小限制: %v", err))
	}

	config := DefaultBodyLimitConfig()
	config.MaxSize = size

	return BodyLimitWithConfig(config)
}

// BodyLimitWithConfig 使用自定义配置创建请求体大小限制中间件
func BodyLimitWithConfig(config BodyLimitConfig) mist.Middleware {
	// 使用默认值填充未设置的配置
	if config.MaxSize <= 0 {
		config.MaxSize = DefaultBodyLimitConfig().MaxSize
	}

	if config.StatusCode <= 0 {
		config.StatusCode = DefaultBodyLimitConfig().StatusCode
	}

	if config.ErrorMessage == "" {
		config.ErrorMessage = DefaultBodyLimitConfig().ErrorMessage
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 检查是否需要跳过限制
			if shouldSkip(ctx, config) {
				next(ctx)
				return
			}

			// 检查Content-Length头
			contentLength := ctx.Request.ContentLength
			if contentLength > config.MaxSize {
				ctx.AbortWithStatus(config.StatusCode)
				ctx.RespondWithJSON(config.StatusCode, map[string]interface{}{
					"error": config.ErrorMessage,
					"limit": config.MaxSize,
					"size":  contentLength,
				})
				return
			}

			// 限制请求体的大小
			ctx.Request.Body = limitedReader(ctx.Request.Body, config.MaxSize, ctx, config)

			next(ctx)
		}
	}
}

// shouldSkip 检查是否应该跳过该请求的限制
func shouldSkip(ctx *mist.Context, config BodyLimitConfig) bool {
	// 检查自定义跳过函数
	if config.SkipFunc != nil && config.SkipFunc(ctx) {
		return true
	}

	// 检查HTTP方法
	method := ctx.Request.Method
	if (config.SkipOptions && method == http.MethodOptions) ||
		(config.SkipHead && method == http.MethodHead) {
		return true
	}

	// 检查白名单路径
	if len(config.WhitelistPaths) > 0 {
		path := ctx.Request.URL.Path
		for _, prefix := range config.WhitelistPaths {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}

	return false
}

// limitedReader 返回一个受限制的读取器
func limitedReader(body io.ReadCloser, limit int64, ctx *mist.Context, config BodyLimitConfig) io.ReadCloser {
	return &limitedReadCloser{
		ReadCloser: body,
		limit:      limit,
		ctx:        ctx,
		config:     config,
		read:       0,
	}
}

// limitedReadCloser 是一个限制大小的ReadCloser实现
type limitedReadCloser struct {
	io.ReadCloser
	limit  int64
	read   int64
	ctx    *mist.Context
	config BodyLimitConfig
}

// Read 实现io.Reader接口，限制读取的总大小
func (l *limitedReadCloser) Read(p []byte) (n int, err error) {
	n, err = l.ReadCloser.Read(p)
	l.read += int64(n)

	// 检查是否超过限制
	if l.read > l.limit {
		// 中止请求
		l.ctx.AbortWithStatus(l.config.StatusCode)
		_ = l.ctx.RespondWithJSON(l.config.StatusCode, map[string]interface{}{
			"error": l.config.ErrorMessage,
			"limit": l.limit,
		})
		return n, fmt.Errorf("请求体超过大小限制: %d > %d", l.read, l.limit)
	}

	return n, err
}

// parseSize 解析人类可读的大小字符串
// 支持单位: B, K/KB, M/MB, G/GB
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("空大小字符串")
	}

	// 查找单位分隔符
	var numStr string
	var unit string

	if strings.HasSuffix(sizeStr, "B") {
		if len(sizeStr) > 2 && (sizeStr[len(sizeStr)-2] == 'K' ||
			sizeStr[len(sizeStr)-2] == 'M' ||
			sizeStr[len(sizeStr)-2] == 'G') {
			numStr = sizeStr[:len(sizeStr)-2]
			unit = sizeStr[len(sizeStr)-2:]
		} else {
			numStr = sizeStr[:len(sizeStr)-1]
			unit = "B"
		}
	} else if strings.HasSuffix(sizeStr, "K") ||
		strings.HasSuffix(sizeStr, "M") ||
		strings.HasSuffix(sizeStr, "G") {
		numStr = sizeStr[:len(sizeStr)-1]
		unit = sizeStr[len(sizeStr)-1:]
	} else {
		// 假设为纯数字
		numStr = sizeStr
		unit = "B"
	}

	// 解析数字部分
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的大小格式: %v", err)
	}

	// 应用单位
	switch unit {
	case "B":
		return int64(size), nil
	case "K", "KB":
		return int64(size * 1024), nil
	case "M", "MB":
		return int64(size * 1024 * 1024), nil
	case "G", "GB":
		return int64(size * 1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("未知的大小单位: %s", unit)
	}
}
