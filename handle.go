package mist

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
