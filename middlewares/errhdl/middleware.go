package errhdl

import (
	"encoding/json"
	"fmt"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/internal/errs"
)

// ErrorHandlerFunc 定义错误处理函数类型
type ErrorHandlerFunc func(ctx *mist.Context, err error)

// Config 错误处理中间件配置
type Config struct {
	// 是否在生产环境（生产环境不返回详细错误信息）
	IsProduction bool

	// 是否记录错误日志
	LogErrors bool

	// 自定义错误处理函数，按错误类型映射
	CustomHandlers map[errs.ErrorType]ErrorHandlerFunc

	// 全局错误处理函数，处理所有错误
	GlobalHandler ErrorHandlerFunc
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		IsProduction:   false,
		LogErrors:      true,
		CustomHandlers: make(map[errs.ErrorType]ErrorHandlerFunc),
	}
}

// Recovery 创建一个错误处理中间件
func Recovery(config ...Config) mist.Middleware {
	// 使用默认配置
	cfg := DefaultConfig()

	// 应用自定义配置
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			defer func() {
				if r := recover(); r != nil {
					// 处理panic
					var err error
					switch v := r.(type) {
					case error:
						err = v
					case string:
						err = fmt.Errorf(v)
					default:
						err = fmt.Errorf("%v", r)
					}

					// 记录错误日志
					if cfg.LogErrors {
						mist.Error("Recovery middleware caught panic: %v", err)
					}

					// 处理错误
					handleError(ctx, err, cfg)
				}
			}()

			// 继续处理请求
			next(ctx)

			// 检查是否有错误状态码
			if ctx.RespStatusCode >= 400 {
				var err error
				if ctx.RespData != nil && len(ctx.RespData) > 0 {
					err = fmt.Errorf(string(ctx.RespData))
				} else {
					err = fmt.Errorf("HTTP error %d", ctx.RespStatusCode)
				}

				// 处理错误状态码
				handleError(ctx, err, cfg)
			}
		}
	}
}

// ErrorHandler 创建一个处理特定类型错误的中间件
func TypedErrorHandler(errorType errs.ErrorType, handler ErrorHandlerFunc) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 继续处理请求
			next(ctx)

			// 如果发生错误并且返回的是APIError
			if ctx.RespStatusCode >= 400 && ctx.RespData != nil {
				var apiErr *errs.APIError
				if len(ctx.RespData) > 0 {
					// 尝试解析JSON错误
					err := json.Unmarshal(ctx.RespData, &apiErr)
					if err == nil && apiErr != nil && apiErr.Type == errorType {
						// 调用自定义处理函数
						handler(ctx, apiErr)
					}
				}
			}
		}
	}
}

// handleError 统一处理错误的内部函数
func handleError(ctx *mist.Context, err error, cfg Config) {
	// 转换为API错误
	apiErr := errs.WrapError(err)

	// 在生产环境下隐藏详细错误信息
	if cfg.IsProduction {
		apiErr.Details = nil
	}

	// 检查是否有自定义处理函数
	if handler, ok := cfg.CustomHandlers[apiErr.Type]; ok {
		handler(ctx, apiErr)
		return
	}

	// 检查是否有全局处理函数
	if cfg.GlobalHandler != nil {
		cfg.GlobalHandler(ctx, apiErr)
		return
	}

	// 默认错误处理
	ctx.RespStatusCode = apiErr.Code
	ctx.RespData = apiErr.ToJSON()
	ctx.Header("Content-Type", "application/json")
}
