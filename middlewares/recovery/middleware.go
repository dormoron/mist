package recovery

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/dormoron/mist"
)

// StackTraceHandler 是一个处理堆栈跟踪的函数类型
type StackTraceHandler func(stackTrace []string, ctx *mist.Context, err any)

// MiddlewareBuilder is a struct that encapsulates configurations for building middleware.
// It is designed to be used for creating middleware that can handle errors, log requests,
// and manage HTTP response behavior in a flexible and customizable way within an HTTP server
// using the mist framework.
//
// Fields:
//
//   - StatusCode: An integer that represents the default HTTP status code to be sent out when
//     an error occurs within the middleware. This is a convenience for setting a
//     uniform error response code.
//
//   - ErrMsg: A byte slice that holds the default error message to be sent out with the response
//     when an error is encountered. This message is meant to be generic and not disclose
//     sensitive error details that might be exploited.
//
//   - LogFunc: A function that provides logging capabilities for the middleware. It accepts a
//     context object from the mist framework (commonly as `ctx`) which contains details
//     about the request, and an error object (`err`) that may have been captured during
//     the request's lifecycle. It's user-definable, so it can be tailored to log whatever
//     information is necessary in a particular format or to a particular logging sink.
//
// Usage:
//   - An instance of MiddlewareBuilder can be initialized directly with desired configurations,
//     or it may be set up via a constructor-like function which provides defaults that can then
//     be modified.
//
// Examples:
//   - var builder = MiddlewareBuilder{
//     StatusCode: 500, // Set a generic 500 Internal Server Error status code by default.
//     ErrMsg: []byte("An unexpected error occurred"), // Set a generic error message.
//     LogFunc: func(ctx *mist.Context, err any) {
//     // A custom log function that logs to the standard output including the error and request path.
//     fmt.Printf("Error: %v, Path: %s\n", err, ctx.Request.Path)
//     },
//     }
type MiddlewareBuilder struct {
	// StatusCode is the HTTP status code used when an error needs to be conveyed to the client.
	StatusCode int

	// ErrMsg is the content that will be sent in the HTTP response body when an error occurs.
	ErrMsg []byte

	// LogFunc is a callback function to be executed when logging is required. For example,
	// this function can be called upon an error to log the incident for monitoring or debugging.
	LogFunc func(ctx *mist.Context, err any)

	// PrintStack 是否打印堆栈信息
	PrintStack bool

	// StackSize 堆栈大小限制
	StackSize int

	// StackTraceHandler 堆栈跟踪处理函数
	StackTraceHandler StackTraceHandler
}

// InitMiddlewareBuilder initializes a new MiddlewareBuilder with the specified status code and error message.
func InitMiddlewareBuilder(statusCode int, errMsg []byte) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		StatusCode:        statusCode,
		ErrMsg:            errMsg,
		LogFunc:           defaultLogFunc,
		PrintStack:        true,
		StackSize:         4096,
		StackTraceHandler: defaultStackTraceHandler,
	}
}

// SetLogFunc sets the logging function to be used by the middleware.
func (m *MiddlewareBuilder) SetLogFunc(logFunc func(ctx *mist.Context, err any)) *MiddlewareBuilder {
	m.LogFunc = logFunc
	return m
}

// SetPrintStack 设置是否打印堆栈信息
func (m *MiddlewareBuilder) SetPrintStack(printStack bool) *MiddlewareBuilder {
	m.PrintStack = printStack
	return m
}

// SetStackSize 设置堆栈大小限制
func (m *MiddlewareBuilder) SetStackSize(stackSize int) *MiddlewareBuilder {
	m.StackSize = stackSize
	return m
}

// SetStackTraceHandler 设置堆栈跟踪处理函数
func (m *MiddlewareBuilder) SetStackTraceHandler(handler StackTraceHandler) *MiddlewareBuilder {
	m.StackTraceHandler = handler
	return m
}

// defaultLogFunc is the default logging function used by the middleware.
// It logs a message with a timestamp to standard output.
func defaultLogFunc(ctx *mist.Context, err any) {
	mist.Error("Recovery middleware caught panic: %v", err)
}

// defaultStackTraceHandler 默认堆栈跟踪处理函数
func defaultStackTraceHandler(stackTrace []string, ctx *mist.Context, err any) {
	stackStr := strings.Join(stackTrace, "\n")
	mist.Error("Stack trace for panic [%v]:\n%s", err, stackStr)
}

// Build creates and returns a mist.Middleware based on the configurations provided in the MiddlewareBuilder.
// The returned middleware is responsible for recovering from panics that may occur in the HTTP request
// handling cycle, logging the error, and returning a specified error response to the client.
//
// Returns:
// - A configured middleware function that incorporates error recovery and logging as defined in MiddlewareBuilder.
//
// Usage:
//   - mw := builder.Build()
//     After calling Build on a MiddlewareBuilder instance, the resulting middleware can be applied to an HTTP route.
func (m *MiddlewareBuilder) Build() mist.Middleware {
	// Construct and return the middleware function.
	return func(next mist.HandleFunc) mist.HandleFunc {
		// Return a new handler function encapsulating the middleware logic.
		return func(ctx *mist.Context) {
			// Use deferring and recover to catch any panics that occur during the HTTP handling cycle.
			defer func() {
				if err := recover(); err != nil {
					// 获取请求信息
					reqID := ctx.RequestID()
					method := ctx.Request.Method
					path := ctx.Request.URL.Path

					// 收集堆栈跟踪
					if m.PrintStack {
						// 分配堆栈缓冲区
						buf := make([]byte, m.StackSize)
						n := runtime.Stack(buf, false)
						stackTrace := strings.Split(string(buf[:n]), "\n")

						// 调用堆栈处理函数
						if m.StackTraceHandler != nil {
							m.StackTraceHandler(stackTrace, ctx, err)
						}
					}

					// 记录错误信息
					mist.WithField("request_id", reqID).
						WithField("method", method).
						WithField("path", path).
						WithField("error", fmt.Sprintf("%v", err)).
						Error("服务器异常")

					// 设置响应
					ctx.AbortWithStatus(m.StatusCode)

					// In case of panic, set the context response data and status code to the ones specified in MiddlewareBuilder.
					ctx.RespData = m.ErrMsg
					ctx.RespStatusCode = m.StatusCode

					// 添加请求ID以便跟踪
					ctx.ResponseWriter.Header().Set("X-Request-ID", reqID)

					// Use LogFunc to log the error along with context information.
					m.LogFunc(ctx, err)
				}
			}()
			// Call the next middleware/handler in the chain.
			next(ctx)
		}
	}
}

// NewRecoveryMiddleware 创建一个新的恢复中间件，使用默认配置
func NewRecoveryMiddleware() mist.Middleware {
	return InitMiddlewareBuilder(
		http.StatusInternalServerError,
		[]byte(`{"error":"服务器内部错误，请稍后重试"}`),
	).Build()
}

// JSONRecoveryMiddleware 创建一个返回JSON错误的恢复中间件
func JSONRecoveryMiddleware() mist.Middleware {
	builder := InitMiddlewareBuilder(
		http.StatusInternalServerError,
		[]byte(`{"error":"服务器内部错误","timestamp":"0000-00-00T00:00:00Z"}`),
	)

	// 自定义记录函数，更新错误信息中的时间戳
	builder.SetLogFunc(func(ctx *mist.Context, err any) {
		// 获取当前时间戳
		now := time.Now().UTC().Format(time.RFC3339)
		// 构建带有当前时间戳的JSON错误响应
		errorJSON := fmt.Sprintf(`{"error":"服务器内部错误","timestamp":"%s"}`, now)
		// 更新响应数据
		ctx.RespData = []byte(errorJSON)
		// 设置content-type
		ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		// 记录错误
		mist.Error("Recovery middleware caught panic: %v", err)
	})

	return builder.Build()
}
