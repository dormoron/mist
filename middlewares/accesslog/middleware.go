package accesslog

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dormoron/mist"
)

// MiddlewareBuilder is a struct that facilitates the creation of middleware functions with
// logging capabilities for a web service framework (hypothetical 'mist' framework in this context).
// It contains a field that holds a reference to a logging function which can be used to log messages.
// This struct could potentially be extended with additional fields and methods to further
// customize the behavior of the middleware being built, especially for handling logging in a
// consistent manner throughout the application.
//
// Fields:
//
//	logFunc: A function of type 'func(log string)' that represents the logging strategy
//	         the middleware will utilize. The 'logFunc' is intended to be called within middleware
//	         handlers to log various messages, like errors or informational messages, to a log output,
//	         such as the console, a file, or a remote log management system.
//
// Example of adding a log function to MiddlewareBuilder:
//
//	mb := &MiddlewareBuilder{
//	    logFunc: func(log string) {
//	        // Implementation of the log function, could log to stdout or a file, etc.
//	        fmt.Println("Log:", log)
//	    },
//	}
//
//	// With the logFunc set, MiddlewareBuilder can now be used to construct middleware that logs messages.
//	middleware := mb.Build() // Assuming Build method is implemented on MiddlewareBuilder.
//	// The middleware can be used within the web framework to apply logging per request.
type MiddlewareBuilder struct {
	// logFunc is a field that holds a user-defined function intended to handle logging of messages.
	// The function takes a single string argument which contains the log message to be recorded.
	// The behavior of logging—where and how the log messages are output—is determined by the implementation
	// of this function provided by the user.
	logFunc func(log string)
	// 是否使用彩色输出
	colorful bool
	// 是否使用格式化输出而非JSON
	prettyFormat bool
	// 最大路径显示长度
	maxPathLength int
	// 是否显示IP地址
	showIP bool
	// 是否显示图标
	showIcons bool
}

// LogFunc assigns a custom logging function to the MiddlewareBuilder instance. This method is used
// to define how messages should be logged when the middleware operates within the request handling flow.
// It accepts a function parameter that matches the signature needed for logging operations in the middleware.
// The method enables a fluent interface by returning the MiddlewareBuilder instance itself, allowing for
// method chaining when configuring the builder.
//
// Parameters:
//
//	fn: A function that takes a single string argument representing the log message. The function should
//	    contain the code that dictates how the log messages will be processed and where they will be sent.
//	    This could involve logging to the console, writing to a file, or sending the data to a log aggregation
//	    service depending on the actual implementation passed to this method.
//
// Returns:
//
//	*MiddlewareBuilder: A pointer to the current instance of the MiddlewareBuilder, allowing for additional
//	                     configuration calls to be chained.
//
// Example usage:
//
//	mb := new(MiddlewareBuilder).
//	        LogFunc(func(log string) {
//	            // Custom logging implementation, such as sending log to an external service.
//	            SendLogToService(log)
//	        })
//	// At this point, 'mb' has a logging function that sends logs to an external service.
func (b *MiddlewareBuilder) LogFunc(fn func(log string)) *MiddlewareBuilder {
	// Assigns the given function 'fn' as the logging function for MiddlewareBuilder instance 'b'.
	// This function will be used for logging within the middleware built using this builder.
	b.logFunc = fn

	// Returns the MiddlewareBuilder instance to allow for method chaining.
	return b
}

// Colorful 设置是否使用彩色输出
// Parameters:
//
//	enabled: 是否启用彩色输出
//
// Returns:
//
//	*MiddlewareBuilder: 当前构建器实例，支持链式调用
func (b *MiddlewareBuilder) Colorful(enabled bool) *MiddlewareBuilder {
	b.colorful = enabled
	return b
}

// PrettyFormat 设置是否使用格式化输出而非JSON
// Parameters:
//
//	enabled: 是否启用格式化输出
//
// Returns:
//
//	*MiddlewareBuilder: 当前构建器实例，支持链式调用
func (b *MiddlewareBuilder) PrettyFormat(enabled bool) *MiddlewareBuilder {
	b.prettyFormat = enabled
	return b
}

// SetMaxPathLength 设置路径显示的最大长度
// Parameters:
//
//	length: 路径最大显示长度
//
// Returns:
//
//	*MiddlewareBuilder: 当前构建器实例，支持链式调用
func (b *MiddlewareBuilder) SetMaxPathLength(length int) *MiddlewareBuilder {
	b.maxPathLength = length
	return b
}

// ShowIP 设置是否显示IP地址
// Parameters:
//
//	show: 是否显示IP地址
//
// Returns:
//
//	*MiddlewareBuilder: 当前构建器实例，支持链式调用
func (b *MiddlewareBuilder) ShowIP(show bool) *MiddlewareBuilder {
	b.showIP = show
	return b
}

// ShowIcons 设置是否显示图标
// Parameters:
//
//	show: 是否显示图标
//
// Returns:
//
//	*MiddlewareBuilder: 当前构建器实例，支持链式调用
func (b *MiddlewareBuilder) ShowIcons(show bool) *MiddlewareBuilder {
	b.showIcons = show
	return b
}

// InitMiddleware initializes a new instance of the MiddlewareBuilder struct with default
// configuration settings. It sets up a standard logging function that will log access
// events using the Go standard library's log package. The returned MiddlewareBuilder
// can be further configured with additional options before being used to create
// middleware for an HTTP server.
//
// Returns:
// - A pointer to a new MiddlewareBuilder with default log function configuration.
//
// Usage:
//   - builder := InitBuilder()
//     This will create a new MiddlewareBuilder with a default log function.
func InitMiddleware() *MiddlewareBuilder {
	// Create a new MiddlewareBuilder instance with default configurations.
	return &MiddlewareBuilder{
		// Set the logFunc field to a default function that uses the log package's Println method.
		// This function prints the provided accessLog string to the standard logger, which by default
		// outputs to os.Stderr. The log output includes a timestamp and the file name and line number
		// of the log call, a behavior determined by the log package's standard flags.
		logFunc: func(accessLog string) {
			log.Println(accessLog)
		},
		colorful:      false,
		prettyFormat:  false,
		maxPathLength: 50,
		showIP:        false,
		showIcons:     false,
	}
}

// getMethodIcon 根据HTTP方法返回相应的图标
func getMethodIcon(method string) string {
	switch method {
	case "GET":
		return "🔍" // 放大镜
	case "POST":
		return "➕" // 加号
	case "PUT":
		return "📝" // 笔记
	case "DELETE":
		return "🗑️" // 垃圾桶
	case "PATCH":
		return "🔧" // 扳手
	case "HEAD":
		return "👁️" // 眼睛
	case "OPTIONS":
		return "⚙️" // 齿轮
	default:
		return "🔗" // 链接
	}
}

// getStatusIcon 根据状态码返回相应的图标
func getStatusIcon(status int) string {
	if status >= 200 && status < 300 {
		return "✅" // 成功
	} else if status >= 300 && status < 400 {
		return "➡️" // 重定向
	} else if status >= 400 && status < 500 {
		return "⚠️" // 客户端错误
	} else if status >= 500 {
		return "❌" // 服务器错误
	}
	return "❓" // 未知
}

// truncatePath 截断过长的路径，添加省略号
func truncatePath(path string, maxLength int) string {
	if len(path) <= maxLength {
		return path
	}
	return path[:maxLength-3] + "..."
}

// Build constructs a middleware function that is compliant with the mist framework's Middleware type.
// The middleware created by this method encompasses a logging feature as configured via the MiddlewareBuilder.
// The middleware function created here, when executed, performs the following operations:
//   - It first creates a deferred function that collects access log data upon request completion and logs it
//     using the pre-configured `logFunc`.
//   - Then it continues with the execution of the next middleware or final request handler (`next`) in the stack.
//
// The method returns a closure that matches the mist.Middleware type signature and can be integrated into
// the middleware chain during the configuration of the HTTP server or router.
//
// Returns:
//   - mist.Middleware: A middleware function that wraps around the next handler in the middleware stack,
//     providing access logging functionalities.
//
// Example usage:
//   - After creating an instance of MiddlewareBuilder and setting the log function,
//     the Build method is used to obtain a new Middleware function that logs requests.
//     router.Use(mb.Build()) // Assuming 'router' is an instance that has a 'Use' method accepting Middleware.
func (b *MiddlewareBuilder) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		// The function returned here matches the mist.HandleFunc signature. It receives the next handler
		// in the middleware chain and also returns a mist.HandleFunc. This allows it to be used within
		// the 'mist' framework as a middleware.
		return func(ctx *mist.Context) {
			// 记录请求开始时间
			startTime := time.Now()

			// Define a deferred function that will always run after the request processing is completed.
			// This deferred function creates an access log struct containing relevant request information,
			// marshals it to JSON, and then logs it using the `logFunc` defined in the MiddlewareBuilder.
			defer func() {
				// 计算请求处理时间
				duration := time.Since(startTime)

				// 获取客户端IP地址
				clientIP := ctx.ClientIP()

				// Compile access log information into a struct from the provided context `ctx`.
				log := accessLog{
					Host:       ctx.Request.Host,        // Hostname from the HTTP request
					StatusCode: ctx.RespStatusCode,      // status code the HTTP request
					Route:      ctx.MatchedRoute,        // The route pattern matched for the request
					Method:     ctx.Request.Method,      // HTTP method, e.g., GET, POST
					Path:       ctx.Request.URL.Path,    // Request path
					Duration:   duration.Milliseconds(), // 请求处理时间（毫秒）
					ClientIP:   clientIP,                // 客户端IP地址
				}

				var logMessage string
				if b.prettyFormat {
					// 使用格式化输出
					var statusColor, methodColor, resetColor, timeColor, ipColor, routeColor string

					if b.colorful {
						// 颜色代码
						resetColor = "\033[0m"
						timeColor = "\033[90m"  // 灰色
						ipColor = "\033[94m"    // 淡蓝色
						routeColor = "\033[95m" // 紫色

						// 根据状态码选择颜色
						if log.StatusCode >= 200 && log.StatusCode < 300 {
							statusColor = "\033[32m" // 绿色
						} else if log.StatusCode >= 300 && log.StatusCode < 400 {
							statusColor = "\033[33m" // 黄色
						} else if log.StatusCode >= 400 && log.StatusCode < 500 {
							statusColor = "\033[31m" // 红色
						} else {
							statusColor = "\033[35;1m" // 加粗紫色用于500错误
						}

						// 根据HTTP方法选择颜色
						switch log.Method {
						case "GET":
							methodColor = "\033[34m" // 蓝色
						case "POST":
							methodColor = "\033[32m" // 绿色
						case "PUT":
							methodColor = "\033[33m" // 黄色
						case "DELETE":
							methodColor = "\033[31m" // 红色
						case "PATCH":
							methodColor = "\033[35m" // 紫色
						case "HEAD":
							methodColor = "\033[36m" // 青色
						default:
							methodColor = "\033[37m" // 白色
						}
					}

					// 获取当前时间用于日志
					timeStr := time.Now().Format("15:04:05.000")

					// 创建状态码标记 [200]
					var methodIcon, statusIcon string
					if b.showIcons {
						methodIcon = getMethodIcon(log.Method) + " "
						statusIcon = getStatusIcon(log.StatusCode) + " "
					}

					statusStr := fmt.Sprintf("[%d]", log.StatusCode)

					// 截断过长的路径
					truncatedPath := truncatePath(log.Path, b.maxPathLength)

					// 格式化路由，使其更美观
					route := log.Route
					if route != "" {
						route = "→ " + route
					}

					// 创建响应时间标记
					var durationColor string
					if b.colorful {
						if log.Duration < 100 {
							durationColor = "\033[32m" // 绿色（快）
						} else if log.Duration < 500 {
							durationColor = "\033[33m" // 黄色（中）
						} else {
							durationColor = "\033[31m" // 红色（慢）
						}
					}
					durationStr := fmt.Sprintf("+%dms", log.Duration)

					// 构建日志部分
					parts := []string{
						fmt.Sprintf("%s%s%s", timeColor, timeStr, resetColor),
						fmt.Sprintf("%s%s%s%s%s", methodColor, methodIcon, log.Method, resetColor, strings.Repeat(" ", 7-len(log.Method))),
						fmt.Sprintf("%s%s%s%s", statusColor, statusIcon, statusStr, resetColor),
					}

					// 如果显示IP地址
					if b.showIP && log.ClientIP != "" {
						parts = append(parts, fmt.Sprintf("%sfrom %s%s", ipColor, log.ClientIP, resetColor))
					}

					// 添加路径
					parts = append(parts, fmt.Sprintf("%s%s%s", methodColor, truncatedPath, resetColor))

					// 添加路由
					if route != "" {
						parts = append(parts, fmt.Sprintf("%s%s%s", routeColor, route, resetColor))
					}

					// 添加持续时间
					parts = append(parts, fmt.Sprintf("%s%s%s", durationColor, durationStr, resetColor))

					// 连接所有部分
					logMessage = strings.Join(parts, " ")
				} else {
					// 使用JSON格式
					data, _ := json.Marshal(log)
					logMessage = string(data)
				}

				// Log the access log JSON string via the logging function provided to the builder.
				// This employs the strategy we previously set with MiddlewareBuilder.LogFunc.
				b.logFunc(logMessage)
			}()

			// Call the next handler in the middleware chain with the current context.
			next(ctx)
		}
	}
}

// accessLog defines the structure for logging HTTP request details. It is used within middleware to capture
// and marshal HTTP request information into a JSON format that can be logged by the configured log function.
// Each field in the struct is tagged with json struct tags to specify the JSON key names and omit the fields if
// they are empty when marshalling the struct into JSON format.
//
// Fields:
//
//	Host:       The domain name or IP address of the server that received the request.
//	            It is included in the HTTP request header and is represented here as a string.
//	            The `json:"host,omitempty"` struct tag indicates that this field will be marshalled into JSON
//	            with the key "host" and will be omitted from the marshalled JSON object if the field is empty.
//
//	Route:      The matched route pattern for the request. It provides context about which route was matched
//	            for a given request path, aiding in debugging and analytics.
//	            Similar to Host, it will only appear in JSON if it is not empty.
//
//	HTTPMethod: The HTTP method used for the request, e.g., GET, POST, PUT, DELETE, etc.
//	            This helps to understand the action that the client intended to perform.
//	            It will appear in the JSON with the key "http_method" if not empty.
//
//	Path:       The path of the request URL. This represents the specific endpoint or resource requested by the client.
//	            Documented in the JSON with the key "path" and is omitted if it is empty.
//
// An instance of accessLog is created and populated with data from an HTTP request context and then marshalled into JSON.
// The JSON output is then passed to a logging function to record the incoming requests being handled by an HTTP server.
type accessLog struct {
	Host       string `json:"host,omitempty"`     // The server host name or IP address from the HTTP request.
	Route      string `json:"route,omitempty"`    // The matched route pattern for the request.
	Method     string `json:"method,omitempty"`   // The method used in the request (e.g., GET, POST).
	Path       string `json:"path,omitempty"`     // The path of the HTTP request URL.
	StatusCode int    `json:"status,omitempty"`   // The statusCode of the HTTP request status.
	Duration   int64  `json:"duration,omitempty"` // 请求处理时间（毫秒）
	ClientIP   string `json:"ip,omitempty"`       // The client IP address from the HTTP request.
}
