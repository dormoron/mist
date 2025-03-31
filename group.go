package mist

import (
	"net/http"

	"github.com/dormoron/mist/internal/errs"
)

// routerGroup represents a group of routes that share a common path prefix and optionally middleware.
// It allows for organizing routes into subdomains or subsections, making the routing
// structure more modular and easier to maintain. Routes within a group will inherit the
// group's prefix, allowing for concise route definitions.
//
// Fields:
//   - prefix: The common path prefix for all routes within this group. All the routes
//     defined under this group will have this prefix prepended to their individual paths.
//   - parent: Pointer to the parent routerGroup, allowing for nested groups (subgroups)
//     within a larger routing structure. A nil value indicates that there is no
//     parent group (i.e., this is a top-level group).
//   - router: Pointer to the router that this group is a part of. This connection back to the
//     main router allows the group to register new routes under the router's
//     internal structures and middleware stack.
//   - middles: Slice of Middleware functions that are applied to all routes within this group.
//     These middleware functions are executed in the order that they are added to
//     this slice, prior to the route-specific handler being called. They can be used
//     for logging, auth, session management, etc.
type routerGroup struct {
	prefix  string
	parent  *routerGroup
	router  *router
	middles []Middleware
}

// registerRoute adds a new route to the routerGroup with the specified HTTP method, path, and handler.
// The route is added with the group's prefix and any middleware specified for the group and the route.
// Middleware for the group are applied before the route-specific middleware.
//
// Parameters:
//   - method: The HTTP method (e.g., GET, POST) for which the handler is being registered.
//   - path: The endpoint path for the route without the group prefix. The group's prefix is automatically
//     prepended to this path to form the full route path.
//   - handler: The HandleFunc to be invoked when the route is accessed with the specified method. It represents
//     the core logic that should be executed when the route is matched.
//   - ms: Optional variadic Middleware functions that will be applied to this route in addition to any
//     middleware associated with the routerGroup. These are executed after group-specific middleware
//     but before the handler function.
//
// Usage:
// Assume 'g' is already initialized routerGroup with a prefix such as "/api".
// g.registerRoute("GET", "/users", usersHandler, loggingMiddleware)
//
// The above will register a route that handles GET requests at "/api/users"
// with loggingMiddleware executed before usersHandler.
func (g *routerGroup) registerRoute(method, path string, handler HandleFunc, ms ...Middleware) {
	// Calculate the full path for the route by prepending the group's prefix
	fullPath := g.calculateFullPath(path) // Assume calculateFullPath does what it says
	// Combine the middleware attached to the group with any additional middleware provided for the route
	middles := append(g.middles, ms...) // Group middleware is applied first, then route-specific middleware
	// Register the route within the parent router using the method, full path, handler and all middleware
	g.router.registerRoute(method, fullPath, handler, middles...)
}

// calculateFullPath constructs the full path for a route by concatenating the routerGroup's prefix
// with the provided path. It ensures that the path given as an argument begins with a forward slash
// and is not empty to prevent malformed URLs. If the path is found to be violating these rules, the method
// panics with an appropriate error message.
//
// Parameters:
// - path: The specific endpoint path that needs to be appended to the routerGroup's prefix.
//
// Returns:
// - string: The complete path that combines the routerGroup's prefix with the provided path.
//
// Panics:
// - This method will panic if the provided 'path' doesn't start with a '/' or if it is an empty string.
//
// Usage:
// Assuming we have a `routerGroup` with a prefix of "/api", calling `calculateFullPath("/users")`
// will return "/api/users".
func (g *routerGroup) calculateFullPath(path string) string {
	// 空路径处理为根路径"/"
	if path == "" {
		return g.prefix
	}

	// Validate that the path starts with a forward slash '/'
	if path[0] != '/' {
		panic(errs.ErrRouterChildConflict()) // Panic with a predefined error if the path is invalid
	}
	// Concatenate the group's prefix with the provided path to form the full path
	return g.prefix + path
}

// GET registers a new GET route within the routerGroup.
// It is a convenience method that wraps the generic registerRoute method,
// specifically setting the HTTP method to "GET". This makes it easier to set up
// GET handlers for specific paths within the group. The GET method is typically used
// for retrieving resources without changing the server's state.
//
// Parameters:
//
//   - path: The endpoint path (relative to the routerGroup's prefix) where the GET handler will be applied.
//     The path should start with a '/' and should not contain the group's prefix, which is automatically
//     prepended to the path in the registerRoute method.
//
//   - handler: The HandleFunc that should be executed when a GET request matches the specified path.
//     It contains the logic to service the GET request for that route.
//
//   - ms: Optional. A variadic sequence of Middleware functions that will be applied to this route.
//     Middleware can perform various tasks such as logging, auth, rate-limiting, etc.,
//     and are executed in the order provided before the handler function upon a route match.
//
// Usage:
// A GET route can be added to the routerGroup like this:
// g.GET("/users", usersHandler, loggingMiddleware, authMiddleware)
// This example would register a GET route at "/users" on the routerGroup's prefix, with both logging
// and auth middleware applied to the route, followed by the execution of usersHandler.
func (g *routerGroup) GET(path string, handler HandleFunc, ms ...Middleware) {
	// Calls the internal registerRoute method, providing the "GET" method
	// along with the path, handler, and any middleware provided in the call.
	g.registerRoute(http.MethodGet, path, handler, ms...)
}

// HEAD registers a route for HTTP HEAD requests. The HEAD method is used to retrieve
// the headers that are returned if the specified resource would be requested with an HTTP GET method.
// Such a request can be done to check what a GET request would return before actually making a GET request -
// like before downloading a large file or response body.
//
// This method associates the path with a specific handler and optional middleware. When a HEAD request
// for the given path is received, the router will execute the middleware in the order they are provided
// before finally calling the given handler to process the request.
//
// Parameters:
//
//   - path: The URL path to be associated with the handler and middleware. It should start with a '/'
//     and be unique within the context of this routerGroup.
//
//   - handler: The HandleFunc to be invoked when the router matches a HEAD request to the specified path.
//     This function will handle the request logic specific to the route.
//
//   - ms: A variadic slice of Middleware functions that will be executed in the order they are passed
//     before the handler is invoked. These functions can perform tasks such as logging, auth,
//     and input validation, among other pre-processing needs.
//
// By using this method, the router is informed about how to handle HEAD requests specifically for the
// path specified. Middleware can be leveraged to handle cross-cutting concerns that are needed across
// various routes.
//
// Usage example:
// Assuming you have a `routerGroup` instance named `api`, you can register a route for a HEAD request as follows:
//
//	api.HEAD("/resources", resourceHandler, loggingMiddleware, authenticationMiddleware)
//
// When a HEAD request is made to '/resources', the `resourceHandler` will be invoked after the
// `loggingMiddleware` and `authenticationMiddleware` have been executed in that order.
func (g *routerGroup) HEAD(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodHead, path, handler, ms...)
}

// POST adds a new route to the routerGroup to handle HTTP POST requests for a specific path.
// The HTTP POST method is used to send data to the server to create or update a resource.
// This method is commonly used when submitting form data or uploading a file.
//
// When a POST request is made to the registered path, the provided handler function is invoked to
// process the request. The route can also have an associated sequence of middleware functions,
// which are executed before the handler function in the order they were provided. Middleware may
// handle concerns such as request logging, auth, rate limiting, or other pre-processing tasks.
//
// Parameters:
//
//   - path: A string representing the endpoint path to which the POST handler will be attached.
//     The path must begin with a '/' and be specific to the context of the routerGroup.
//     It should be noted that the path is appended to any existing base path of the routerGroup.
//
//   - handler: A HandleFunc that is called to process the POST request for the matched path.
//     This function should contain the necessary logic to handle the expected data submission
//     for the route.
//
//   - ms: An optional variadic parameter that allows zero or more Middleware functions to be
//     specified which are to be applied to the route. They are called in the order they are provided
//     and are each given an opportunity to handle or modify the request before it reaches the actual
//     handler function.
//
// Usage:
// The POST route is registered to the routerGroup using this method, and the handler and any
// middleware are specified. For example:
//
//	g.POST("/submit-form", formSubmitHandler, csrfMiddleware, logMiddleware)
//
// This will register a route at the path "/submit-form" that, upon receiving a POST request, will
// process the request using the `formSubmitHandler` after applying CSRF protection and logging actions
// through `csrfMiddleware` and `logMiddleware`, respectively.
//
// The POST method is a critical part of the CRUD operations supported by RESTful services, and it
// enables the client-server interaction necessary for creating resources.
func (g *routerGroup) POST(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodPost, path, handler, ms...)
}

// PUT registers a new route in the routerGroup specifically for handling HTTP PUT requests.
// The PUT method is idempotent and typically used for updating existing resources or creating
// a new resource at a specific URI when the client may already know the resource's URI.
// For instance, updating a user's profile or replacing the contents of a file.
//
// Similar to other routing methods, PUT binds a path with a handler function and an optional
// sequence of middleware functions. The middleware is invoked in the order they are provided
// before the handling function, enabling pre-processing or filtering tasks to be executed prior
// to the main handling logic of the PUT request.
//
// Parameters:
//
//   - path: A string that represents the endpoint path for the PUT request handler, starting with '/'.
//     It is relative to the routerGroup's prefix and should be unique within this routerGroup.
//
//   - handler: The HandleFunc for the PUT request, which includes the core logic for processing the
//     request made to the route's path.
//
//   - ms: Optional. A list of variadic Middleware functions that will be executed sequentially before
//     the request reaches the handler. These can provide additional functionality such as authorization,
//     validation, and logging.
//
// This method is primarily used when the client is sending a complete replacement for a specific resource.
// It differs from POST in that POST may be used for creating a new resource without a given URI, while PUT
// should only be used when referring to a specific resource.
//
// Usage example:
// Here is how to set up a PUT request for updating a user's profile in a group of admin routes:
//
//	adminRoutes.PUT("/users/:id", updateUserHandler, authMiddleware, logMiddleware)
//
// In this case, a PUT request to '/users/:id' will trigger the `updateUserHandler` after successfully
// passing through the `authMiddleware` and `logMiddleware` checks. The ':id' is a path parameter which
// will be used to identify the specific user to be updated.
func (g *routerGroup) PUT(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodPut, path, handler, ms...)
}

// PATCH adds a route to the routerGroup to handle HTTP PATCH requests. Unlike PUT, the PATCH method
// partially updates an existing resource and is not idempotent, meaning successive identical PATCH
// requests may have different effects. It is typically used when the client wants to make changes to
// a single aspect of a resource, such as updating a user's email address without changing the entire user profile.
//
// This operation binds the specified path with a handler function and an optional array of middleware
// functions, which are executed in sequence before the request reaches the handler. Middleware functions
// can handle tasks such as rate-limiting, input validation, and auth. By using this method,
// routes can be assigned specific logic to effectively update parts of resources conditionally.
//
// Parameters:
//
//   - path: A string representing the URL path to which the PATCH method will respond. The path should
//     start with '/' and should be uniquely defined within this routerGroup's namespace.
//
//   - handler: A HandleFunc that defines the logic to be executed for a PATCH request to the specified path.
//     It should contain the code to process the partial update to the resource identified by the path.
//
//   - ms: An optional, variadic set of Middleware functions that will be applied to the request before it
//     reaches the handler. These are used for pre-request processing and are executed in the order provided.
//
// Usage:
// The PATCH method is used within a routerGroup to facilitate conditional updates to resources, enabling
// flexible and targeted modifications. For example:
//
//	g.PATCH("/profile/avatar", updateAvatarHandler, authenticateUser, logUserActivity)
//
// This will create a route at "/profile/avatar" that will handle PATCH requests to update a user's avatar.
// Before the `updateAvatarHandler` is invoked, the middleware functions `authenticateUser` and
// `logUserActivity` are applied, checking if the user is authenticated and logging the user's activity,
// respectively.
func (g *routerGroup) PATCH(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodPatch, path, handler, ms...)
}

// DELETE adds a new route to the routerGroup for handling HTTP DELETE requests. The DELETE method is
// used for deleting resources specified by the URI. This method is idempotent, which means that multiple
// identical requests should have the same effect as a single request. It's most commonly used for operations
// that involve removing resources from a database or file system.
//
// The function creates a binding between an HTTP DELETE request on a given path, a handler function that
// defines the logic for handling the request, and an optional sequence of middleware functions. When a DELETE
// request is made to the registered path, the middlewares, if provided, are executed first, followed by the
// handler function. Middleware functions can perform various tasks, such as authorization checks, rate limiting,
// and request logging.
//
// Parameters:
//
//   - path: A string indicating the endpoint's URI path for which the DELETE request will be handled. It should
//     begin with a '/', denoting the root of the route within the routerGroup's scope, and should be unique
//     to avoid conflicts within the routerGroup.
//
//   - handler: A HandleFunc that contains the code to execute in response to the DELETE request. This function
//     is responsible for the logic that deletes the resource and for sending the appropriate response
//     back to the client, such as a confirmation of deletion or an error message if the resource cannot
//     be found or deleted.
//
//   - ms: An optional array of Middleware functions that can be provided to add intermediate processing steps before
//     the DELETE request reaches the handler. These middleware functions are applied in the order they appear.
//
// Usage:
// The DELETE method is used to set up an endpoint that listens for DELETE requests, enabling clients to request
// resource deletions. For example:
//
//	g.DELETE("/user/:userID", deleteUserHandler, authMiddleware, logMiddleware)
//
// This creates a route that will handle DELETE requests at the path "/user/:userID", where `:userID` is a path
// parameter that represents a specific user's ID. `deleteUserHandler` handles the deletion logic after
// `authMiddleware` authenticates the user who made the request and `logMiddleware` records the request details.
func (g *routerGroup) DELETE(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodDelete, path, handler, ms...)
}

// CONNECT registers a new route in the routerGroup to handle HTTP CONNECT requests. The CONNECT method
// is a specialized mechanism used to establish a tunnel between the client and the server over HTTP.
// This is typically used for facilitating communication through a proxy server by instructing it to
// set up a direct network connection to an upstream server and behave like a transparent tunnel.
//
// This method is particularly useful in scenarios like establishing a Secure Sockets Layer (SSL) connection
// over an HTTP proxy. The route is defined by associating the specified path with a handler function and an
// optional slice of middleware functions. The middleware functions are designed to execute in sequence before
// the handler function is invoked, allowing for preprocessing such as auth and logging of the connection
// requests.
//
// Parameters:
//
//   - path: A string that defines the URI endpoint path that the CONNECT request responds to. It starts with '/'
//     signifying the root in the context of the routerGroup's base path and should be distinct within the
//     routerGroup to prevent overlap with other routes.
//
//   - handler: A HandleFunc which is responsible for implementing the logic to handle the CONNECT request. It should
//     perform needed operations to establish the tunnel, authenticate the request if necessary, and manage
//     the connection lifecycle.
//
//   - ms: An optional series of variadic Middleware functions that are applied to the request before the handler is
//     executed. These can be used for various intermediate processing tasks and are executed in the order they
//     are specified.
//
// Usage:
// The CONNECT method is primarily used to create routes that handle tunneling-like requests over HTTP/HTTPS. For
// example, if a proxy server needs to offer SSL connection establishment features, a route can be configured as:
//
//	g.CONNECT("/secure-tunnel", sslTunnelHandler, authMiddleware)
//
// In this example, a CONNECT request to the path "/secure-tunnel" will initiate the `sslTunnelHandler` after
// passing through the `authMiddleware`, which could authenticate the client request before the tunnel is established.
func (g *routerGroup) CONNECT(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodConnect, path, handler, ms...)
}

// OPTIONS creates a new route in the routerGroup to handle HTTP OPTIONS requests. The OPTIONS method is used
// to describe the communication options for the target resource, allowing the client to determine which HTTP
// methods and other options the server supports for a given URL. This can be useful for features like CORS
// (Cross-Origin Resource Sharing), where an OPTIONS request is sent as a preflight to understand if the
// cross-origin request is safe to send with regards to methods and headers. A response to an OPTIONS request
// typically includes headers such as 'Allow' indicating supported methods and 'Access-Control-Allow-Methods'
// for detailing accepted cross-origin request methods when CORS is in use.
//
// Adding an OPTIONS route to the routerGroup enables defining custom behavior and responses for OPTIONS
// requests, beyond default server configurations. It can be particularly important in APIs that need to
// communicate capabilities to clients or are consumed by web applications using CORS.
//
// Parameters:
//
//   - path: A string specifying the route's URI pattern that the OPTIONS method will respond to. It should
//     start with a '/' representing the route's base in the routerGroup's scope and should be unique to
//     avoid conflicts with other routes within the same routerGroup.
//
//   - handler: A HandleFunc designed to take action upon receiving an OPTIONS request at the specified path.
//     The handler should generate appropriate headers indicating the allowed methods and other
//     options supported by the target resource. It should consider security implications and correctly
//     reflect the server's capabilities.
//
//   - ms: An optional list of Middleware functions, serving as additional processing layers for the OPTIONS
//     request before it reaches the specified handler. These middleware can perform tasks like logging,
//     setting response headers, or other request pre-processing actions, and are invoked in the order
//     they were added.
//
// Usage:
// The OPTIONS method is typically used in RESTful API services to provide clients with information about
// what they can rightfully do with the service. An example of setting up an OPTIONS route is as follows:
//
//	g.OPTIONS("/resource", optionsHandler, corsMiddleware, loggingMiddleware)
//
// Here, an OPTIONS request to the path "/resource" will trigger the `optionsHandler` to run after first
// processing the request through `corsMiddleware` to handle appropriate CORS headers, followed by
// `loggingMiddleware` to record the event. This route can be used by clients to discover allowable
// methods like GET, POST, PUT, DELETE, etc., and headers like 'Content-Type' for interacting with
// "/resource".
func (g *routerGroup) OPTIONS(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodOptions, path, handler, ms...)
}
