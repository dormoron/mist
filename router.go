package mist

import (
	"github.com/dormoron/mist/internal/errs"
	"net/http"
	"regexp"
	"strings"
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
//     for logging, authentication, session management, etc.
type routerGroup struct {
	prefix  string
	parent  *routerGroup
	router  *router
	middles []Middleware
}

// NewGroup creates and returns a new routerGroup attached to the router it is called on.
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
// group := r.NewGroup("/api", loggingMiddleware, authMiddleware)
func (r *router) NewGroup(prefix string, ms ...Middleware) *routerGroup {
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
	// Validate that the path is not empty and starts with a forward slash '/'
	if path == "" || path[0] != '/' {
		panic(errs.ErrRouterChildConflict()) // Panic with a predefined error if the path is invalid
	}
	// Concatenate the group's prefix with the provided path to form the full path
	return g.prefix + path
}

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

// nodeType is an enumerated type (int) used to categorize the different kinds of nodes that can exist within the
// routing structure of a router. Each node in the routing hierarchy represents a segment of a route's path,
// and the type of node can affect route matching behavior and the way in which parameters are extracted from
// the path during route resolution.
//
// A route's path can be composed of fixed, parameterized, or wildcard segments, and the nodeType helps to
// distinguish between these possibilities. Fixed segments match exactly, parameterized segments capture path
// variables, and wildcard nodes can match any segment or sequence of segments.
//
// This custom type enhances type safety by restricting the set of values that can represent node types to those
// explicitly defined in the associated constant definitions that follow the type declaration. This enables
// compile-time checks for the values of nodeType, ensuring that only valid types are used within the routing logic.
//
// Usage of nodeType:
//   - The nodeType is used by internal routing structures to manage and process different kinds of route segments.
//   - It is used within switch statements or conditional blocks when processing incoming paths to determine how to
//     match a given segment and how to proceed with traversal or parameter extraction.
//
// Declaration of nodeType constants:
// Following the type definition, constants are typically declared to represent the allowable nodeType values.
// For example:
//
//	const (
//	    staticNodeType nodeType = iota // Represents a node with a static segment.
//	    paramNodeType                  // Represents a node with a parameterized segment.
//	    wildcardNodeType               // Represents a node with a wildcard segment.
//	)
//
// - `staticNodeType` would be used for routes with fixed paths like "/books".
// - `paramNodeType` would be used for dynamically parameterized paths like "/books/:id" where ":id" is a parameter.
// - `wildcardNodeType` might be used for routes that should match any remaining path like "/files/*filepath".
//
// With these constants, the developer can then work with nodes of specific types without worrying about using
// integer literals throughout the code, enhancing readability, and reducing the potential for errors.
type nodeType int

// Enumeration of node types for structuring route segments within the routing tree. Each constant represents a
// specific kind of route node and dictates how match operations should be conducted for the segment of the path
// it represents. The node types are defined using iota for incrementing integer values starting from 0, which
// provides a unique identifier for each node type.
//
// nodeTypeStatic:
// Used to represent nodes with static path segments. A static node is one that matches exactly with the path segment.
// For example, in the path "/api/books", 'api' and 'books' represent static segments. These segments must be
// present and identical in the request path for a match to occur. Static nodes are the most common and are used
// for fixed-path routing.
//
// nodeTypeReg:
// This type is used for nodes that should match a regular expression pattern. It allows more complex and flexible
// matching beyond static equality. Such nodes allow the matching of segments that conform to a specific pattern
// defined by a regular expression.
//
// nodeTypeParam:
// Used to represent nodes with parameterized path segments. Parameterized segments capture dynamic values. These
// nodes often start with a colon (':') followed by the parameter name in the path pattern (e.g., "/books/:id").
// The actual path segment in the request URL at this position will be captured as a named parameter that can be
// used within the application (e.g., to retrieve a book by its 'id' from a database).
//
// nodeTypeAny:
// Represents nodes that are intended to match any path segment(s). It often symbolizes wildcard or catch-all
// segments in routing, which can be used to capture all remaining path information. For instance, a pattern like
// "/files/*" with a nodeTypeAny node can match any subsequent path elements after "/files/", allowing flexibility
// in handling requests for a variable-depth file directory structure.
//
// These constants are integral to the route matching logic within the routing system. They guide the router when
// determining whether a given route node matches a segment of the request URL and whether to process it as a static
// value, a pattern, a parameter, or a wildcard segment.
const (
	nodeTypeStatic = iota // Indicates a node matches a specific and unchanging route segment.
	nodeTypeReg           // Indicates a node matches a route segment based on a regular expression.
	nodeTypeParam         // Indicates a node represents a named parameter within a route segment.
	nodeTypeAny           // Indicates a node is a wildcard, matching any sequence of route segments.
)

// node represents a segment within the URL path hierarchy of a routing structure.
// Each node can represent a static path segment, a parameter (dynamic segment), or a
// wildcard, and can hold additional information like handlers and middleware necessary for routing.
//
// Fields:
// - typ: nodeType indicates the type of the node (e.g., static, parameter, wildcard).
//
//   - route: A string that captures the full route pattern this node is part of,
//     which could be helpful for debugging or route listing features.
//
// - path: The specific segment of the route that this node represents.
//
//   - children: A map where keys are path segments and values are pointers to child nodes,
//     allowing the representation of a hierarchical routing structure.
//
//   - handler: The HandleFunc to invoke when this node's route is matched. It contains the logic to
//     handle the incoming request for the associated route.
//
//   - starChild: A pointer to a child node that represents a wildcard segment, capturing any text in
//     a path segment where '*” has been used within the route pattern.
//
//   - paramChild: A pointer to a child node that specifies a route parameter segment, such as ':id',
//     which captures a named variable from the path.
//
// - paramName: The name of the parameter captured by this node if it's a paramChild (e.g., 'id' from ':id').
//
//   - mils: A slice of Middleware functions that are associated with the node, to be executed
//     before the handler when the route is matched.
//
//   - matchedMils: A slice of Middleware that were matched and need to be invoked for the current
//     route. This can be built up as the route is resolved.
//
//   - regChild: A pointer to a child node that represents a regular expression pattern segment,
//     which captures complex variable patterns from the path.
//
//   - regExpr: The compiled regular expression (if applicable) that matches the route segment associated
//     with this node.
//
// - parent: A pointer to the parent node in the routing hierarchy, allowing traversal to the root.
//
// Usage:
// The node structure is typically used within the implementation of a router or a middleware
// to build a hierarchical representation of the application's routes. Each route in the application
// corresponds to a chain of nodes, from the root node to the leaf node representing the endpoint.
type node struct {
	typ         nodeType
	route       string
	path        string
	children    map[string]*node
	handler     HandleFunc
	starChild   *node
	paramChild  *node
	paramName   string
	mils        []Middleware
	matchedMils []Middleware
	regChild    *node
	regExpr     *regexp.Regexp
	parent      *node
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

// childrenOf searches through the current node's children to construct a slice of child nodes that match or relate to the given path segment.
// The method considers static (exact match), parameterized, and wildcard children and includes them in the result as they represent possible
// routes for a given path in a routing hierarchy. This is useful in situations where multiple routes could handle the same path, such as in web
// frameworks that support route parameters and wildcards.
//
// Parameters:
// - path: A string representing the path segment to search for among the current node's children.
//
// Returns:
//   - []*node: A slice of pointers to node objects that are children of the current node and correspond to the path segment provided. This slice
//     contains wildcard and parameterized children, with static children (if any match exists) appended last.
func (n *node) childrenOf(path string) []*node {
	// Initialize a slice of node pointers with an initial capacity to store potential children.
	res := make([]*node, 0, 4)

	// Declare a variable to hold a static child node if it exists.
	var static *node

	// Check if the current node has children and attempt to find a static child that matches the path.
	if n.children != nil {
		static = n.children[path]
	}

	// If the current node has a wildcard child, append it to the result slice.
	if n.starChild != nil {
		res = append(res, n.starChild)
	}

	// If the current node has a parameterized child, append it to the result slice.
	if n.paramChild != nil {
		res = append(res, n.paramChild)
	}

	// If a static child exists, append it to the result slice after wildcard and parameterized children.
	if static != nil {
		res = append(res, static)
	}

	// Return the populated slice of child nodes.
	return res
}

// childOf attempts to retrieve a child node associated with a given path segment from the current node.
// It differentiates between exact path matches, parameterized path segments, and wildcard path segments.
// This method is usually called during the traversal of a routing tree in a web framework or any nested
// data structure where nodes may represent different parts of a hierarchical path, such as a filesystem.
//
// Parameters:
// - path: A string representing the exact path segment to match against the node's children.
//
// Returns:
//   - *node: A pointer to the child node that matches the path segment. If there is no exact match, it returns
//     the parameterized or wildcard child node. If no matches are found, it returns nil.
//   - bool: A boolean value that indicates if the returned node is a parameterized child node. It is true if the
//     result is from a parameterized path segment, false otherwise.
//   - bool: A boolean value which indicates whether a successful match was found. It is true if either an exact match,
//     parameterized match, or wildcard match is found, false if there is no child node for the path segment.
func (n *node) childOf(path string) (*node, bool, bool) {
	// If the current node does not have any children nodes, check for parameterized or wildcard child nodes.
	if n.children == nil {
		// If a parameterized child exists, return it along with true for both boolean values, indicating a match and parameterized match.
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		// If only a star child exists (wildcard node), return it with false for parameterized match but true to indicate a match was found.
		return n.starChild, false, n.starChild != nil
	}

	// Attempt to find an exact match for the path in the children node map.
	res, ok := n.children[path]
	if !ok {
		// If no exact match is found, check again for parameterized or wildcard children, similar to the logic above.
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		return n.starChild, false, n.starChild != nil
	}

	// If an exact match is found, return it along with false for both boolean values, indicating an exact match without any parameterization.
	return res, false, ok
}

// childOfNonStatic attempts to find a non-static (dynamic) child node of the current node (n) that matches the given
// path segment. This includes children nodes that represent regular expression patterns, named parameters, or wildcard
// segments. It returns a pointer to the matching child node and a boolean flag indicating whether a match was found.
//
// Parameters:
// - path: A string representing the path segment to match against the current node's dynamic children.
//
// The childOfNonStatic function operates in the following sequence:
//
//  1. Checks if the current node has a regular expression child (regChild). If so, it uses the compiled regular
//     expression stored in regChild.regExpr to determine if the given path segment matches the pattern.
//  2. If a match is confirmed with the regular expression, the regChild node and 'true' are returned to indicate
//     successful matching.
//  3. If there is no regChild or if the path does not match the regular expression, the function then checks whether
//     the current node has a parameterized child (paramChild). Parameterized children represent path segments with
//     named parameters (e.g., /users/:userId).
//  4. If a paramChild exists, it is assumed to match the path segment (since parameterized segments can match any
//     value), and the paramChild node and 'true' are returned.
//  5. If neither a regChild nor a paramChild are applicable, the function finally checks for the presence of a wildcard
//     child (starChild). Wildcard children are used to match any remaining path segments, typically represented by an
//     asterisk (*).
//  6. If a starChild exists, it is returned along with 'true', as it matches any path by definition. If starChild does
//     not exist, the function returns nil and 'false', meaning no match was found among the node's dynamic children.
//
// This method is specifically designed to handle dynamic routing scenarios where path segments may not be known
// statically and can contain patterns, parameters, or wildcards that need to be resolved at runtime.
func (n *node) childOfNonStatic(path string) (*node, bool) {
	// Attempt to match the path segment with a regular expression pattern if regChild exists.
	if n.regChild != nil {
		// If the regular expression matches the path, return the regChild and true.
		if n.regChild.regExpr.Match([]byte(path)) {
			return n.regChild, true
		}
	}

	// If no regular expression match is found, check for a parameterized child node.
	if n.paramChild != nil {
		// Parameterized child nodes match any path segment, so return the paramChild and true.
		return n.paramChild, true
	}

	// If no other dynamic match is found, check for a wildcard child node.
	// Wildcard nodes (if any) match any path segment, so return starChild and a boolean indicating its existence.
	return n.starChild, n.starChild != nil
}

// childOrCreate locates a child node within the current node (n) that matches the given 'path' or creates a new
// child node if one does not already exist. It handles different node types: static, parameterized, regular
// expression-based, and wildcard. The method returns a pointer to the child node. If the given 'path' represents a
// wildcard or parameterized path and violates the routing rules (such as being mixed with parameterized paths or
// regular expressions), the method panics with an appropriate error.
//
// Parameters:
// - path: A string representing the path segment to match against or to create within the current node's children.
//
// The childOrCreate function operates as follows:
//
//  1. Checks if the given 'path' is a wildcard "*". If so, it ensures that no parameterized (paramChild) or regular
//     expression-based (regChild) children exist, as these are not allowed in conjunction with a wildcard. If this
//     rule is violated, a panic occurs with a descriptive error message.
//  2. If a wildcard child node does not exist, it creates one, initializes it with the path, and sets its type to
//     nodeTypeAny.
//  3. If the given 'path' starts with ':', indicating it is a parameterized path, the method parses the parameter name
//     and any associated regular expression (if present) using 'parseParam'.
//  4. Depending on whether a regular expression is part of the parameterized path, it calls either 'childOrCreateReg'
//     or 'childOrCreateParam' to either create or fetch the existing child.
//  5. If 'path' does not start with '*' or ':', indicating a static path, it initializes 'n.children' if it's nil and
//     then looks for or creates a static child node with the given path.
//  6. It inserts the new static child node into the 'children' map if it does not exist already and initializes it
//     with the path and type 'nodeTypeStatic'.
//
// Note:
// - This method modifies the current node 'n', potentially adding new child nodes to it.
// - This method assumes that 'path' is a non-empty string.
func (n *node) childOrCreate(path string) *node {
	// Wildcard path handling: creates or retrieves a wildcard child, enforcing rules against mixing wildcard
	// with parameter and regular expression children.
	if path == "*" {
		// Check and enforce routing rule: Wildcards cannot exist alongside parameterized paths.
		if n.paramChild != nil {
			panic(errs.ErrPathNotAllowWildcardAndPath(path))
		}
		// Check and enforce routing rule: Wildcards cannot exist alongside regular expression paths.
		if n.regChild != nil {
			panic(errs.ErrRegularNotAllowWildcardAndRegular(path))
		}
		// Create a wildcard child node if one does not exist, initialize and store it for future retrievals.
		if n.starChild == nil {
			n.starChild = &node{path: path, typ: nodeTypeAny}
		}
		return n.starChild // Return the wildcard child node.
	}

	// Parameterized path handling: parses the parameter name and expression, and creates or retrieves
	// the corresponding parameterized or regular expression child node.
	if path[0] == ':' {
		paramName, expr, isReg := n.parseParam(path)
		if isReg {
			// For paths with an embedded regular expression, create or retrieve a regular expression child node.
			return n.childOrCreateReg(path, expr, paramName)
		}
		// For standard parameterized paths, create or retrieve a parameterized child node.
		return n.childOrCreateParam(path, paramName)
	}

	// Static path handling: creates or retrieves a static child node.
	if n.children == nil {
		// Initialize the children map if it hasn't been already to prevent nil map assignment errors.
		n.children = make(map[string]*node)
	}
	// Look for or create a static child node for the given path.
	child, ok := n.children[path]
	if !ok {
		// If the child node does not exist already, create it, initialize it with the path and type,
		// and add it to the children map.
		child = &node{path: path, typ: nodeTypeStatic}
		n.children[path] = child
	}
	return child // Return the static child node.
}

// childOrCreateParam is used to retrieve an existing or create a new parameterized child node associated with the current
// node (n). It manages nodes that represent path parameters in a URL, usually denoted by a colon (':') followed by the
// parameter name (e.g., ":id" in "/users/:id"). The method ensures that parameter nodes do not coexist with wildcard or
// regular expression nodes, as per routing rules. It panics if a routing conflict occurs.
//
// Parameters:
// - path: The path segment that the method attempts to match or create a node for.
// - paramName: The name of the parameter as extracted from the path.
//
// The childOrCreateParam function performs the following actions:
//
//  1. First, it checks if the current node has a child that is a regular expression node (regChild). If such a child
//     exists, it's considered a routing conflict because a regular expression child cannot coexist with a parameterized
//     path. In this case, the method panics with an appropriate error.
//  2. Next, it checks for the presence of a wildcard child (starChild). Again, as per routing rules, a wildcard child
//     cannot coexist with a parameterized child, and if found, the method panics with an error.
//  3. The method then checks if a parameterized child node (paramChild) already exists. If it does and its path differs
//     from the given 'path', this is considered a routing conflict (two different parameterized paths cannot be the same
//     route segment), prompting the method to panic with a path clash error.
//  4. If no parameterized child exists or if the existing one has the same path, the method is safe to proceed. If a new
//     child needs to be created, it's initialized with the given 'path', 'paramName', and set to nodeTypeParam to
//     denote its nature as a parameterized node.
//  5. Finally, the existing or newly created parameterized child node is returned.
//
// Note:
// - This method updates the current node 'n' by potentially adding a paramChild.
// - It only handles parameterized paths and is part of a broader routing system with rules to prevent routing conflicts.
func (n *node) childOrCreateParam(path string, paramName string) *node {
	// Enforce routing rules by checking for the presence of regular expression and wildcard children,
	// and panic if necessary to prevent invalid routing configurations.
	if n.regChild != nil {
		panic(errs.ErrRegularNotAllowRegularAndPath(path))
	}
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndPath(path))
	}
	// Check if a parameterized child node already exists with the same path.
	if n.paramChild != nil {
		// If the paths differ, this denotes a routing conflict, and panic with an error.
		if n.paramChild.path != path {
			panic(errs.ErrPathClash(n.paramChild.path, path))
		}
	} else {
		// If no parameterized child exists, create one with the provided path and parameter name.
		n.paramChild = &node{path: path, paramName: paramName, typ: nodeTypeParam}
	}
	// Return the existing or newly created parameterized child node.
	return n.paramChild
}

// childOrCreateReg retrieves or creates a child node that represents a path segment with an embedded regular
// expression. This method is called when the path segment includes a parameter with a custom regular expression
// constraint, denoting a more complex matching requirement than standard parameterized routes.
//
// Parameters:
// - path: The full path segment including the parameter and its associated regular expression (e.g., ":id(\\d+)").
// - expr: The raw regular expression string used to match this parameter (e.g., "\\d+").
// - paramName: The name of the parameter to be extracted from the path (e.g., "id").
//
// The childOrCreateReg function performs these steps:
//
//  1. It ensures that no wildcard child (starChild) exists, as mixing wildcards with regular expression constrained
//     parameters is not permissible. If a wildcard is present, the function panics with the appropriate error.
//  2. It ensures that no simple parameterized child (paramChild) exists, as such nodes cannot coexist with regular
//     expression constrained parameters. If found, the function panics with a relevant error message.
//  3. If a regular expression child (regChild) already exists, the method checks that its regular expression and
//     parameter name match the current ones. If they do not, indicating a clash in the routing definitions, the
//     method panics with a routing conflict error.
//  4. If no regChild exists that meets the required criteria, the method creates one. This involves compiling the
//     passed regular expression to create a 'regexp.Regexp' object. If compiling fails, it panics with an error
//     that indicates an issue with the regular expression.
//  5. Finally, the new or existing regular expression child node is returned.
//
// Note:
//   - The method updates the current 'node' by adding a regChild if necessary.
//   - It only manages nodes with regular expression constraints and upholds routing system integrity by checking for
//     potential routing definition clashes.
func (n *node) childOrCreateReg(path string, expr string, paramName string) *node {
	// Check for and enforce routing conflicts with wildcard and param nodes. Panic if a conflict exists.
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndRegular(path))
	}
	if n.paramChild != nil {
		panic(errs.ErrPathNotAllowPathAndRegular(path))
	}
	// If a regular expression child already exists, ensure it matches the new requirements. Otherwise, panic.
	if n.regChild != nil {
		// A routing definition clash occurs when the existing regChild's regular expression or parameter name
		// does not match the new requirements. Panic with an error indicating this conflict.
		if n.regChild.regExpr.String() != expr || n.paramName != paramName {
			panic(errs.ErrRegularClash(n.regChild.path, path))
		}
	} else {
		// Compile the new regular expression, and panic with an error if there's an issue with the compilation.
		regExpr, err := regexp.Compile(expr)
		if err != nil {
			panic(errs.ErrRegularExpression(err))
		}
		// If successful, create a new regChild node with the compiled expression and other data, and assign it to the current node.
		n.regChild = &node{path: path, paramName: paramName, regExpr: regExpr, typ: nodeTypeReg}
	}
	// Return the existing or newly created regChild node.
	return n.regChild
}

// parseParam analyzes a given path segment to identify and extract the name of the parameter and, optionally,
// any regular expression associated with it. This is used in routing to handle dynamic segments in URLs. The
// method returns a tuple with the parameter name, the extracted regular expression (if any), and a boolean
// indicating whether a regular expression was found.
//
// Parameters:
//   - path: A string representing the segment of the URL path that contains the parameter. This should start with
//     a colon (':') followed by the parameter name and may include an embedded regular expression.
//
// The parseParam function proceeds as follows:
//
//  1. Removes the leading colon (':') from the path segment, as it only serves as an identifier of a parameter segment.
//  2. Splits the remaining string into two parts at the first occurrence of an opening parenthesis '(' which would
//     indicate the start of a regular expression constraint on the parameter.
//  3. If the split result has two segments, then a regular expression is assumed to be present:
//     - It further checks if the second segment has a closing parenthesis ')'. This confirms a well-formed regular
//     expression constraint. If it is well-formed, the regular expression is extracted, excluding the parentheses.
//     - It returns the parameter name, the regular expression without the enclosing parentheses, and true (for the
//     boolean indicating the presence of a regular expression).
//  4. If no regular expression is found or the regular expression is not well-formed (e.g., missing the closing
//     parenthesis or not having any parentheses at all), it returns the parameter name as the whole path after
//     the colon, an empty string for the regular expression, and false (no regular expression was found).
//
// Note:
//   - This method is utilized when building the routing tree to recognize and correctly process different node types
//     based on their path definitions.
//   - It is crucial for ensuring that URL parameters can be correctly matched and extracted during request handling.
func (n *node) parseParam(path string) (string, string, bool) {
	// Remove the leading colon from the path to isolate the parameter name and potential regular expression.
	path = path[1:]
	// Attempt to split the path segment at the opening parenthesis to separate the parameter name from any regular expression.
	segs := strings.SplitN(path, "(", 2)
	// Check if a regular expression is present by seeing if there are two segments after the split.
	if len(segs) == 2 {
		// Assuming the second segment is a regular expression, check if it ends with a closing parenthesis.
		expr := segs[1]
		if strings.HasSuffix(expr, ")") {
			// If so, return the parameter name, the regular expression without parentheses, and true.
			return segs[0], expr[:len(expr)-1], true
		}
	}
	// If there is no regular expression, return the parameter name, an empty string, and false.
	return path, "", false
}

// matchInfo holds the necessary information for a matched route. It encapsulates the node that has been matched,
// any path parameters extracted from the URL, and a list of middleware that should be applied for the route.
// This struct is typically used in the context of a routing system, where it is responsible for carrying the
// cumulative data required to handle an HTTP request after a route has been successfully matched.
//
// Fields:
//   - n (*node): A pointer to the matched 'node' which represents the endpoint in the routing tree that has been
//     matched against the incoming request path. This 'node' contains the necessary information to process
//     the request further, such as associated handlers or additional routing information.
//   - pathParams (map[string]string): A map that stores the path parameters as key-value pairs, where the key is
//     the name of the parameter (as defined in the path) and the value is the actual
//     string that has been matched from the request URL. For example, for a route
//     pattern "/users/:userID/posts/:postID", this map would contain entries for
//     "userID" and "postID" if the incoming request path matched that pattern.
//   - mils ([]Middleware): A slice of 'Middleware' functions that are meant to be executed for the matched route
//     in the order they are included in the slice. Middleware functions are used to perform
//     operations such as request logging, authentication, and input validation before the
//     request reaches the final handler function.
//
// Usage:
// The 'matchInfo' struct is populated during the route-matching process. Once a request path is matched against
// the routing tree, a 'matchInfo' instance is created and filled with the corresponding node, extracted path
// parameters, and any middleware associated with the matched route. This instance is then passed along to the
// request handling logic, where it guides the processing of the request through various middleware layers and
// eventually to the appropriate handler that will generate the response.
type matchInfo struct {
	// n is the node corresponding to the matched route in the routing tree. It provides access to any additional
	// route-specific information required to handle the request.
	n *node

	// pathParams stores the parameters identified in the URL path, such as "id" in "/users/:id", mapped to their
	// actual values as resolved from the incoming request.
	pathParams map[string]string

	// mils is a collection of middleware functions to be executed sequentially for the matched route. These functions
	// can modify the request context, perform checks, or carry out other pre-processing tasks.
	mils []Middleware
}

// addValue is a method that adds a key-value pair to the pathParams map of the matchInfo struct. This method
// serves to accumulate the parameters extracted from a matched URL path and store them for later use during
// the request-handling process.
//
// Parameters:
//   - key: A string representing the name of the URL parameter (e.g., "userID").
//   - value: A string representing the value of the URL parameter that has been extracted from the request
//     URL (e.g., "42" for a userID).
//
// The addValue function performs these steps:
//
//  1. Checks if the pathParams map inside the matchInfo struct is nil, which would indicate that no parameters
//     have been added yet. If it is nil, it initializes the pathParams map and instantly adds the key-value
//     pair to it. This is necessary because you cannot add keys to a nil map; it must be initialized first.
//  2. If the pathParams map is already initialized, it adds or overwrites the entry for the key with the new value.
//     This ensures that the most recently processed value for a given key is stored in the map.
//
// Usage:
// The addValue method is typically called during the route matching process, where path segments corresponding
// to parameters in the route pattern are parsed and their values accumulated. Each time a segment is processed
// and a parameter value is extracted, addValue is used to save that value with the corresponding parameter name.
//
// Example:
// For a URL pattern like "/users/:userID", when processing a request path like "/users/42", the method would
// be invoked as addValue("userID", "42"), adding the parameter "userID" with the value "42" to the pathParams map.
func (m *matchInfo) addValue(key string, value string) {
	// Initialize the pathParams map if it hasn't been already to avoid nil map assignment panic.
	if m.pathParams == nil {
		m.pathParams = map[string]string{key: value}
	}
	// Add or update the pathParams map with the key-value pair representing the URL parameter and its value.
	m.pathParams[key] = value
}

// GET registers a new GET route within the routerGroup.
// It is a convenience method that wraps the generic registerRoute method,
// specifically setting the HTTP method to "GET". This makes it easier to set up
// GET handlers for specific paths within the group. The GET method is typically used
// for retrieving resources without changing the server’s state.
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
//     Middleware can perform various tasks such as logging, authentication, rate-limiting, etc.,
//     and are executed in the order provided before the handler function upon a route match.
//
// Usage:
// A GET route can be added to the routerGroup like this:
// g.GET("/users", usersHandler, loggingMiddleware, authMiddleware)
// This example would register a GET route at "/users" on the routerGroup's prefix, with both logging
// and authentication middleware applied to the route, followed by the execution of usersHandler.
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
//     before the handler is invoked. These functions can perform tasks such as logging, authentication,
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
// handle concerns such as request logging, authentication, rate limiting, or other pre-processing tasks.
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
// can handle tasks such as rate-limiting, input validation, and authentication. By using this method,
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
// `logUserActivity` are applied, checking if the user is authenticated and logging the user’s activity,
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
// the handler function is invoked, allowing for preprocessing such as authentication and logging of the connection
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

// TRACE registers a new route to the routerGroup to handle HTTP TRACE requests. The TRACE method is primarily
// used for diagnostic purposes and is a way to echo back the incoming request so that clients can see what
// (if any) changes or additions have been made by intermediate servers. It can be used to test and diagnose
// the network path between the client and the server, and to ensure the integrity of request data such as
// HTTP headers and query parameters.
//
// It is important to note that the TRACE method should be used with caution as it can introduce some security
// vulnerabilities, particularly the potential for cross-site tracing attacks where malicious scripts can gain
// access to information in HTTP cookies or authorization headers through JavaScript. Therefore, it is typical
// for TRACE requests to be blocked at the network edge or web application firewall.
//
// Parameters:
//
//   - path: A string defining the path for the TRACE route. This path is relative to the routerGroup's base path.
//     The function will match incoming HTTP requests that use the TRACE method where the URI matches this path.
//
//   - handler: A HandleFunc that will be executed when a TRACE request matches the specified path. The handler function
//     is responsible for processing the request and preparing the response that will be echoed back to the client.
//
//   - ms: A variadic list of optional Middleware functions that can be specified to perform additional processing on the
//     request before it reaches the handler function. These Middleware functions are called sequentially in the order
//     they were provided.
//
// Example usage:
// Imagine that we want to provide a route that clients can use to trace HTTP requests to '/tracepath'. The route will
// be set up with an additional middleware that logs each request. The following code snippet would establish such a route:
//
//	g.TRACE("/tracepath", traceHandler, logRequestMiddleware)
//
// In this configuration, any TRACE requests to '/tracepath' will first be logged by the `logRequestMiddleware`. Then,
// the `traceHandler` will be called to process the request further, typically by echoing the received request headers
// and body back to the client.
func (g *routerGroup) TRACE(path string, handler HandleFunc, ms ...Middleware) {
	g.registerRoute(http.MethodTrace, path, handler, ms...)
}
