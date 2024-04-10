package mist

import (
	"fmt"
	"net"
	"net/http"
)

// HandleFunc defines the function signature for an HTTP request handler specific to your web framework.
//
// This type represents a function that takes a pointer to a Context object as its argument and does
// not return any value. The Context object typically encapsulates all the information about the current
// HTTP request, including the request itself, response writer, path parameters, query parameters, and
// any other metadata or utilities needed to process the request.
//
// Usage:
// A HandleFunc is intended to be used as a callback for specific routes to handle incoming HTTP requests.
// Each route will have an associated HandleFunc that will be executed when the route is matched.
//
// Example:
//
//	func HelloWorldHandler(ctx *Context) {
//	  ctx.ResponseWriter.Write([]byte("Hello, World!"))
//	}
//
//	// Registering the handler with a route:
//	server.registerRoute("GET", "/hello", HelloWorldHandler)
type HandleFunc func(ctx *Context) // Type signature for request handling functions within the framework

// This line asserts that HTTPServer implements the Server interface at compile time.
// If HTTPServer does not implement all the methods defined in the Server interface,
// the program will not compile, and the compiler will provide an error message indicating
// which method(s) are missing. This is a safeguard to ensure that HTTPServer correctly
// fulfills the contract required by the Server interface, such as handling HTTP requests,
// starting the server, and registering routes with associated middleware and handlers.
var _ Server = &HTTPServer{} // Ensures HTTPServer implements the Server interface

// Server defines the interface for an HTTP server that can handle requests and be started on a
// specified address. It extends the http.Handler interface of the net/http package, which requires
// a ServeHTTP method to serve HTTP requests. In addition to handling HTTP requests, the server can
// register routes with associated handlers and middleware, and be started on a network address.
//
// Methods:
//
//   - Start(addr string) error: Starts the server listening on the specified network address (addr).
//     If the server fails to start, it returns an error.
//
//   - registerRoute(method string, path string, handleFunc, mils ...Middleware): Registers
//     a new route with a specific HTTP method and path. If provided, handleFunc becomes the main
//     function to handle matched requests; mils represents variadic middleware functions which will be
//     processed before the handleFunc upon route matching.
//
// Note:
//   - The registerRoute method is generally not exposed and is intended for internal use by implementations
//     of the Server interface. Implementations should ensure that routes are properly registered and
//     middleware is correctly applied within this method.
//
// Example:
// An implementation of the Server interface could manage its own routing table and middleware stack,
// allowing for modular and testable server designs. It would typically be used within an application
// like so:
//
//	func main() {
//	  srv := NewMyHTTPServer()  // MyHTTPServer implements the Server interface
//	  srv.registerRoute("GET", "/", HomePageHandler, LoggingMiddleware)
//	  err := srv.Start(":8080")
//	  if err != nil {
//	    log.Fatalf("Failed to start server: %v", err)
//	  }
//	}
type Server interface {
	http.Handler                                                                         // Inherited ServeHTTP method for handling requests
	Start(addr string) error                                                             // Method to start the server on a given address
	registerRoute(method string, path string, handleFunc HandleFunc, mils ...Middleware) // Internal route registration
}

// HTTPServerOption defines a function type used to apply configuration options to an HTTPServer.
//
// Each HTTPServerOption is a function that accepts a pointer to an HTTPServer and modifies it
// according to some configuration logic. This pattern, often called "functional options", allows
// for flexible, clear, and safe configurations when constructing an instance of HTTPServer.
// It enables the programmer to chain multiple configuration options in a declarative way when
// creating a new server instance or adjusting its settings.
//
// Usage:
// Developers can define custom HTTPServerOption functions that set various fields or initialize
// certain parts of the HTTPServer. These options can then be passed to a constructor function
// that applies them to the server instance.
//
// Example:
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
//	// When initializing a new HTTPServer:
//	srv := NewHTTPServer(
//	  WithTemplateEngine(myTemplateEngine),
//	  WithMiddleware(AuthMiddleware, LoggingMiddleware),
//	)
type HTTPServerOption func(server *HTTPServer) // Functional option for configuring an HTTPServer

// HTTPServer defines the structure of an HTTP server with routing capabilities, middleware support,
// logging functionality, and a template engine for rendering HTML templates.
//
// Fields:
//
//   - router: Embeds the routing information which includes the tree of routes and handlers that
//     the server uses to match incoming requests to their respective handlers.
//
//   - mils []Middleware: A slice that holds the globally applied middleware. These middleware
//     functions are executed for every request in the order they are added. They can be used to
//     modify the http.Handler behaviour, perform actions such as logging, authentication etc.
//
//   - log func(msg string, args ...any): A function for logging where msg is a formatted log message
//     and args are optional arguments which may be included in the formatted output. Developers can
//     use this to output relevant log information or to integrate third-party logging libraries.
//
//   - templateEngine: An interface that represents the template engine used by the server
//     to render HTML templates. This allows for dynamic HTML content generation based on data models
//     and can be customized by the developer to use the desired templating system.
//
// Usage:
// An instance of HTTPServer typically starts by configuring routes, middleware, and the template engine.
// Once set up, the server can handle HTTP requests, match them to their designated handlers, and generate
// dynamic responses. Logging is facilitated through the provided log function, enabling tracking of server
// activity and diagnosing issues.
type HTTPServer struct {
	router                                       // Embedded routing management
	mils           []Middleware                  // Middleware stack
	log            func(msg string, args ...any) // Logging function
	templateEngine TemplateEngine                // Template processor interface
}

// InitHTTPServer initializes a new instance of HTTPServer with the provided options.
// The options are applied to the server instance to configure various aspects such as
// middlewares, template engine, and other settings. The log function for the server is
// set to print messages to stdout using fmt.Printf by default. A default router is also
// initialized and set on the server.
//
// Usage example:
//
//	server := InitHTTPServer(
//	    ServerWithMiddleware(loggingMiddleware, authMiddleware),
//	    ServerWithTemplateEngine(myTemplateEngine),
//	)
//	// The server is now configured with logging, authentication middleware and a template engine.
//
// Parameters:
//   - opts ...HTTPServerOption : A variadic slice of configuration options, each being a function
//     that takes a pointer to an HTTPServer and applies a configuration to it.
//
// Returns:
// - *HTTPServer : A pointer to the newly created and configured HTTPServer instance.
//
// Note:
// The server is not started by this function and needs to be started separately using its
// Start or ListenAndServe method.
func InitHTTPServer(opts ...HTTPServerOption) *HTTPServer {
	res := &HTTPServer{
		router: initRouter(),
		log: func(msg string, args ...any) {
			fmt.Printf(msg, args...)
		},
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

// ServerWithTemplateEngine is a configuration function that returns an HTTPServerOption.
// This option is used to set a specific TemplateEngine to the HTTPServer, which can then
// be used to render HTML templates for the client. It's useful when your server needs to
// deliver dynamic web pages that are generated from templates.
//
// A TemplateEngine is an interface or a set of functionalities that processes templates
// with given data and produces an HTML output that the HTTP server can send to the client's
// web browser.
//
// Usage example:
//
//	server := NewHTTPServer(
//	    ServerWithTemplateEngine(myTemplateEngine),
//	)
//
// Parameters:
//   - templateEngine : The template engine to be set on the HTTPServer.
//     This parameter specifies the concrete implementation of a template engine
//     that the server will use for rendering templates.
//
// Returns:
//   - HTTPServerOption : A function that configures the server with the specified template engine.
//     When applied as an option to the server, it assigns the 'templateEngine' to the server's
//     internal field for later use.
func ServerWithTemplateEngine(templateEngine TemplateEngine) HTTPServerOption {
	return func(server *HTTPServer) {
		server.templateEngine = templateEngine
	}
}

// ServerWithMiddleware takes a variadic slice of Middleware functions and returns
// an HTTPServerOption. This option configures a HTTPServer with the provided
// middlewares. Middlewares are used to intercept or otherwise modify requests
// and responses in an HTTP server. Middleware functions are typically used for
// logging, security controls, rate limiting, etc.
//
// Example of using ServerWithMiddleware to configure an HTTPServer with middlewares:
//
//	myServer := NewHTTPServer(
//	    ServerWithMiddleware(loggingMiddleware, authenticationMiddleware),
//	)
//
// Parameters:
// - mils ...Middleware : A variadic slice of Middleware functions to be applied to the server.
//
// Returns:
//   - HTTPServerOption : A function that takes an HTTPServer pointer and assigns the provided
//     middlewares to it. This function can be applied as a configuration option when creating
//     an HTTPServer.
func ServerWithMiddleware(mils ...Middleware) HTTPServerOption {
	return func(server *HTTPServer) {
		server.mils = mils
	}
}

// Use attaches the provided middlewares to the existing set of middlewares in the HTTPServer instance.
// If no middleware has been set yet, it initializes the middleware list with the provided ones.
// If there are already middlewares present in the server, it appends the new ones to the end
// of the middleware chain.
//
// Middlewares are executed in the order they are added to the server, meaning that the order
// of middlewares can affect the request/response processing. They are commonly used to handle tasks
// such as request logging, authentication, input validation, error handling, etc.
//
// Usage example:
//
//	server := &HTTPServer{}
//	server.Use(loggingMiddleware)
//	server.Use(authenticationMiddleware)
//
// Parameters:
// - mils ...Middleware : One or multiple Middleware functions to add to the server's middleware chain.
//
// Note:
// This method appends provided middlewares variably, allowing for zero or more middlewares to be added
// at once. If called with no arguments, it will simply do nothing to the current middleware chain.
func (s *HTTPServer) Use(mils ...Middleware) {
	if s.mils == nil {
		s.mils = mils
		return
	}
	s.mils = append(s.mils, mils...)
}

// UseRoute associates a new route with the specified HTTP method and path to the server's routing system.
// Additionally, it allows for the chaining of middleware functions that can intercept and modify the
// request or response, or perform specific actions like logging, authentication, etc., before the
// request reaches the final handler function.
//
// Parameters:
//   - method string: The HTTP method (e.g., GET, POST, PUT, DELETE) for which the route is to be registered.
//   - path string: The path pattern to be matched against the URL of incoming requests.
//   - mils ...Middleware: A variadic parameter that allows passing an arbitrary number of middleware
//     functions. These functions are executed in the order they are provided, prior to the final handler.
//
// Usage:
// When registering a route, you can specify the HTTP method and path, followed by the series of middleware
// you wish to apply. If no final handler is provided at the time of route registration, one must be
// attached later for the route to be functional.
//
// Example usage:
//
//	s.UseRoute("GET", "/articles", AuthMiddleware, LogMiddleware)
//
// Here, `AuthMiddleware` would be used to authenticate the request, and `LogMiddleware` would log the
// request details. A route handler would need to be added subsequently to handle the GET requests for
// `/articles` path.
//
// Note:
// This method is used for initial route setup and must be combined with a handler registration to
// create a complete, functional route. If a handler is not attached later, the route will not have any effect.
func (s *HTTPServer) UseRoute(method string, path string, mils ...Middleware) {
	s.registerRoute(method, path, nil, mils...)
}

// ServeHTTP implements the http.Handler interface and is the entry point for HTTP requests
// coming to the server. This method wraps the incoming http.ResponseWriter and *http.Request
// into a new Context object. It also sets up the middleware chain by wrapping the server's
// main handler function with the middleware functions in reverse order.
//
// Once the middleware chain is set up, a special Middleware function is defined that ensures
// the final response is written using flashResp method after the main handler function completes.
// This middleware is then applied to the root of the chain.
//
// Finally, ServeHTTP calls the root of the middleware/handler chain with the newly created context
// to handle the request and generate a response. Once the handling is complete, the response is sent
// back to the client by flashResp being called from within the final Middleware function.
//
// Parameters:
// - writer http.ResponseWriter : The ResponseWriter that is used to write the HTTP response.
// - request *http.Request : The incoming HTTP request that needs to be handled.
//
// Note:
// ServeHTTP is automatically called by the net/http package when the server receives a new
// request, matching the signature required by the http.Handler interface.
func (s *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := &Context{
		Request:        request,
		ResponseWriter: writer,
		templateEngine: s.templateEngine,
	}
	root := s.server
	for i := len(s.mils) - 1; i >= 0; i-- {
		root = s.mils[i](root)
	}

	var m Middleware = func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)
			s.flashResp(ctx)
		}
	}

	root = m(root)

	root(ctx)
}

// flashResp sends the HTTP response data to the client using the Context's ResponseWriter.
// It first checks if a status code has been set on the context, and if so, writes that status
// code to the response header. Then it writes the response data to the client. If the writing
// operation encounters any errors or does not complete fully (i.e., all the response data
// is not written), it logs an error message using the server's log function.
//
// The server's log function should provide a way of logging messages, matching the signature
// of the log method defined on the HTTPServer. The default log implementation uses fmt.Printf
// to print log messages to standard output, but it could be overridden by a custom logging
// middleware if configured.
//
// Parameters:
//   - ctx *Context : The context associated with the current HTTP request. It contains the
//     ResponseWriter used to send headers and data to the client, as well as other response
//     information such as status code and response data.
//
// Note:
// This method does not handle the errors beyond logging. If logging or error handling
// needs to be more robust, it should be implemented as part of the server's configuration
// or within the middleware.
func (s *HTTPServer) flashResp(ctx *Context) {
	if ctx.RespStatusCode != 0 {
		ctx.ResponseWriter.WriteHeader(ctx.RespStatusCode)
	}
	write, err := ctx.ResponseWriter.Write(ctx.RespData)
	if err != nil || write != len(ctx.RespData) {
		s.log("写入响应数据失败 %v", err)
	}
}

// server is a core method that dispatches incoming requests to the appropriate handler functions
// based on the HTTP method and the URL path specified in the request. It is responsible for
// executing the logic to match the request against registered routes and invoking the corresponding
// handler.
//
// Parameters:
//   - ctx *Context: A pointer to the Context struct which holds information about the current
//     HTTP request and response state. This includes the request, response writer, status code,
//     response data, path parameters, and the matched route.
//
// Operation:
//  1. The method attempts to find a route matching the request's method and path.
//  2. If no matching route is found, or the handler function is nil, the server responds with
//     a 404 Not Found error.
//  3. If a match is found, the method updates the Context with the path parameters and matched route.
//  4. Lastly, it calls the registered handler function with the updated Context.
//
// Note:
//   - This method is typically not called directly by the user, but is an integral part of the server's
//     internal routing mechanism that automatically handles incoming requests.
func (s *HTTPServer) server(ctx *Context) {
	info, ok := s.findRoute(ctx.Request.Method, ctx.Request.URL.Path)
	if !ok || info.n == nil || info.n.handler == nil {
		ctx.RespStatusCode = http.StatusNotFound
		ctx.RespData = []byte("NOT FOUND")
		return
	}
	ctx.PathParams = info.pathParams
	ctx.MatchedRoute = info.n.route
	info.n.handler(ctx)
}

// Start begins running the HTTP server on the specified address. It sets up a TCP
// listener on the given address and then starts handling incoming HTTP requests using
// the server's ServeHTTP method.
//
// This method blocks while the server is running and only returns an error if there is
// an issue starting the listener or serving the requests.
//
// Parameters:
//   - addr string : The address where the server should listen for incoming HTTP requests,
//     in the form "host:port", where host is the IP address or hostname, and port is the
//     port number on which the server should listen.
//
// Returns:
//   - error : Non-nil error if there was an issue with starting the listener or handling
//     the requests; otherwise, nil.
//
// Example usage:
//
//	err := server.Start("localhost:8080")
//	if err != nil {
//	    log.Fatal("Failed to start server:", err)
//	}
//
// Note:
// The method uses the net.Listen function to set up the TCP listener and the http.Serve
// function to handle requests, which are part of Go's standard library.
func (s *HTTPServer) Start(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(l, s)
}

// Get registers a new route and its associated handler function for HTTP GET requests.
// This method is a shortcut for registering routes that should only respond to GET HTTP
// method, typically used for retrieving resources.
//
// Parameters:
//   - path string: The URL pattern to match against incoming requests. The route pattern
//     can contain parameters that will be parsed from the URL and made available to the
//     handler function during request handling.
//   - handleFunc: The function to be called when a request matching the path is
//     received. The handler function is defined to take a *Context as its only parameter,
//     through which it can access the request data and send a response back.
//
// Example usage:
//
//	s.Get("/home", func(ctx *Context) {
//	    // Handler logic for the `/home` path when an HTTP GET request is received
//	})
//
// Note:
// The method internally calls registerRoute to add the route to the server's routing
// table with the method specified as `http.MethodGet`, which ensures that only GET
// requests are handled by the provided handler.
func (s *HTTPServer) Get(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodGet, path, handleFunc)
}

// Head registers a new route and its associated handler function for HTTP HEAD requests.
// This method is used to handle requests where the client is interested only in the response headers,
// and not the actual body of the response, which is typical behavior of a HEAD request in HTTP.
//
// Parameters:
//   - path string: The path pattern to which the route will respond. When a HEAD request to this
//     pattern is received, the registered handler function will be executed.
//   - handleFunc: The handler function that will be associated with the provided path
//     pattern. This function will be called with a *Context parameter that contains information
//     about the request and mechanisms to construct a response.
//
// Example usage:
//
//	s.Head("/resource", func(ctx *Context) {
//	    // Handler logic to return response headers for the '/resource' path
//	    // without returning the actual body.
//	})
//
// Note:
// The method utilizes the registerRoute internal function to add the route to the server's
// routing table specifically for the HEAD HTTP method, which ensures that only HEAD
// requests will trigger the execution of the provided handler function.
func (s *HTTPServer) Head(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodHead, path, handleFunc)
}

// Post registers a new route and its associated handler function for handling HTTP POST requests.
// This method is used for routes that should accept data sent to the server, usually for the purpose of
// creating or updating resources.
//
// Parameters:
//   - path string: The URL pattern to match against incoming POST requests. It defines the endpoint at
//     which the handler function will be called for incoming POST requests.
//   - handleFunc: The function to be executed when a POST request is made to the specified path.
//     It receives a *Context object that contains the request information and provides the means to write
//     a response back to the client.
//
// Example usage:
//
//	s.Post("/submit", func(ctx *Context) {
//	    // Handler logic for processing the POST request to the `/submit` path.
//	})
//
// Note:
// The method delegates to registerRoute, internally setting the HTTP method to `http.MethodPost`. This
// ensures that the registered handler is invoked only for POST requests matching the specified path.
func (s *HTTPServer) Post(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPost, path, handleFunc)
}

// Put registers a new route and its associated handler function for handling HTTP PUT requests.
// This method is typically used to update an existing resource or create a new resource at a specific URL.
//
// Parameters:
//   - path string: The URL pattern to which the server should listen for PUT requests. This pattern may include
//     placeholders for dynamic segments of the URL, which can be used to pass variables to the handler function.
//   - handleFunc: A callback function that will be invoked when a PUT request is made to the
//     specified path. The function takes a *Context parameter that provides access to the request data and
//     response writer.
//
// Example usage:
//
//	s.Put("/items/{id}", func(ctx *Context) {
//	    // Handler logic for updating an item with a particular ID using a PUT request.
//	})
//
// Note:
// By calling registerRoute and specifying `http.MethodPut`, this method ensures that the handler is
// specifically associated with PUT requests. If a PUT request is made on the matched path, the
// corresponding handler function will be executed.
func (s *HTTPServer) Put(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPut, path, handleFunc)
}

// Patch registers a new route with an associated handler function for HTTP PATCH requests.
// This method is generally used for making partial updates to an existing resource.
//
// Parameters:
//   - path string: The pattern of the URL that the server will match against incoming PATCH requests.
//     The path can include variables that will be extracted from the URL and passed to the handler.
//   - handleFunc: The function to execute when the server receives a PATCH request at the
//     specified path. This function is provided with a *Context object, enabling access to request
//     information and response functionalities.
//
// Example usage:
//
//	s.Patch("/profile/{id}", func(ctx *Context) {
//	    // Handler logic to apply partial updates to a profile based on the ID in the URL.
//	})
//
// Note:
// Registering the route with the `http.MethodPatch` constant ensures that only PATCH requests are
// handled by the provided function. The PATCH method is typically used to apply a partial update to
// a resource, and this function is where you would define how the server handles such requests.
func (s *HTTPServer) Patch(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPatch, path, handleFunc)
}

// Delete registers a new route with an associated handler function for HTTP DELETE requests.
// This method is used to remove a resource identified by a URI.
//
// Parameters:
//   - path string: The URL pattern that the server will listen on for incoming DELETE requests.
//     This parameter defines the endpoint at which the handler will be called when a DELETE
//     request matches the path.
//   - handleFunc: A function that is called when a DELETE request is made to the
//     registered path. This function should contain the logic to handle the deletion of a
//     resource, and it is provided with a *Context object to interact with the request and
//     response data.
//
// Example usage:
//
//	s.Delete("/users/{id}", func(ctx *Context) {
//	    // Handler logic to delete a user resource with the given ID.
//	})
//
// Note:
// Using `http.MethodDelete` in the call to registerRoute confines this handler to respond
// solely to DELETE requests, providing a way to define how the server handles deletions.
func (s *HTTPServer) Delete(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodDelete, path, handleFunc)
}

// Connect registers a new route with an associated handler function for handling HTTP CONNECT
// requests. The HTTP CONNECT method is utilized primarily for establishing a tunnel to a server
// identified by a given URI.
//
// Parameters:
//   - path string: The endpoint or route pattern where the server will listen for incoming
//     CONNECT requests. This may include parameter placeholders that can be used to extract
//     values from the URL during request handling.
//   - handleFunc: A callback function that is invoked in response to a CONNECT
//     request to the given path. This function has access to the request and response through
//     a *Context, providing the necessary tools to implement the tunneling behavior or other
//     custom logic expected on a CONNECT request.
//
// Example usage:
//
//	s.Connect("/proxy", func(ctx *Context) {
//	    // Logic to establish a proxy connection.
//	})
//
// Note:
// The use of `http.MethodConnect` ensures that only HTTP CONNECT requests are matched to
// this handler, facilitating the appropriate processing logic for these specialized request
// types, which are different from the standard GET, POST, PUT, etc., methods.
func (s *HTTPServer) Connect(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodConnect, path, handleFunc)
}

// Options registers a new route with an associated handler function for HTTP OPTIONS requests.
// The HTTP OPTIONS method is used to describe the communication options for the target resource.
//
// Parameters:
//   - path string: The URL pattern that the server will match against incoming OPTIONS requests.
//     Defining the endpoint allows clients to find out which methods and operations are supported
//     at a given URL or server.
//   - handleFunc: The function to be executed when an OPTIONS request is received.
//     It typically provides information about the HTTP methods that are available for a
//     particular URL endpoint. The handleFunc is supplied with a *Context object to facilitate
//     interaction with the HTTP request and response.
//
// Example usage:
//
//	s.Options("/articles/{id}", func(ctx *Context) {
//	    // Handler logic to indicate supported methods like GET, POST, PUT on the article resource.
//	})
//
// Note:
// This registration only affects OPTIONS requests due to the use of `http.MethodOptions`. It is
// standard practice to implement this method on a server to inform clients about the methods and
// content types that the server is capable of handling, thereby aiding the client's decision-making
// regarding further actions.
func (s *HTTPServer) Options(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodOptions, path, handleFunc)
}

// Trace registers a new route with an associated handler function for HTTP TRACE requests.
// The HTTP TRACE method is used to echo back the received request so that a client can see what
// (if any) changes or additions have been made by intermediate servers.
//
// Parameters:
//   - path string: The endpoint on the server that will respond to the TRACE requests. This defines
//     the path pattern that must be matched for the handler function to be invoked.
//   - handleFunc: A function that handles the TRACE request. It should process the request
//     and typically returns the same request message in the response body. This function has a
//     *Context object allowing access to the request details and the ability to write the response.
//
// Example usage:
//
//	s.Trace("/echo", func(ctx *Context) {
//	    // Handler logic that echoes the incoming request back to the client.
//	})
//
// Note:
// Registering this route specifically listens for HTTP TRACE requests by using `http.MethodTrace`.
// This is helpful for debugging purposes where the client needs to understand what headers and
// body are being received by the server after any possible alterations by intermediate devices.
func (s *HTTPServer) Trace(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodTrace, path, handleFunc)
}
