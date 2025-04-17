package mist

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// 此行断言HTTPServer实现了Server接口（在编译时进行检查）。
// 如果HTTPServer没有实现Server接口中定义的所有方法，
// 程序将无法编译，编译器会提供错误消息指出缺少哪些方法。
// 这是一个安全措施，确保HTTPServer正确履行Server接口所要求的职责，
// 例如处理HTTP请求、启动服务器以及注册带有关联中间件和处理程序的路由。
var _ Server = &HTTPServer{} // 确保HTTPServer实现了Server接口

// Server定义了一个可以处理请求并在指定地址上启动的HTTP服务器接口。
// 它扩展了net/http包的http.Handler接口，该接口要求实现ServeHTTP方法来处理HTTP请求。
// 除了处理HTTP请求外，服务器还可以注册带有关联处理程序和中间件的路由，并在网络地址上启动。
//
// 方法:
//
//   - Start(addr string) error: 在指定的网络地址(addr)上启动服务器。
//     如果服务器启动失败，它会返回一个错误。
//
//   - registerRoute(method string, path string, handleFunc, mils ...Middleware): 注册
//     一个带有特定HTTP方法和路径的新路由。如果提供了handleFunc，它将成为处理匹配请求的
//     主要函数；mils代表在路由匹配时将在handleFunc之前处理的可变中间件函数。
//
// 注意:
//   - registerRoute方法通常不对外暴露，它主要供Server接口的实现内部使用。
//     实现应确保正确注册路由并在此方法中正确应用中间件。
//
// 示例:
// Server接口的实现可以管理自己的路由表和中间件栈，
// 允许模块化和可测试的服务器设计。通常在应用程序中的使用方式如下:
//
//	func main() {
//	  srv := NewMyHTTPServer()  // MyHTTPServer实现了Server接口
//	  srv.registerRoute("GET", "/", HomePageHandler, LoggingMiddleware)
//	  err := srv.Start(":8080")
//	  if err != nil {
//	    log.Fatalf("启动服务器失败: %v", err)
//	  }
//	}
type Server interface {
	http.Handler                                                                         // 继承的ServeHTTP方法用于处理请求
	Start(addr string) error                                                             // 方法用于在给定地址上启动服务器
	registerRoute(method string, path string, handleFunc HandleFunc, mils ...Middleware) // 内部路由注册
}

// HTTPServerOption定义了一个用于应用配置选项到HTTPServer的函数类型。
//
// 每个HTTPServerOption是一个接受指向HTTPServer的指针并根据某些配置逻辑修改它的函数。
// 这种模式，通常称为"函数选项模式"，允许在构建HTTPServer实例时以灵活、清晰且安全的方式进行配置。
// 它使程序员能够在创建新服务器实例或调整其设置时以声明式方式链接多个配置选项。
//
// 用法:
// 开发者可以定义自定义HTTPServerOption函数，这些函数设置HTTPServer的各个字段或初始化其某些部分。
// 然后可以将这些选项传递给一个构造函数，该函数将它们应用于服务器实例。
//
// 示例:
//
//	func WithTemplateEngine(engine TemplateEngine) HTTPServerOption {
//	  return func(server *HTTPServer) {
//	    server.templateEngine = engine
//	  }
//	}
//
//	func WithMiddleware(middleware ...Middleware) HTTPServerOption {
//	  return func(server *HTTPServer) {
//	    server.mils = append(server.mils, middleware...)
//	  }
//	}
//
//	// 初始化新的HTTPServer:
//	srv := NewHTTPServer(
//	  WithTemplateEngine(myTemplateEngine),
//	  WithMiddleware(AuthMiddleware, LoggingMiddleware),
//	)
type HTTPServerOption func(server *HTTPServer) // 用于配置HTTPServer的函数选项

// HTTPServer是一个结构体，定义了Web应用程序中HTTP服务器的基本结构。
// 它封装了处理HTTP请求所需的组件，如路由、中间件处理、日志记录和模板渲染。
// 通过将这些功能组织到单个结构体中，它为开发者提供了一个内聚的框架，以高效地管理
// 服务器的行为并配置其各种组件。
// 嵌入字段和属性:
//
//	router: router是一个嵌入字段，代表服务器的路由机制。作为嵌入字段，
//	        它为HTTPServer提供了直接访问路由方法的能力。router负责
//	        根据URL路径和HTTP方法将传入请求映射到适当的处理函数。
//	middlewares ([]Middleware): middlewares切片保存服务器将按顺序为每个请求执行的中间件函数。
//	                           中间件函数用于拦截和操作请求和响应，允许身份验证、日志记录
//	                           和会话管理等任务以模块化方式处理。
//	log (Logger): log字段是Logger接口的一个实例。这种抽象允许服务器利用各种日志
//	             实现，以标准化的方式灵活记录服务器事件、错误和其他信息。
//	templateEngine (TemplateEngine): templateEngine字段是一个接口，它抽象了HTML模板
//	                                 处理和渲染的具体细节。它允许服务器执行模板并提供
//	                                 动态内容，使得根据应用程序需求轻松集成不同的模板处理
//	                                 系统成为可能。
//
// 用法:
// 构建HTTPServer时，开发者必须在启动服务器前初始化每个组件:
//   - router必须设置路由，将URL映射到处理函数。
//   - 必须按照必要的顺序将中间件函数添加到middlewares切片中，因为它们将按顺序执行。
//   - 必须为log字段提供Logger实现，以记录服务器操作、错误和其他事件。
//   - 如果服务器将提供动态HTML内容，必须分配一个符合templateEngine接口的TemplateEngine，
//     使服务器能够用动态数据渲染HTML模板。
//
// 通过确保所有这些组件都被正确初始化，HTTPServer可以高效地管理入站请求，
// 应用必要的预处理，处理路由，执行业务逻辑，并生成动态响应。
type HTTPServer struct {
	router                        // 嵌入式路由管理。提供对路由方法的直接访问。
	log            Logger         // 日志接口。允许灵活和一致的日志记录。
	templateEngine TemplateEngine // 模板处理器接口。便于HTML模板渲染。
	middlewares    []Middleware   // 全局中间件
	httpServer     *http.Server   // 内置的HTTP服务器，用于配置和优雅关闭
}

// ServerConfig 定义HTTP服务器的配置选项
type ServerConfig struct {
	ReadTimeout       time.Duration // 读取整个请求的超时时间
	WriteTimeout      time.Duration // 写入响应的超时时间
	IdleTimeout       time.Duration // 连接空闲超时时间
	ReadHeaderTimeout time.Duration // 读取请求头的超时时间
	MaxHeaderBytes    int           // 请求头的最大字节数
	RequestTimeout    time.Duration // 单个请求处理的最大时间

	// HTTP/2和HTTP/3支持
	EnableHTTP2            bool          // 是否启用HTTP/2
	EnableHTTP3            bool          // 是否启用HTTP/3
	HTTP3IdleTimeout       time.Duration // HTTP/3连接空闲超时时间
	QuicMaxIncomingStreams int           // QUIC最大并发流数量
}

// DefaultServerConfig 返回默认的服务器配置
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,          // 1MB
		RequestTimeout:    30 * time.Second, // 默认单个请求处理超时时间

		// HTTP/2和HTTP/3默认配置
		EnableHTTP2:            true,
		EnableHTTP3:            false, // 默认不启用HTTP/3，需要显式开启
		HTTP3IdleTimeout:       30 * time.Second,
		QuicMaxIncomingStreams: 100,
	}
}

// WithServerConfig 设置HTTP服务器配置
func WithServerConfig(config ServerConfig) HTTPServerOption {
	return func(s *HTTPServer) {
		if s.httpServer == nil {
			s.httpServer = &http.Server{}
		}
		s.httpServer.ReadTimeout = config.ReadTimeout
		s.httpServer.WriteTimeout = config.WriteTimeout
		s.httpServer.IdleTimeout = config.IdleTimeout
		s.httpServer.ReadHeaderTimeout = config.ReadHeaderTimeout
		s.httpServer.MaxHeaderBytes = config.MaxHeaderBytes
	}
}

// InitHTTPServer 初始化并返回一个新的HTTPServer实例指针。可以通过传入各种HTTPServerOption函数
// 来自定义服务器，这些函数将根据这些选项封装的功能修改服务器的配置。这种模式被称为"函数选项模式"，
// 它允许灵活且可读的服务器配置，而不需要一个可能很长的参数列表。
//
// 参数:
//   - opts: 一个可变的HTTPServerOption函数数组。每个函数都被应用于HTTPServer实例，并可以
//     设置或修改诸如中间件、日志记录、服务器地址、超时等配置。
//
// InitHTTPServer函数按以下步骤操作:
//
//  1. 使用一些初始默认设置创建一个新的HTTPServer实例。
//     a. 为服务器初始化一个router来管理传入请求的路由。
//     b. 设置一个默认的日志记录函数，将消息打印到标准输出，可通过选项覆盖。
//  2. 遍历每个提供的HTTPServerOption，将其应用于服务器实例。这些选项是接受*HTTPServer参数
//     并修改其属性的函数，从而根据应用程序的特定需求定制服务器。
//  3. 应用所有选项后，函数返回定制的HTTPServer实例，准备启动并开始处理传入的HTTP请求。
//
// 这个初始化函数抽象了服务器设置的复杂性，允许开发者仅指定与其应用程序相关的选项，
// 从而使服务器初始化代码更加清晰和可维护。
func InitHTTPServer(opts ...HTTPServerOption) *HTTPServer {
	// 使用默认配置创建一个新的HTTPServer。
	res := &HTTPServer{
		router: initRouter(), // 初始化HTTPServer的路由器，用于请求处理。
	}

	// 对HTTPServer应用每个提供的HTTPServerOption，根据用户需求配置它。
	for _, opt := range opts {
		opt(res) // 每个'opt'是一个修改'res' HTTPServer实例的函数。
	}

	// 返回现在可能已配置的HTTPServer实例。
	return res
}

// ServerWithTemplateEngine 是一个配置函数，返回一个HTTPServerOption。
// 这个选项用于为HTTPServer设置特定的模板引擎，然后可以用来为客户端渲染HTML模板。
// 当你的服务器需要提供从模板生成的动态网页时，这个功能很有用。
//
// 模板引擎是一个接口或一组功能，它处理模板及给定数据，并生成HTTP服务器可以发送到
// 客户端网页浏览器的HTML输出。
//
// 使用示例:
//
//	server := NewHTTPServer(
//	    ServerWithTemplateEngine(myTemplateEngine),
//	)
//
// 参数:
//   - templateEngine : 要设置到HTTPServer上的模板引擎。
//     此参数指定服务器将用于渲染模板的模板引擎的具体实现。
//
// 返回:
//   - HTTPServerOption : 一个用指定的模板引擎配置服务器的函数。
//     当作为服务器的选项应用时，它将'templateEngine'分配给服务器的
//     内部字段，以供以后使用。
func ServerWithTemplateEngine(templateEngine TemplateEngine) HTTPServerOption {
	return func(server *HTTPServer) {
		server.templateEngine = templateEngine
	}
}

// Use 注册全局中间件
func (s *HTTPServer) Use(mdls ...Middleware) {
	if len(mdls) == 0 {
		return
	}
	s.middlewares = append(s.middlewares, mdls...)
}

// UseRoute 将一个新的路由与指定的HTTP方法和路径关联到服务器的路由系统中。
// 此外，它允许链接可以在请求到达最终处理函数之前拦截并修改请求或响应，
// 或执行特定操作如日志记录、身份验证等的中间件函数。
//
// 参数:
//   - method string: 要为其注册路由的HTTP方法（例如，GET，POST，PUT，DELETE）。
//   - path string: 与传入请求的URL匹配的路径模式。
//   - mils ...Middleware: 一个可变参数，允许传递任意数量的中间件函数。
//     这些函数按照提供的顺序执行，在最终处理程序之前。
//
// 用法:
// 注册路由时，可以指定HTTP方法和路径，然后是你希望应用的一系列中间件。
// 如果在路由注册时没有提供最终处理程序，则以后必须附加一个，路由才能正常工作。
//
// 使用示例:
//
//	s.UseRoute("GET", "/articles", AuthMiddleware, LogMiddleware)
//
// 这里，`AuthMiddleware`将用于认证请求，`LogMiddleware`将记录请求详情。
// 随后需要添加一个路由处理程序来处理`/articles`路径的GET请求。
//
// 注意:
// 此方法用于初始路由设置，必须与处理程序注册组合以创建完整、功能性的路由。
// 如果稍后没有附加处理程序，路由将不会有任何效果。
func (s *HTTPServer) UseRoute(method string, path string, mils ...Middleware) {
	s.registerRoute(method, path, nil, mils...)
}

// UseForAll 为指定路径注册所有HTTP方法的中间件
func (s *HTTPServer) UseForAll(path string, mdls ...Middleware) {
	// 为指定路径的HTTP GET方法注册中间件。
	s.registerRoute(http.MethodGet, path, nil, mdls...)
	// 为指定路径的HTTP POST方法注册中间件。
	s.registerRoute(http.MethodPost, path, nil, mdls...)
	// 为指定路径的HTTP OPTIONS方法注册中间件。
	s.registerRoute(http.MethodOptions, path, nil, mdls...)
	// 为指定路径的HTTP CONNECT方法注册中间件。
	s.registerRoute(http.MethodConnect, path, nil, mdls...)
	// 为指定路径的HTTP DELETE方法注册中间件。
	s.registerRoute(http.MethodDelete, path, nil, mdls...)
	// 为指定路径的HTTP HEAD方法注册中间件。
	s.registerRoute(http.MethodHead, path, nil, mdls...)
	// 为指定路径的HTTP PATCH方法注册中间件。
	s.registerRoute(http.MethodPatch, path, nil, mdls...)
	// 为指定路径的HTTP PUT方法注册中间件。
	s.registerRoute(http.MethodPut, path, nil, mdls...)
	// 为指定路径的HTTP TRACE方法注册中间件。
	s.registerRoute(http.MethodTrace, path, nil, mdls...)
}

// ServeHTTP 是处理HTTP请求的主要入口点。
func (s *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// 从对象池获取Context
	var requestTimeout time.Duration
	if s.httpServer != nil && s.httpServer.ReadTimeout > 0 {
		requestTimeout = s.httpServer.ReadTimeout
	} else {
		requestTimeout = 30 * time.Second
	}

	ctx := GetContext(writer, request, requestTimeout)

	// 处理完成后释放Context
	defer ReleaseContext(ctx)

	// 请求处理
	s.server(ctx)
}

// flashResp 将上下文中的响应数据发送回客户端
func (s *HTTPServer) flashResp(ctx *Context) {
	// 已经写入响应头但没有主体内容的情况
	if ctx.headerWritten && ctx.RespData == nil {
		return
	}

	// 如果状态码已经被设置为0，使用默认的200 OK
	if ctx.RespStatusCode == 0 {
		ctx.RespStatusCode = http.StatusOK
	}

	// 写入响应头
	ctx.writeHeader(ctx.RespStatusCode)

	// 如果没有响应数据，直接返回
	if ctx.RespData == nil || len(ctx.RespData) == 0 {
		return
	}

	// 使用缓冲区池优化大响应
	if len(ctx.RespData) > 4096 {
		buffer := GetBuffer()
		defer ReleaseBuffer(buffer)

		// 将响应数据分块写入
		const chunkSize = 4096
		data := ctx.RespData
		for len(data) > 0 {
			// 计算当前块大小
			size := chunkSize
			if len(data) < size {
				size = len(data)
			}

			// 重用buffer
			buffer = append(buffer[:0], data[:size]...)
			_, err := ctx.ResponseWriter.Write(buffer)
			if err != nil {
				s.log.Error("写入响应失败: %v", err)
				return
			}

			// 移动到下一块
			data = data[size:]
		}
	} else {
		// 小响应直接写入
		_, err := ctx.ResponseWriter.Write(ctx.RespData)
		if err != nil {
			s.log.Error("写入响应失败: %v", err)
		}
	}
}

// server 是一个方法，通过解析适当的路由和执行关联的处理程序以及任何适用的中间件来处理传入的HTTP请求。
func (s *HTTPServer) server(ctx *Context) {
	// 查找匹配请求方法和路径的路由。
	mi, ok := s.findRoute(ctx.Request.Method, ctx.Request.URL.Path)

	// 如果没有匹配到路由，直接返回404
	if !ok || mi.n == nil || mi.n.handler == nil {
		ctx.RespStatusCode = 404
		s.flashResp(ctx)
		return
	}

	// 如果找到匹配的节点，填充上下文与特定路由相关的路径参数和匹配的路由。
	if mi.n != nil {
		ctx.PathParams = mi.pathParams
		ctx.MatchedRoute = mi.n.route
	}

	// 定义一个根处理函数，它将尝试执行匹配路由的处理程序。
	var root HandleFunc = func(ctx *Context) {
		// 路由已经匹配，直接调用处理函数
		mi.n.handler(ctx)
	}

	// 收集所有中间件，修改顺序为：路由中间件 -> 全局中间件
	var mdls []Middleware

	// 先添加路由特定的中间件（外层）
	if len(mi.mils) > 0 {
		mdls = append(mdls, mi.mils...)
	}

	// 再添加全局中间件（内层）
	if len(s.middlewares) > 0 {
		mdls = append(mdls, s.middlewares...)
	}

	// 反向应用中间件，确保路由顺序是：
	// 路由中间件开始 -> 全局中间件开始 -> 处理函数 -> 全局中间件结束 -> 路由中间件结束
	for i := len(mdls) - 1; i >= 0; i-- {
		root = mdls[i](root)
	}

	// 定义一个中间件，确保在处理程序（和任何其他中间件）完成处理后正确发送响应。
	var m Middleware = func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			if ctx.Aborted {
				// 如果请求已被中止，立即刷新响应并且不调用任何进一步的中间件或处理程序。
				s.flashResp(ctx)
				return
			}

			next(ctx) // 调用下一个中间件或最终处理程序。

			if ctx.Aborted {
				// 执行下一个中间件或最终处理程序后，再次检查请求是否已被中止。如果是，立即刷新响应。
				s.flashResp(ctx)
				return
			}

			s.flashResp(ctx)
		}
	}

	// 使用刷新中间件包装根处理程序。
	root = m(root)

	// 调用根函数，它代表以路由的处理程序结束的中间件链。
	root(ctx)
}

// Start 启动HTTP服务器监听指定地址。它在给定地址上设置TCP网络监听器，
// 然后启动HTTP服务器以使用此监听器接受和处理传入请求。如果创建网络监听器
// 或启动服务器有问题，它会返回一个错误。
//
// 参数:
//   - addr: 一个字符串，指定服务器要监听的TCP地址。这通常包括主机名或IP，
//     后跟冒号和端口号（例如，"localhost:8080"或":80"）。如果只指定带前导冒号的
//     端口号，服务器将在给定端口上的所有可用IP地址上监听。
//
// Start函数按以下方式运行:
//
//  1. 以"tcp"作为网络类型和提供的地址调用net.Listen。这尝试创建一个可以
//     在指定地址上接受传入TCP连接的监听器。
//  2. 如果net.Listen返回错误，它会立即返回给调用者，表明无法创建监听器
//     （可能是由于无效地址、无法绑定到端口等）。
//  3. 如果监听器成功创建，该函数然后以监听器和服务器本身作为参数调用http.Serve。
//     这启动HTTP服务器，它开始监听和处理请求。服务器将使用HTTPServer的ServeHTTP方法
//     处理每个请求。
//  4. 如果http.Serve遇到错误，它也将返回给调用者。当服务器运行时遇到意外问题时
//     可能会发生这种情况，比如无法接受连接。
//
// Start方法是一个阻塞调用。一旦调用，它将继续运行，服务传入的HTTP请求，
// 直到遇到错误或服务器被手动停止。
func (s *HTTPServer) Start(addr string) error {
	// 如果server未初始化，使用默认配置
	if s.httpServer == nil {
		s.httpServer = &http.Server{
			Handler:           s,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
		}
	} else {
		s.httpServer.Handler = s
	}

	s.httpServer.Addr = addr

	// 创建监听器
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.httpServer.Serve(l)
}

// StartTLS 启动HTTPS服务器
func (s *HTTPServer) StartTLS(addr, certFile, keyFile string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// 如果HTTPServer未初始化
	if s.httpServer == nil {
		s.httpServer = &http.Server{}
	}

	// 确保设置处理器
	s.httpServer.Handler = s

	// 配置TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"}, // 支持HTTP/2和HTTP/1.1
	}

	// 应用服务器配置
	s.httpServer.TLSConfig = tlsConfig
	s.httpServer.Addr = addr

	// 启动HTTPS服务器
	return s.httpServer.ServeTLS(listener, certFile, keyFile)
}

// StartHTTP3 启动支持HTTP/3的服务器
func (s *HTTPServer) StartHTTP3(addr, certFile, keyFile string) error {
	// 确保初始化HTTP服务器
	if s.httpServer == nil {
		s.httpServer = &http.Server{
			Handler: s,
		}
	}

	// 创建配置
	config := DefaultHTTP3Config()

	// 创建HTTP/3服务器
	h3Server := NewHTTP3Server(s.httpServer, config, s.log)

	// 同时启动HTTP/2作为回退选项
	go func() {
		if err := s.StartTLS(addr, certFile, keyFile); err != nil {
			s.log.Error("启动TLS服务器失败: %v", err)
		}
	}()

	// 启动HTTP/3服务器
	return h3Server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Shutdown 优雅关闭服务器，等待现有请求完成
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// GET 注册一个带有关联处理函数的新路由，用于HTTP GET请求。
// 此方法主要用于从服务器检索数据。
//
// 参数:
//   - path string: 用于匹配传入GET请求的URL模式。当到达匹配此模式的GET请求时，
//     关联的处理函数将被执行。
//   - handleFunc: 当接收到匹配路径的请求时要调用的函数。处理函数被定义为
//     只接受一个*Context参数，通过它可以访问请求数据并发送响应。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Get("/home", func(ctx *Context) {
//	    // 接收到HTTP GET请求时处理`/home`路径的处理程序逻辑
//	})
//
// 注意:
// 该方法内部调用registerRoute来将路由添加到服务器的路由表中，
// 将方法指定为`http.MethodGet`，这确保只有GET请求会被提供的处理程序处理。
func (s *HTTPServer) GET(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodGet, path, handleFunc, ms...)
}

// HEAD 注册一个新的路由及其关联的处理函数，用于HTTP HEAD请求。
// 此方法用于处理客户端只对响应头感兴趣而不需要实际响应体的请求，
// 这是HTTP HEAD请求的典型行为。
//
// 参数:
//   - path string: 路由将响应的路径模式。当收到对此模式的HEAD请求时，
//     注册的处理函数将被执行。
//   - handleFunc: 与提供的路径模式关联的处理函数。此函数将使用包含请求信息
//     和构建响应机制的*Context参数调用。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Head("/resource", func(ctx *Context) {
//	    // 为'/resource'路径返回响应头的处理逻辑，
//	    // 而不返回实际的响应体。
//	})
//
// 注意:
// 该方法使用registerRoute内部函数将路由添加到服务器的路由表中，
// 专门针对HEAD HTTP方法，这确保只有HEAD请求才会触发提供的处理函数的执行。
func (s *HTTPServer) HEAD(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodHead, path, handleFunc, ms...)
}

// POST 注册一个新的路由及其关联的处理函数，用于处理HTTP POST请求。
// 此方法用于应接受发送到服务器的数据的路由，通常是用于创建或更新资源。
//
// 参数:
//   - path string: 用于匹配传入POST请求的URL模式。它定义了处理函数将在
//     传入POST请求时被调用的端点。
//   - handleFunc: 当对指定路径发出POST请求时要执行的函数。它接收一个
//     *Context对象，该对象包含请求信息并提供向客户端写回响应的方法。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Post("/submit", func(ctx *Context) {
//	    // 处理对`/submit`路径的POST请求的逻辑。
//	})
//
// 注意:
// 该方法委托给registerRoute，内部将HTTP方法设置为`http.MethodPost`。
// 这确保只有匹配指定路径的POST请求才会调用注册的处理程序。
func (s *HTTPServer) POST(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodPost, path, handleFunc, ms...)
}

// PUT 注册一个新的路由及其关联的处理函数，用于处理HTTP PUT请求。
// 此方法通常用于更新现有资源或在特定URL创建新资源。
//
// 参数:
//   - path string: 服务器应监听PUT请求的URL模式。此模式可能包括URL的动态段的占位符，
//     这些占位符可用于向处理函数传递变量。
//   - handleFunc: 当对指定路径发出PUT请求时将调用的回调函数。该函数接受
//     一个*Context参数，该参数提供对请求数据和响应编写器的访问。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Put("/items/{id}", func(ctx *Context) {
//	    // 使用PUT请求更新具有特定ID的项目的处理逻辑。
//	})
//
// 注意:
// 通过调用registerRoute并指定`http.MethodPut`，此方法确保处理程序
// 专门与PUT请求相关联。如果对匹配的路径发出PUT请求，将执行相应的处理函数。
func (s *HTTPServer) PUT(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodPut, path, handleFunc, ms...)
}

// PATCH 注册一个新的路由及其关联的处理函数，用于HTTP PATCH请求。
// 此方法通常用于对现有资源进行部分更新。
//
// 参数:
//   - path string: 服务器将匹配传入PATCH请求的URL模式。
//     路径可以包含将从URL提取并传递给处理程序的变量。
//   - handleFunc: 当服务器在指定路径接收到PATCH请求时执行的函数。
//     此函数提供了*Context对象，使其能够访问请求信息和响应功能。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Patch("/profile/{id}", func(ctx *Context) {
//	    // 基于URL中的ID对配置文件应用部分更新的处理逻辑。
//	})
//
// 注意:
// 使用`http.MethodPatch`常量注册路由确保只有PATCH请求由提供的函数处理。
// PATCH方法通常用于对资源应用部分更新，而此函数是您定义服务器如何处理此类请求的地方。
func (s *HTTPServer) PATCH(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodPatch, path, handleFunc, ms...)
}

// DELETE 注册一个新的路由及其关联的处理函数，用于HTTP DELETE请求。
// 此方法用于删除由URI标识的资源。
//
// 参数:
//   - path string: 服务器将监听传入DELETE请求的URL模式。
//     此参数定义了当DELETE请求匹配路径时将调用处理程序的端点。
//   - handleFunc: 当对注册的路径发出DELETE请求时调用的函数。此函数应包含
//     处理资源删除的逻辑，并提供*Context对象与请求和响应数据交互。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Delete("/users/{id}", func(ctx *Context) {
//	    // 删除给定ID的用户资源的处理逻辑。
//	})
//
// 注意:
// 在调用registerRoute时使用`http.MethodDelete`将此处理程序限制为仅响应
// DELETE请求，提供了一种定义服务器如何处理删除的方式。
func (s *HTTPServer) DELETE(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodDelete, path, handleFunc, ms...)
}

// CONNECT 注册一个新的路由及其关联的处理函数，用于处理HTTP CONNECT请求。
// HTTP CONNECT方法主要用于建立到由给定URI标识的服务器的隧道。
//
// 参数:
//   - path string: 服务器将监听传入CONNECT请求的端点或路由模式。这可能包括
//     参数占位符，可用于在请求处理期间从URL提取值。
//   - handleFunc: 响应对给定路径的CONNECT请求而调用的回调函数。此函数通过
//     *Context访问请求和响应，提供实现隧道行为或CONNECT请求上预期的
//     其他自定义逻辑所需的工具。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Connect("/proxy", func(ctx *Context) {
//	    // 建立代理连接的逻辑。
//	})
//
// 注意:
// 使用`http.MethodConnect`确保只有HTTP CONNECT请求匹配到此处理程序，
// 便于为这些专门的请求类型提供适当的处理逻辑，这些请求类型与标准的
// GET、POST、PUT等方法不同。
func (s *HTTPServer) CONNECT(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodConnect, path, handleFunc, ms...)
}

// OPTIONS 注册一个新的路由及其关联的处理函数，用于HTTP OPTIONS请求。
// HTTP OPTIONS方法用于描述目标资源的通信选项。
//
// 参数:
//   - path string: 服务器将匹配传入OPTIONS请求的URL模式。
//     定义端点允许客户端找出在给定URL或服务器上支持哪些方法和操作。
//   - handleFunc: 收到OPTIONS请求时要执行的函数。它通常提供关于
//     特定URL端点可用的HTTP方法的信息。handleFunc提供*Context对象，
//     以促进与HTTP请求和响应的交互。
//   - ms: 一个可变的Middleware函数切片，在路由匹配时，这些函数将按传入顺序
//     在处理函数之前执行。
//
// 使用示例:
//
//	s.Options("/articles/{id}", func(ctx *Context) {
//	    // 处理逻辑，指示文章资源上支持的方法如GET、POST、PUT等。
//	})
//
// 注意:
// 由于使用了`http.MethodOptions`，此注册只影响OPTIONS请求。服务器上实现此方法是标准做法，
// 以便向客户端通知服务器能够处理的方法和内容类型，从而帮助客户端决定进一步的操作。
func (s *HTTPServer) OPTIONS(path string, handleFunc HandleFunc, ms ...Middleware) {
	s.registerRoute(http.MethodOptions, path, handleFunc, ms...)
}

// registerRoute是HTTPServer结构体上的一个方法，用于在服务器上注册路由。
// 这个方法被各种HTTP方法特定的函数（如GET、POST等）调用，并且
// 在内部用于设置路由及其各自的处理程序和中间件。
//
// 参数:
//   - method: 路由应响应的HTTP方法（例如，GET、POST、PUT等）。
//   - path: 与传入请求匹配的URL模式。它可以包含用冒号':'标记的参数
//     （例如，'/users/:id'）或通配符'*'。
//   - handleFunc: 当路由匹配时要调用的函数。这个函数处理HTTP请求并生成响应。
//     如果您只是附加中间件，它可以为nil。
//   - mils: 一个可变参数，包含要应用于路由的中间件函数。这些函数
//     按提供的顺序执行，在主处理函数之前。
//
// 这个函数在内部:
//   - 验证路径以'/'开始，并且不包含不必要的尾部斜杠。
//   - 将路由及其处理程序添加到路由器的内部路由树中。
//   - 将提供的中间件与路由关联。
//
// 注意:
// 如果您想为某个路径下的所有路由添加中间件，请考虑使用Group
// 功能或Use方法。
func (s *HTTPServer) registerRoute(method string, path string, handleFunc HandleFunc, mils ...Middleware) {
	s.router.registerRoute(method, path, handleFunc, mils...)
}

// EnableRouteStats 启用路由统计
func (s *HTTPServer) EnableRouteStats() {
	s.router.EnableStats()
}

// DisableRouteStats 禁用路由统计
func (s *HTTPServer) DisableRouteStats() {
	s.router.DisableStats()
}

// GetRouteStats 获取指定路由的统计信息
func (s *HTTPServer) GetRouteStats(route string) (map[string]interface{}, bool) {
	return s.router.GetRouteStats(route)
}

// GetAllRouteStats 获取所有路由的统计信息
func (s *HTTPServer) GetAllRouteStats() map[string]map[string]interface{} {
	return s.router.GetAllRouteStats()
}

// ResetRouteStats 重置路由统计信息
func (s *HTTPServer) ResetRouteStats() {
	s.router.ResetRouteStats()
}
