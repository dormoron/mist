package mist

import (
	"github.com/dormoron/mist/internal/errs"
	"regexp"
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

// registerRoute is a method for registering a new route within the router. It associates an HTTP method and path with
// a specific handler function and a slice of optional middleware. This method updates the routing tree of the router,
// ensuring that incoming requests matching the method and path can be properly dispatched to the handler. If any errors
// are detected during the registration process (such as invalid paths or conflicting routes), the method will panic.

// Parameters:
// - method: The HTTP method (e.g., GET, POST, etc.) for which the route is being registered.
// - path: The URL path that the route will handle, starting with a forward slash '/'.
// - handler: The HandleFunc type function that will be invoked when the route is matched.
// - ms: A variadic slice of Middleware functions that will be applied to the request before the handler is invoked.

// The method performs the following actions:

// 1. Validates the input path ensuring that it is not empty, starts with a forward slash '/' and does not end with a
//    slash unless it is the root path "/".
// 2. Looks up the root node for the provided HTTP method in the router's trees. If the tree for that method does not
//    exist, it is created with an initial root node.
// 3. For the special case of the root path "/", it immediately registers the handler and associated middleware.
//    If the root node already has a handler, it panics to prevent route conflicts.
// 4. For non-root paths, it splits the path into segments and iteratively creates or retrieves nodes in the routing tree
//    corresponding to each segment.
// 5. Checks if the final node in the segment sequence already has a handler to avoid route conflicts. If a handler exists,
//    it panics to signify a conflict with an existing route.
// 6. Registers the handler and route path at the final node found or created from the segments.
// 7. Assigns the provided middleware functions to the final node, completing the route's registration.

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
	root.mils = ms
}

// findRoute is a method used to locate a specific route based on the HTTP method and path provided. It attempts to find
// a matching node in the router's routing tree that corresponds to the route being searched. If a match is found, it
// constructs and returns a 'matchInfo' struct containing the matched node and any associated middleware. It also
// returns a boolean indicating whether a route match was found. This method is essential for the router's ability to
// dispatch incoming requests to the appropriate handlers based on the request's method and path.

// Parameters:
// - method: The HTTP request method ('GET', 'POST', etc.) used to locate the routing sub-tree.
// - path:   The path of the request used to search for a matching node within the routing sub-tree.

// The method operates as follows:

//  1. It first looks for a subtree corresponding to the request method. Every HTTP method (GET, POST, etc.) has an
//     associated sub-tree.
//  2. If no sub-tree is found for the method, it indicates that there are no routes registered for that HTTP method, and
//     it returns false, indicating no route match.
//  3. For the special path "/", it immediately returns a matchInfo with the root node and its middleware since this is a
//     common special case representing the root of the domain.
//  4. It splits the path string into segments by trimming the leading and trailing "/" and then splitting by "/".
//  5. It creates an empty 'matchInfo' struct for storing information about the matched route as it traverses the tree.
//  6. It iterates over the path segments, attempting to find a corresponding child node in the routing tree from the
//     root node down to the leaf nodes.
//     a. For each segment, the algorithm searches for the children of the current 'root' (iterative context) that match
//     the segment, using the 'childOf' method. If no child is found, and the current node is not a catch-all (nodeTypeAny),
//     then there is no complete match, and it returns false.
//     b. If the segment matches a parameterized node, it captures the parameter value and adds it to the 'matchInfo'.
//     c. The search context 'root' is updated to point to the child for the next iteration.
//  7. After all segments have been processed, the 'matchInfo' contains the last node found during the search. This
//     node's middleware, if any, is then combined with middleware from the tree using 'findMils'.
//  8. It returns the 'matchInfo' struct with the matched node and determined middleware, along with true to indicate a
//     successful route find.
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	// Attempt to retrieve the routing sub-tree for the provided HTTP method.
	root, ok := r.trees[method]
	if !ok {
		// If no such sub-tree exists, no routes are registered for this method; thus, no match is found.
		return nil, false
	}

	if path == "/" {
		// If the path is simply "/", return the root node of the method-specific sub-tree, including any middleware.
		return &matchInfo{n: root, mils: root.mils}, true
	}

	// Parse the path into segments for iterative matching against the routing tree nodes.
	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi := &matchInfo{} // Initialize an empty matchInfo struct.

	// Iterate over the path segments from the root node down the tree.
	for _, s := range segs {
		var child *node // Placeholder for the matched child node.
		child, ok = root.childOf(s)
		if !ok {
			// If no child node matches and the current node is not a wildcard, the route does not exist.
			return nil, false
		}
		// If a parameterized path segment is matched, add the parameter value to the matchInfo struct.
		if child.paramName != "" {
			mi.addValue(child.paramName, s)
		}
		// Shift the search context to the child node for the next iteration.
		root = child
	}

	// Once all segments are processed, record the matched node in the matchInfo.
	mi.n = root
	// Retrieve and append all relevant middleware to the matchInfo from the matched route.
	mi.mils = r.findMils(root, segs)

	// Return the matchInfo and true indicating the route was successfully found.
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

// node is a struct that represents a segment in the routing tree of a web server's router. Each node can be considered
// as part of the route's hierarchical path, with potential branching for nested routes. The node struct captures
// various details about a route segment, including its type, patterns, handlers, and any middleware that must be
// applied.

// Fields:
// - typ:       Holds the type of the node (e.g., static, parameterized, any) as defined by nodeType.
// - route:     The route pattern that the node represents, it is useful for debugging and route listing.
// - path:      Contains the explicit path segment associated with this node in the routing tree. For parameterized
//              nodes, this will be the parameter identifier (e.g., ":id").
// - children:  A map of child nodes, keyed by string, that allows the hierarchical structure of route paths. The keys
//              are segments from the path that lead to the respective child nodes.
// - handler:   The HandleFunc type handler that should be invoked when a route is matched to this node. It is
//              responsible for processing the request and providing a response.
// - starChild: A special pointer to a "star" or wildcard child node, which is used to implement wildcard routing,
//              matching any sequence of path segments.
// - paramChild: A pointer to a child node that represents a named parameter within the route segment. It allows the
//               router to handle dynamic URLs where segments can vary and are captured as parameters.
// - paramName: The name of the parameter (without the leading ":") when the node type is nodeTypeParam. This name is
//              used to retrieve the parameter value from the URL during routing.
// - mils:      A slice of Middleware that should be applied to the request if this node is part of the matched route.
//              Middleware functions are executed in the order in which they appear in the slice.
// - matchedMils: Stores any matched middlewares that have been applied during the route matching process. This can be
//                used to execute middleware in the correct order once a route match is confirmed.
// - regChild:  A pointer to a child node that should be analyzed using regular expressions. It expands the routing
//              capabilities to allow complex pattern matching as defined by the regExpr field.
// - regExpr:   When the node type is nodeTypeReg, this field holds the compiled regular expression object used to
//              match route segments against the registered patterns.

// The node struct is fundamental within the router for building a route matching system that accounts for fixed,
// dynamic, and wildcard routes. It allows the router to handle HTTP requests intelligently, mapping URLs to the
// appropriate handlers and ensuring middleware is invoked appropriately.
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
}

// findMils is a method that traverses the routing tree starting from a given root node and a slice of string segments
// representing a path. The method collects and returns all the middleware functions relevant to the path. Middleware
// is collected in the order encountered during traversal, which is depth-first and based on the path segments provided.

// Parameters:
// - root: The starting node representing the current context in the routing tree from which the search begins.
// - segs: A slice of strings representing individual segments of the request path that will guide the traversal
//         through the tree.

// The method uses a breadth-first algorithm to explore the tree:

// 1. It initializes a queue with the root node.
// 2. Iteratively processes each segment in the 'segs' slice:
//    a. The method dequeues nodes from the current level and processes them in a first-in, first-out (FIFO) manner.
//    b. For each node, if it has middleware attached (mils field is not empty), those middleware functions are
//       appended to the resulting slice 'res'.
//    c. It then obtains the children of the current node that correspond to the current path segment and adds them
//       to the list of nodes to be processed in the next iteration.
//
// 3. Once all segments are processed, the queue will have nodes that correspond to the final segment in 'segs'.
//    The middleware attached to these nodes is appended to the resulting slice, ensuring that middleware associated
//    with more specific paths is included.

// The findMils method returns a slice of Middleware functions that should be executed for the given path. This allows
// the router to apply all necessary middleware to the request before handling it with a route-specific handler.

// Note that the returned Middleware slice does not eliminate duplicates - if a middleware function appears in multiple
// nodes along the route, it will be included multiple times in the returned slice.

func (r *router) findMils(root *node, segs []string) []Middleware {
	// Initialize the queue with the root node and prepare a slice to store the resulting middleware.
	queue := []*node{root}
	res := make([]Middleware, 0, 16) // Allocate with a capacity of 16 to reduce slice-growing operations.

	// Process each segment in the URL path.
	for i := 0; i < len(segs); i++ {
		seg := segs[i]       // Current segment of the path.
		var children []*node // Temporary slice to hold next level of nodes.

		// Dequeue and process nodes to collect their middleware and find the next level of relevant nodes.
		for _, cur := range queue {
			if len(cur.mils) > 0 {
				// Append the middleware of the current node to the result.
				res = append(res, cur.mils...)
			}
			// Add children that correspond to the current path segment for future processing.
			children = append(children, cur.childrenOf(seg)...)
		}
		// Update the queue with the nodes for the next level.
		queue = children
	}

	// Collect middleware from the final set of nodes.
	for _, cur := range queue {
		if len(cur.mils) > 0 {
			res = append(res, cur.mils...)
		}
	}

	// Return the complete list of Middleware relevant to the path.
	return res
}

// childrenOf is a method that takes a path segment as input and returns a slice of nodes that represent the children
// of the current node (`n`) which correspond to the given path segment. This method is typically used in the process
// of walking the routing tree to find all the nodes that match a specific segment of a route path. The children are
// returned in a specific order, with any wildcard (star) child nodes first, followed by parameterized (param) child
// nodes, and finally the exact match (static) child node, if such a node exists.

// Parameters:
// - path: The path segment string used to find matching children of the current node.

// The method proceeds as follows:

// 1. Initialize an empty slice of node pointers with an initial capacity of 4, since it is uncommon for there to be
//    more than four possible matches (star, param, regexp, and static).
// 2. Checks if the current node has any children nodes that match the given path segment exactly. This would be an
//    exact (static) match and would indicate the next node in the path for a static route.
// 3. If there is a wildcard (star) child associated with the current node, which matches any segment, it is added to
//    the results slice. Wildcard child nodes can be used to match various path patterns and are useful for catch-all
//    route handling.
// 4. If the current node has a parameterized (param) child, it is considered a match since param children
//    can match any segment and represent a variable part of the path. The param child is appended to the slice.
// 5. The exact (static) match, if one was found, is appended to the slice. This ensures that more specific static
//    routes are evaluated after any dynamic routes like wildcard or param routes.

// The order in which the child nodes are added to the result is important, as it dictates the priority of the route
// matching. The method returns a slice of nodes that represent the combined possible matches for the given path
// segment in the context of the current node in the routing tree.

func (n *node) childrenOf(path string) []*node {
	// Initialize an empty slice with a small initial capacity for potential child nodes.
	res := make([]*node, 0, 4)

	// Look for a static child node that matches the path segment exactly.
	var static *node // Placeholder for the static child node.
	if n.children != nil {
		static = n.children[path]
	}

	// If a wildcard child exists, add it to the result slice. The wildcard child is used for catch-all routing paths.
	if n.starChild != nil {
		res = append(res, n.starChild)
	}

	// If a parameterized child node exists, it matches any path segment by definition and is added to the result.
	if n.paramChild != nil {
		res = append(res, n.paramChild)
	}

	// Finally, if a static child node was found, add it to the result slice.
	if static != nil {
		res = append(res, static)
	}

	// Return the slice of child nodes that can potentially match the provided path segment.
	return res
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {
	if n.children == nil {
		return n.childOfNonStatic(path)
	}
	res, ok := n.children[path]
	if !ok {
		return n.childOfNonStatic(path)
	}
	return res, ok
}

// childOfNonStatic 从非静态匹配的子节点里面查找
func (n *node) childOfNonStatic(path string) (*node, bool) {
	if n.regChild != nil {
		if n.regChild.regExpr.Match([]byte(path)) {
			return n.regChild, true
		}
	}
	if n.paramChild != nil {
		return n.paramChild, true
	}
	return n.starChild, n.starChild != nil
}

// childOrCreate 查找子节点，
func (n *node) childOrCreate(path string) *node {
	// 判断 path 是不是通配符路径
	if path == "*" {
		if n.paramChild != nil {
			panic(errs.ErrPathNotAllowWildcardAndPath(path))
		}
		if n.regChild != nil {
			panic(errs.ErrRegularNotAllowWildcardAndRegular(path))
		}
		if n.starChild == nil {
			n.starChild = &node{path: path, typ: nodeTypeAny}
		}
		return n.starChild
	}

	// 判断 path 是不是参数路径，即以 : 开头的路径，需要进一步解析，判断是参数路由还是正则路由
	if path[0] == ':' {
		paramName, expr, isReg := n.parseParam(path)
		if isReg {
			return n.childOrCreateReg(path, expr, paramName)
		}
		return n.childOrCreateParam(path, paramName)
	}

	if n.children == nil {
		n.children = make(map[string]*node)
	}
	// 从 children 里面查找
	child, ok := n.children[path]
	if !ok {
		// 如果没有找到，创建一个新的节点，并且保存在 node 里面
		child = &node{path: path, typ: nodeTypeStatic}
		n.children[path] = child
	}
	return child
}

func (n *node) childOrCreateParam(path string, paramName string) *node {
	if n.regChild != nil {
		panic(errs.ErrRegularNotAllowRegularAndPath(path))
	}
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndPath(path))
	}
	if n.paramChild != nil {
		if n.paramChild.path != path {
			panic(errs.ErrPathClash(n.paramChild.path, path))
		}
	} else {
		n.paramChild = &node{path: path, paramName: paramName, typ: nodeTypeParam}
	}
	return n.paramChild
}

func (n *node) childOrCreateReg(path string, expr string, paramName string) *node {
	if n.starChild != nil {
		panic(errs.ErrWildcardNotAllowWildcardAndRegular(path))
	}
	if n.paramChild != nil {
		panic(errs.ErrPathNotAllowPathAndRegular(path))
	}
	if n.regChild != nil {
		if n.regChild.regExpr.String() != expr || n.paramName != paramName {
			panic(errs.ErrRegularClash(n.regChild.path, path))
		}
	} else {
		regExpr, err := regexp.Compile(expr)
		if err != nil {
			panic(errs.ErrRegularExpression(err))
		}
		n.regChild = &node{path: path, paramName: paramName, regExpr: regExpr, typ: nodeTypeReg}
	}
	return n.regChild
}

// parseParam 用于解析判断是不是正则表达式
// 第一个返回值是参数名字
// 第二个返回值是正则表达式
// 第三个返回值为 true 则说明是正则路由
func (n *node) parseParam(path string) (string, string, bool) {
	// 去除 :
	path = path[1:]
	segs := strings.SplitN(path, "(", 2)
	if len(segs) == 2 {
		expr := segs[1]
		if strings.HasSuffix(expr, ")") {
			return segs[0], expr[:len(expr)-1], true
		}
	}
	return path, "", false
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
	mils       []Middleware
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}
