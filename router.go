package mist

import (
	"github.com/dormoron/mist/internal/errs"
	"strings"
)

// router is a data structure that is used to store and retrieve the routing information
// of a web server or similar application which requires URL path segmentation and pattern matching.
// This struct is at the core of the routing mechanism and is responsible for associating
// URL paths with their corresponding handlers.
//
// The 'router' struct consists of a map called 'trees', where the keys are HTTP methods
// like "GET", "POST", "PUT", etc. For each HTTP method, there is an associated 'node' which
// is the root of a tree data structure. This tree is used to store the routes in a hierarchical
// manner, where each part of the URL path is represented as a node within the tree.
//
// By breaking down the URL paths into segments and structuring them in a tree fashion, the 'router'
// struct allows for quick and efficient matching of URL paths to their respective handlers,
// even as the number of routes grows large. This organization provides an improvement over linear
// search methods, reducing the routing complexity from O(n) to O(k), where k is the number of path
// segments in the URL.
//
// The 'trees' map is therefore a critical component to efficiently handle HTTP requests matching
// and dispatching, supporting dynamic URL patterns and increasing the performance and scalability
// of the application.
//
// Example of 'router' usage:
// - Creating new router instance:
//
//	r := &router{
//	    trees: make(map[string]*node),
//	}
//
// - Adding routes to router:
//
//	r.addRoute("GET", "/home", homeHandler)
//	r.addRoute("POST", "/users", usersHandler)
//	...
//
// - Using the router to match a request's method and path to the appropriate handler:
//
//	handler, params := r.match("GET", "/home")
//	if handler != nil {
//	  handler.ServeHTTP(w, r)
//	}
//
// Considerations:
//   - For maximal efficiency, the trees should only be modified during the initialization phase
//     of the application, or at times when no requests are being handled, as modifications to the
//     tree structure could otherwise lead to race conditions or inconsistent routing.
//   - Expansion of dynamic route parameters (like '/users/:userID') is supported by most routing trees,
//     this should be taken into account when designing the node and its matching algorithm.
//   - Error handling such as detecting duplicate routes, invalid patterns, or unsupported HTTP methods,
//     should be considered and implemented according to the needs of the application.
type router struct {
	trees map[string]*node
}

// initRouter is a factory function that initializes and returns a new instance of the 'router' struct.
// This function prepares the 'router' for use by setting up its internal data structures, which are necessary
// for registering and matching URL routes to their associated handlers within a web application.
//
// When called, it constructs a 'router' instance with an initialized 'trees' map, which is essential for
// storing the root nodes of route trees for different HTTP methods. The 'trees' map is keyed on HTTP methods
// as strings, such as "GET", "POST", "PUT", etc., and the values are pointers to 'node' struct instances
// that represent the root of a tree for that HTTP method.
//
// Since the routing logic requires a distinct tree for each HTTP method to allow for efficient route matching
// and to accommodate the unique paths that may exist under each method, the 'trees' map is initialized as empty
// with no root nodes. Roots are typically added as new routes are registered to the router via a separate
// function or method, not shown in this example.
//
// Usage of the 'initRouter' function is typically seen during the setup phase of a web server or application,
// where routing is being established before starting to serve requests. The returned 'router' instance is
// then ready to have routes added to it and subsequently used to route incoming HTTP requests to the correct
// handler based on the path and method.
//
// Example of usage:
// - Initializing a new router instance at application startup
//
//		r := initRouter()
//		r.addRoute("GET", "/", homeHandler)
//		r.addRoute("POST", "/users", usersHandler)
//		...
//		http.ListenAndServe(":8080", r)
//
//	  - The application's main or setup function would typically include a call to 'initRouter' followed by route
//	    registration code and eventually starting the server with the router configured.
//
// Considerations:
//   - It's important that this initialization is done before the router begins handling any requests to ensure
//     thread safety. If the application is expected to modify the router after serving requests has started,
//     proper synchronization mechanisms should be employed.
//   - The 'initRouter' function abstracts the initialization details and ensures that all required invariants
//     of the 'router' struct are satisfied, improving code readability and safety by centralizing router setup logic.
func initRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// Group creates and returns a new routerGroup attached to the router it is called on.
// The method ensures the provided prefix conforms to format requirements by checking
// if it starts and does not end with a forward slash, unless it is the root group ("/").
// This method panics if the prefix is invalid to prevent router misconfiguration.
//
// Parameters:
//   - prefix: The path prefix for the new group of routes. It should start with a forward slash and,
//     except for the root group, not end with a forward slash.
//   - ms: Zero or more Middleware functions that will be applied to every route
//     within the created routerGroup.
//
// Returns:
// *routerGroup: A pointer to the newly created routerGroup with the given prefix and middlewares.
//
// Panics:
// This method will panic if 'prefix' does not start with a '/' or if 'prefix' ends with a '/'
// (except when 'prefix' is exactly "/").
//
// Usage:
// r := &router{} // Assume router is already initialized
// group := r.Group("/api", loggingMiddleware, authMiddleware)
func (r *router) Group(prefix string, ms ...Middleware) *routerGroup {
	// Check if the prefix is empty or doesn't start with a '/'
	if prefix == "" || prefix[0] != '/' {
		panic(errs.ErrRouterGroupFront()) // Panic with a predefined error for incorrect prefix start
	}
	// Check if the prefix is not the root '/' and ends with a '/'
	if prefix != "/" && prefix[len(prefix)-1] == '/' {
		panic(errs.ErrRouterGroupBack()) // Panic with a predefined error for incorrect prefix end
	}
	// If the prefix is correct, initialize a new routerGroup with the provided details and return it
	return &routerGroup{prefix: prefix, router: r, middles: ms}
}

// registerRoute is a method for registering a new route within the router. It associates an HTTP method and path with
// a specific handler function and a slice of optional middleware. This method updates the routing tree of the router,
// ensuring that incoming requests matching the method and path can be properly dispatched to the handler. If any errors
// are detected during the registration process (such as invalid paths or conflicting routes), the method will panic.
//
// Parameters:
// - method: The HTTP method (e.g., GET, POST, etc.) for which the route is being registered.
// - path: The URL path that the route will handle, starting with a forward slash '/'.
// - handler: The HandleFunc type function that will be invoked when the route is matched.
// - ms: A variadic slice of Middleware functions that will be applied to the request before the handler is invoked.
//
// The method performs the following actions:
//
//  1. Validates the input path ensuring that it is not empty, starts with a forward slash '/' and does not end with a
//     slash unless it is the root path "/".
//  2. Looks up the root node for the provided HTTP method in the router's trees. If the tree for that method does not
//     exist, it is created with an initial root node.
//  3. For the special case of the root path "/", it immediately registers the handler and associated middleware.
//     If the root node already has a handler, it panics to prevent route conflicts.
//  4. For non-root paths, it splits the path into segments and iteratively creates or retrieves nodes in the routing tree
//     corresponding to each segment.
//  5. Checks if the final node in the segment sequence already has a handler to avoid route conflicts. If a handler exists,
//     it panics to signify a conflict with an existing route.
//  6. Registers the handler and route path at the final node found or created from the segments.
//  7. Assigns the provided middleware functions to the final node, completing the route's registration.
//
// This method ensures that the routing tree accurately reflects all registered routes for each HTTP method, with the
// appropriate handlers and middleware attached.
func (r *router) registerRoute(method string, path string, handler HandleFunc, ms ...Middleware) {
	// Validate the incoming path to ensure it follows the expected format.
	if path == "" {
		// An empty path is invalid and indicative of an erroneous registration call.
		panic(errs.ErrRouterNotString())
	}
	if path[0] != '/' {
		// All paths must start with a '/' character, denoting the path's beginning relative to the root.
		panic(errs.ErrRouterFront())
	}
	if path != "/" && path[len(path)-1] == '/' {
		// No path (other than the root path "/") should end with a '/', preventing ambiguities in route matching.
		panic(errs.ErrRouterBack())
	}

	// Obtain or initialize the root node for the specified HTTP method.
	root, ok := r.trees[method]
	if !ok {
		// If no such node exists, create and map one for the specified method.
		root = &node{path: "/"}
		r.trees[method] = root
	}

	// Register the route for the root path "/".
	if path == "/" {
		// Check if a handler for the root path is already set and panic if so to signal the conflict.
		if root.handler != nil {
			panic(errs.ErrRouterConflict("/"))
		}
		// Assign the handler and middleware to the root node for the provided HTTP method.
		root.handler = handler
		root.route = "/"
		root.mils = ms
		return
	}

	// Process each segment in the path to build the respective nodes in the routing tree.
	segs := strings.Split(path[1:], "/") // Remove the leading '/' and split the path into segments.
	for _, s := range segs {
		// Each path segment must be a valid string.
		if s == "" {
			panic(errs.ErrRouterNotSymbolic(path))
		}
		// Create or retrieve the child node for each segment, updating root to point to the latest node.
		root = root.childOrCreate(s)
	}

	// At the final segment node, check and set the route handler, avoiding conflicts.
	if root.handler != nil {
		// If a handler is already set for this path, panic to avoid overwriting an existing route.
		panic(errs.ErrRouterConflict(path))
	}

	// Set the handler and middleware for the final node in the path sequence, registering the route.
	root.handler = handler
	root.route = path
	root.mils = appendCollectMiddlewares(root, ms)
}

// appendCollectMiddlewares traverses up the tree from the given node to the root and collects all
// middleware in the order from the root to the node. This function is typically used to gather all
// middleware that should be applied to a request, as it travels from the root node down to a specific route.
//
// Parameters:
//   - n: A pointer to the starting node from where to begin collecting middleware. This usually represents
//     a node matched by a specific route.
//   - ms: A slice of Middleware that may already contain some middleware. Additional middleware from the
//     nodes will be prepended to this slice.
//
// Returns:
//   - []Middleware: A slice of Middleware that includes the provided middleware and all middleware gathered
//     from traversing the tree up to the root node.
func appendCollectMiddlewares(n *node, ms []Middleware) []Middleware {
	// Initialize a local slice starting with the passed-in middleware.
	// This will allow middleware to be collected in the correct order.
	middles := ms // The slice that will ultimately contain the collected middleware.

	// Iterate up the tree starting from the given node until the root is reached.
	for n != nil {
		// Prepend the middleware of the current node to the 'middles' slice.
		// Prepending ensures that the middleware is in the correct order when applying it later: from root down to the node.
		middles = append(n.mils, middles...)
		// Climb up to the parent node for the next iteration.
		n = n.parent
	}
	// Return the collected middleware in the tree from the root to the given node.
	return middles
}

// findRoute searches the router's trees to find a route that matches the specified HTTP method and path.
// It traverses the tree according to the path segments, collecting matching nodes and associated middleware.
// If a matching route is found, it creates a `matchInfo` struct detailing the matched node and middleware.
// This is commonly used in web frameworks to resolve incoming requests to their appropriate handlers.
//
// Parameters:
// - method: A string representing the HTTP method to match (GET, POST, etc.).
// - path: A string representing the request path that needs to be matched to a route.
//
// Returns:
// - *matchInfo: A pointer to a `matchInfo` struct representing the matched route info, if a route is found.
// - bool: A boolean indicator that is true if a route is found, false otherwise.
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	// Attempt to retrieve the root node for the HTTP method from the router's trees.
	root, ok := r.trees[method]
	// If the method does not have a corresponding tree, return no match.
	if !ok {
		return nil, false
	}

	// Special case for root path "/".
	if path == "/" {
		// Return the root node's information with associated middleware, indicating a match is found.
		return &matchInfo{n: root, mils: root.mils}, true
	}

	// Split the path into segments for traversal, ignoring any trailing slashes.
	segs := strings.Split(strings.Trim(path, "/"), "/")
	// Initialize matchInfo to store the matching route's info as we traverse.
	mi := &matchInfo{}
	// Start from the root node.
	cur := root
	// Loop through the path segments to traverse the routing tree.
	for _, s := range segs {
		var matchParam bool // Used to check if the current node match is a parameterized path segment.

		// Find the child node matching the current path segment, capturing if it's a match with a parameter.
		cur, matchParam, ok = cur.childOf(s)
		// If there's no corresponding child node, the path does not match any route, return no match.
		if !ok {
			return &matchInfo{}, false
		}
		// If the current node match is a parameterized segment, record the parameter value in matchInfo.
		if matchParam {
			mi.addValue(root.path[1:], s)
		}
	}

	// Having traversed all segments, assign the last node and collected middleware to `mi`.
	mi.n = cur
	mi.mils = r.findMils(root, segs)
	// Return the populated matchInfo indicating a successful match.
	return mi, true
}

// findMils searches through the routing tree for all middleware associated with the provided path segments.
// It traverses the tree following the path segments, collecting middleware from each matching node along
// the way. This process is typical of a web framework's router, where middleware needs to be gathered and
// applied in the order it's defined along a route.
//
// Parameters:
// - root: A pointer to the root node of the routing tree from where the search begins.
// - segs: A slice of strings representing the individual segments of the path to search for.
//
// Returns:
//   - []Middleware: A slice of Middleware that has been collected from the routing tree. The middleware is
//     accumulated in the order encountered during the traversal.
func (r *router) findMils(root *node, segs []string) []Middleware {
	// Initialize a queue with the root node to begin the level-order traversal.
	queue := []*node{root}
	// Create a slice to store the middleware found.
	res := make([]Middleware, 0, 16)

	// Loop through each segment in the path.
	for i := 0; i < len(segs); i++ {
		seg := segs[i]       // Current path segment.
		var children []*node // Keep track of the children nodes of the current queue nodes.

		// Loop through the current queue to search for middleware and child nodes matching the current segment.
		for _, cur := range queue {
			// Check if the current node has middleware and append it to the result if it does.
			if len(cur.mils) > 0 {
				res = append(res, cur.mils...)
			}
			// Collect all children of the current node that correspond to the current path segment.
			children = append(children, cur.childrenOf(seg)...)
		}
		// Update the queue with the newly found children nodes for the next iteration.
		queue = children
	}

	// After going through all the segments, check if any of the remaining nodes in the queue have middleware to append.
	for _, cur := range queue {
		if len(cur.mils) > 0 {
			res = append(res, cur.mils...)
		}
	}
	// Return the collected middleware.
	return res
}
