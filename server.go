package mist

import (
	"net"
	"net/http"
	"strconv"
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

// HTTPServer is a struct that defines the basic structure of an HTTP server
// within a web application. It encapsulates the components necessary for handling
// HTTP requests, such as routing, middleware processing, logging, and template
// rendering. By organizing these functionalities into a single struct, it provides
// a cohesive framework for developers to manage the server's behavior and configure
// its various components efficiently.

// Embedded and Fields:
//   router: The router is an embedded field representing the server's routing
//           mechanism. As an embedded field, it provides the HTTPServer direct
//           access to the routing methods. The router is responsible for
//           mapping incoming requests to the appropriate handler functions
//           based on URL paths and HTTP methods.

//   mils ([]Middleware): The mils slice holds the middleware functions that
//                        the server will execute sequentially for each request.
//                        Middleware functions are used to intercept and manipulate
//                        requests and responses, allowing for tasks such as
//                        authentication, logging, and session management to be
//                        handled in a modular fashion.

//   log (Logger): The log field is an instance of the Logger interface. This
//                 abstraction allows the server to utilize various logging
//                 implementations, providing the flexibility to log server events,
//                 errors, and other informational messages in a standardized manner.

//   templateEngine (TemplateEngine): The templateEngine field is an interface
//                                    that abstracts away the specifics of how
//                                    HTML templates are processed and rendered.
//                                    It allows the server to execute templates
//                                    and serve dynamic content, making it easy
//                                    to integrate different template processing
//                                    systems according to the application's needs.

// Usage:
// When constructing an HTTPServer, developers must initialize each component
// before starting the server:

// - The router must be set up with routes that map URLs to handler functions.
// - Middleware functions must be added to the mils slice in the necessary order
//   as they will be executed sequentially on each request.
// - A Logger implementation must be provided to the log field to record server
//   operations, errors, and other events.
// - If the server will serve dynamic HTML content, a TemplateEngine that
//   complies with the templateEngine interface must be assigned, enabling the
//   server to render HTML templates with dynamic data.

// By ensuring all these components are properly initialized, the HTTPServer
// can efficiently manage inbound requests, apply necessary pre-processing,
// handle routing, execute business logic, and generate dynamic responses.

type HTTPServer struct {
	router                        // Embedded routing management. Provides direct access to routing methods.
	mils           []Middleware   // Middleware stack. A slice of Middleware functions to process requests.
	log            Logger         // Logger interface. Allows for flexible and consistent logging.
	templateEngine TemplateEngine // Template processor interface. Facilitates HTML template rendering.
}

// InitHTTPServer initializes and returns a pointer to a new HTTPServer instance. The server can be customized by
// passing in various HTTPServerOption functions, which will modify the server's configuration according to the
// functionalities encapsulated by those options. This pattern is known as the "functional options" pattern and allows
// for flexible and readable server configuration without the need for a potentially long list of parameters.
//
// Parameters:
//   - opts: A variadic array of HTTPServerOption functions. Each one is applied to the HTTPServer instance and can
//     set or modify configurations such as middlewares, logging, server address, timeouts, etc.
//
// The InitHTTPServer function operates in the following steps:
//
//  1. Creates a new HTTPServer instance with some initial default settings.
//     a. A router is initialized for the server to manage routing of incoming requests.
//     b. A default logging function is set up to print messages to standard output, which can be overridden by an option.
//  2. Iterates through each provided HTTPServerOption, applying it to the server instance. These options are functions
//     that accept a *HTTPServer argument and modify its properties, thereby customizing the server according to the
//     specific needs of the application.
//  3. After applying all options, the function returns the customized HTTPServer instance, ready to be started and to
//     begin handling incoming HTTP requests.
//
// This initialization function abstracts away the complexity of server setup and allows developers to specify only the
// options relevant to their application, leading to cleaner and more maintainable server initialization code.
func InitHTTPServer(opts ...HTTPServerOption) *HTTPServer {
	// Create a new HTTPServer with a default configuration.
	res := &HTTPServer{
		router: initRouter(), // Initialize the HTTPServer's router for request handling.
	}

	// Apply each provided HTTPServerOption to the HTTPServer to configure it according to the user's requirements.
	for _, opt := range opts {
		opt(res) // Each 'opt' is a function that modifies the 'res' HTTPServer instance.
	}

	// Return the now potentially configured HTTPServer instance.
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

// UseForAll associates the provided middlewares with all HTTP methods for the specified path.
// This method configures the HTTP server to apply the given middlewares to every request method
// (GET, POST, OPTIONS, etc.) for the specified route path. This is useful when you want consistent
// middleware execution for a path, irrespective of the HTTP method used in the request.
//
// Parameters:
// - path: The URL pattern to match against incoming request URLs.
// - mdls: A variadic list of middlewares to be applied to all requests for the route.
//
// Usage:
//   - server.UseForAll("/api/resource", loggerMiddleware, authMiddleware)
//     This example would apply the logger and auth middlewares to all request
//     methods for path "/api/resource".
func (s *HTTPServer) UseForAll(path string, mdls ...Middleware) {
	// Register the middlewares for the HTTP GET method for the specified path.
	s.registerRoute(http.MethodGet, path, nil, mdls...)
	// Register the middlewares for the HTTP POST method for the specified path.
	s.registerRoute(http.MethodPost, path, nil, mdls...)
	// Register the middlewares for the HTTP OPTIONS method for the specified path.
	s.registerRoute(http.MethodOptions, path, nil, mdls...)
	// Register the middlewares for the HTTP CONNECT method for the specified path.
	s.registerRoute(http.MethodConnect, path, nil, mdls...)
	// Register the middlewares for the HTTP DELETE method for the specified path.
	s.registerRoute(http.MethodDelete, path, nil, mdls...)
	// Register the middlewares for the HTTP HEAD method for the specified path.
	s.registerRoute(http.MethodHead, path, nil, mdls...)
	// Register the middlewares for the HTTP PATCH method for the specified path.
	s.registerRoute(http.MethodPatch, path, nil, mdls...)
	// Register the middlewares for the HTTP PUT method for the specified path.
	s.registerRoute(http.MethodPut, path, nil, mdls...)
	// Register the middlewares for the HTTP TRACE method for the specified path.
	s.registerRoute(http.MethodTrace, path, nil, mdls...)
}

// ServeHTTP is the core method for handling incoming HTTP requests in the HTTPServer. This method fulfills the
// http.Handler interface, making an HTTPServer instance compatible with Go's built-in HTTP server machinery.
// ServeHTTP is responsible for creating the context for the request, applying middleware, and calling the final
// request handler. After the request is processed, it ensures that any buffered response (if applicable) is flushed
// to the client.
//
// Parameters:
// - writer: An http.ResponseWriter that is used to write the HTTP response to be sent to the client.
// - request: An *http.Request that represents the client's HTTP request being handled.
//
// The ServeHTTP function operates in the following manner:
//
//  1. It begins by creating a new Context instance, which is a custom type holding the HTTP request and response
//     writer, along with other request-specific information like the templating engine.
//  2. It retrieves the root handler from the server's configuration 's.server', which represents the starting point
//     for the request handling pipeline.
//  3. Iteratively wraps the root handler with the server's configured middleware in reverse order. Middleware is
//     essentially a chain of functions that can execute before and/or after the main request handler to perform
//     tasks such as logging, authentication, etc.
//  4. Introduces a final middleware that calls the next handler in the chain and then flushes any buffered response
//     using 's.flashResp'. This ensures that even if a response is buffered (for performance reasons or to allow
//     for manipulations), it gets sent out after the request is processed.
//  5. Calls the fully wrapped root handler, beginning the execution of the middleware chain and ultimately invoking
//     the appropriate request handler.
func (s *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Create the context that will traverse the request handling chain.
	ctx := &Context{
		Request:        request,          // The original HTTP request.
		ResponseWriter: writer,           // The ResponseWriter to work with the HTTP response.
		templateEngine: s.templateEngine, // The templating engine, if any, to render HTML views.
	}
	root := s.server // Start with the base server handler.
	// Loop through all the server's middleware in reverse to construct the middleware chain.
	for i := len(s.mils) - 1; i >= 0; i-- {
		root = s.mils[i](root) // Wrap the current 'root' with the middleware function.
	}
	// Define a final Middleware to ensure that the response is flushed after handling the request.
	var m Middleware = func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			next(ctx)        // Execute the next handler in the chain.
			s.flashResp(ctx) // Flush the response after handling the request.
		}
	}
	root = m(root) // Wrap the final Middleware around the root handler.
	root(ctx)      // Execute the request handling starting from the root of the middleware chain.
}

// flashResp is a method on the HTTPServer struct that commits the HTTP response
// to the client. It is responsible for finalizing the response status code, setting
// the appropriate headers, and writing the response data to the client. If any
// errors occur during the response writing process, it will log a fatal error using
// the server's configured default logger.
//
// Parameters:
//
//	ctx *Context: A pointer to the Context struct that contains information about
//	              the current request, response, and additional data relevant to the
//	              HTTP transaction. The Context struct holds the response writer,
//	              the status code, and the response data to be sent back to the client.
//
// Usage:
// This method is typically called after an HTTP request has been processed by
// the server's handler functions and any associated middleware. It ensures that
// the HTTP response is correctly formed and transmitted to the client, concluding
// the request-handling cycle.
func (s *HTTPServer) flashResp(ctx *Context) {
	// If a status code has been set on the Context, write it as the HTTP response status code.
	if !ctx.headerWritten && ctx.RespStatusCode > 0 {
		ctx.writeHeader(ctx.RespStatusCode)
	}

	// Calculate the length of the response data and set the "Content-Length" header accordingly.
	// The Content-Length header is important as it tells the client how many bytes of data to expect.
	ctx.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(ctx.RespData)))

	// Write the response data to the HTTP client. The Write method of ResponseWriter
	// is used to send the response payload contained within ctx.RespData.
	_, err := ctx.ResponseWriter.Write(ctx.RespData)
	if err != nil {
		// In the event of a failure to write the response data to the client,
		// log a fatal error with the defaultLogger. A fatal log typically indicates an
		// error so severe that it is impossible to continue the operation of the program.
		defaultLogger.Fatalln("Failed to write response data:", err)
	}
}

// server is a method of HTTPServer that processes an incoming HTTP request by searching for a matching route
// and invoking the associated handler. If no matching route is found, or the found node doesn't have a handler,
// it responds with a 404 Not Found status. This method marks the entry point for a request's handling logic
// after passing through any configured middleware.
//
// Parameters:
//   - ctx: A pointer to a Context that contains the HTTP request contextual details such as the request itself,
//     ResponseWriter for writing back the response, and other metadata regarding the request.
//
// The server function performs the following actions:
//
//  1. Searches for a route matching the current request's method and URL path in the server's routing structure.
//  2. If a match is found (ok is true), and the found node (info.n) has a non-nil handler function, then the Context
//     is enriched with the path parameters extracted from the URL and the matched route info. The associated handler
//     function is then called with the updated context.
//  3. If there is no match for the route, or the node doesn't have a handler, the server responds with a 404 Not Found
//     status by setting the appropriate status code and response data in the context, which will later be sent back to
//     the client.
//
// This method ensures that a request is routed to the correct handler based on the defined routing rules.
// If no appropriate handler is registered for the request, it ensures that the server responds with a proper
// error message and status code.
func (s *HTTPServer) server(ctx *Context) {
	// Attempt to find a matching route in the server's routing structure based on the HTTP method and URL path.
	info, ok := s.findRoute(ctx.Request.Method, ctx.Request.URL.Path)

	// Check if a matching route is found and has a non-nil handler.
	if !ok || info.n == nil || info.n.handler == nil {
		// Prepare a Not Found (404) response if no matching route is available.
		ctx.RespStatusCode = http.StatusNotFound // Set the status code to 404.
		ctx.RespData = []byte("NOT FOUND")       // Set the response body to "NOT FOUND".
		return                                   // Return immediately since no further action is needed.
	}

	// Populate the context with the path parameters and the matched route info from the findRoute method.
	ctx.PathParams = info.pathParams // Store any dynamic parameters from the URL path.
	ctx.MatchedRoute = info.n.route  // Store the exact matched route pattern.

	// Call the handler associated with the found route, passing in the updated context.
	info.n.handler(ctx)
}

// Start initiates the HTTP server listening on the specified address. It sets up a TCP network listener on the
// given address and then starts the HTTP server to accept and handle incoming requests using this listener. If
// there is a problem creating the network listener or starting the server, it returns an error.
//
// Parameters:
//   - addr: A string specifying the TCP address for the server to listen on. This typically includes a hostname or
//     IP followed by a colon and the port number (e.g., "localhost:8080" or ":80"). If only the port number
//     is specified with a leading colon, the server will listen on all available IP addresses on the given port.
//
// The Start function operates in the following manner:
//
//  1. Calls net.Listen with "tcp" as the network type and the provided address. This attempts to create a listener
//     that can accept incoming TCP connections on the specified address.
//  2. If net.Listen returns an error, it is immediately returned to the caller, indicating that the listener could
//     not be created (possibly due to an invalid address, inability to bind to the port, etc.).
//  3. If the listener is successfully created, the function then calls http.Serve with the listener and the server
//     itself as arguments. This starts the HTTP server, which begins listening for and handling requests. The server
//     will use the ServeHTTP method of the HTTPServer to process each request.
//  4. If http.Serve encounters an error, it will also be returned to the caller. This can happen if there's an
//     unexpected issue while the server is running, such as a failure to accept a connection.
//
// The Start method is a blocking call. Once called, it will continue to run, serving incoming HTTP requests until
// an error is encountered or the server is manually stopped.
func (s *HTTPServer) Start(addr string) error {
	// Create a new TCP listener on the specified address.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err // Return the error if the listener could not be created.
	}

	// Start the HTTP server with the newly created listener, using 's' (HTTPServer) as the handler.
	return http.Serve(l, s) // Return the result of http.Serve, which will block until the server stops.
}

// GET registers a new route and its associated handler function for HTTP GET requests.
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
func (s *HTTPServer) GET(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodGet, path, handleFunc)
}

// HEAD registers a new route and its associated handler function for HTTP HEAD requests.
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
func (s *HTTPServer) HEAD(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodHead, path, handleFunc)
}

// POST registers a new route and its associated handler function for handling HTTP POST requests.
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
func (s *HTTPServer) POST(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPost, path, handleFunc)
}

// PUT registers a new route and its associated handler function for handling HTTP PUT requests.
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
func (s *HTTPServer) PUT(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPut, path, handleFunc)
}

// PATCH registers a new route with an associated handler function for HTTP PATCH requests.
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
func (s *HTTPServer) PATCH(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodPatch, path, handleFunc)
}

// DELETE registers a new route with an associated handler function for HTTP DELETE requests.
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
func (s *HTTPServer) DELETE(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodDelete, path, handleFunc)
}

// CONNECT registers a new route with an associated handler function for handling HTTP CONNECT
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
func (s *HTTPServer) CONNECT(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodConnect, path, handleFunc)
}

// OPTIONS registers a new route with an associated handler function for HTTP OPTIONS requests.
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
func (s *HTTPServer) OPTIONS(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodOptions, path, handleFunc)
}

// TRACE registers a new route with an associated handler function for HTTP TRACE requests.
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
func (s *HTTPServer) TRACE(path string, handleFunc HandleFunc) {
	s.registerRoute(http.MethodTrace, path, handleFunc)
}
