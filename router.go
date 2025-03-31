package mist

import (
	"strings"
	"sync"

	"github.com/dormoron/mist/internal/errs"
)

// routeKey 是路由缓存的键结构
type routeKey struct {
	method string
	path   string
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

	// 路由缓存相关字段
	routeCache    map[routeKey]*matchInfo
	routeCacheMux sync.RWMutex
	maxCacheSize  int
	cacheHits     uint64
	cacheMisses   uint64
	cacheEnabled  bool
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
		trees:        map[string]*node{},
		routeCache:   make(map[routeKey]*matchInfo, 1024),
		maxCacheSize: 10000, // 默认缓存大小
		cacheEnabled: true,  // 默认启用缓存
	}
}

// EnableCache 启用路由缓存
func (r *router) EnableCache(maxSize int) {
	r.routeCacheMux.Lock()
	defer r.routeCacheMux.Unlock()

	r.cacheEnabled = true
	if maxSize > 0 {
		r.maxCacheSize = maxSize
	}
}

// DisableCache 禁用路由缓存
func (r *router) DisableCache() {
	r.routeCacheMux.Lock()
	defer r.routeCacheMux.Unlock()

	r.cacheEnabled = false
	r.clearCache()
}

// ClearCache 清空路由缓存
func (r *router) clearCache() {
	r.routeCache = make(map[routeKey]*matchInfo, 1024)
	r.cacheHits = 0
	r.cacheMisses = 0
}

// CacheStats 返回缓存统计信息
func (r *router) CacheStats() (hits, misses uint64, size int) {
	r.routeCacheMux.RLock()
	defer r.routeCacheMux.RUnlock()

	return r.cacheHits, r.cacheMisses, len(r.routeCache)
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

// registerRoute registers a new route with its handler and middleware in the router's routing tree. It
// validates the path format, ensures it's properly structured, and may panic if there's a conflict.
//
// This method is called by the various HTTP method-specific functions like GET, POST, and is
// internally used to set up routes with their respective handlers and middleware.
//
// Parameters:
//   - method: The HTTP method (e.g., GET, POST, PUT, etc.) that the route should respond to.
//   - path: The URL pattern to match against incoming requests. It can include parameters marked
//     with a colon ':' (e.g., '/users/:id') or wildcards '*'.
//   - handleFunc: The function to be called when the route is matched. This function handles the
//     HTTP request and generates a response. It can be nil if you're only attaching middleware.
//   - mils: A variadic parameter of Middleware functions to be applied to the route. These are
//     executed in the order they are provided, before the main handler function.
//
// This function internally:
//   - Validates the path starts with a '/' and doesn't contain unnecessary trailing slashes.
//   - Adds the route and its handler to the router's internal routing tree.
//   - Associates the provided middleware with the route.
//
// Note:
// If you want to add middleware to all routes under a certain path, consider using the Group
// functionality or the Use method instead.
func (r *router) registerRoute(method string, path string, handler HandleFunc, ms ...Middleware) {
	if path == "" {
		panic(errs.ErrRouterNotString())
	}
	if path[0] != '/' {
		panic(errs.ErrRouterFront())
	}

	// 检查路径尾部是否有多余的斜杠
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}

	// 如果路径中包含双斜杠，则抛出异常
	if strings.Contains(path, "//") {
		panic(errs.ErrRouterNotSymbolic(path))
	}

	// 获取或创建HTTP方法对应的根节点
	root, ok := r.trees[method]
	if !ok {
		root = &node{
			path:     "/",
			children: make(map[string]*node),
		}
		r.trees[method] = root
	}

	// 处理根路径 "/"
	if path == "/" {
		if root.handler != nil {
			panic(errs.ErrRouterConflict("/"))
		}
		// 给根节点分配处理函数和中间件
		root.handler = handler
		root.route = "/"
		root.mils = ms
		return
	}

	// 处理路径中的每个段来构建路由树中的节点
	segs := strings.Split(path[1:], "/") // 去掉前导'/'并按'/'分割路径
	for _, s := range segs {
		// 每个路径段必须是有效的字符串
		if s == "" {
			panic(errs.ErrRouterNotSymbolic(path))
		}
		// 创建或获取每个段的子节点，不断更新root指向最新节点
		root = root.childOrCreate(s)
	}

	// 在最终段节点设置路由处理函数和中间件，避免冲突
	if root.handler != nil {
		// 如果节点已经有处理函数，抛出异常避免覆盖现有路由
		panic(errs.ErrRouterConflict(path))
	}

	// 为路径序列中的最终节点设置处理函数和中间件，注册路由
	root.handler = handler
	root.route = path
	root.mils = ms
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
	// 检查路由缓存
	if r.cacheEnabled {
		r.routeCacheMux.RLock()
		key := routeKey{method: method, path: path}
		if mi, ok := r.routeCache[key]; ok {
			r.cacheHits++
			r.routeCacheMux.RUnlock()
			return mi, true
		}
		r.cacheMisses++
		r.routeCacheMux.RUnlock()
	}

	// Attempt to retrieve the root node for the HTTP method from the router's trees.
	root, ok := r.trees[method]
	// If the method does not have a corresponding tree, return no match.
	if !ok {
		return nil, false
	}

	// Special case for root path "/".
	if path == "/" {
		mi := &matchInfo{n: root, mils: root.mils}
		// 添加到缓存
		r.addToCache(method, path, mi)
		// Return the root node's information with associated middleware, indicating a match is found.
		return mi, true
	}

	// 优化：直接使用二级缓存进行静态路径快速匹配
	if child, found := r.tryFastMatch(root, path); found {
		mi := &matchInfo{n: child, mils: r.findMils(root, strings.Split(strings.Trim(path, "/"), "/"))}
		// 添加到缓存
		r.addToCache(method, path, mi)
		return mi, true
	}

	// Split the path into segments for traversal, ignoring any trailing slashes.
	segs := strings.Split(strings.Trim(path, "/"), "/")

	// Initialize matchInfo to store the matching route's info as we traverse.
	mi := &matchInfo{
		pathParams: make(map[string]string),
	}

	// Start from the root node.
	cur := root

	// Loop through the path segments to traverse the routing tree.
	for _, s := range segs {
		var found bool

		// 1. 尝试静态匹配 - 优先级最高
		if child, ok := cur.children[s]; ok {
			cur = child
			found = true
		}

		// 2. 尝试正则表达式匹配 - 次优先级
		if !found && cur.regChild != nil && cur.regChild.regExpr != nil {
			if cur.regChild.regExpr.Match([]byte(s)) {
				cur = cur.regChild
				found = true
				// 添加参数值
				mi.addValue(cur.paramName, s)
			}
		}

		// 3. 尝试参数匹配 - 再次优先级
		if !found && cur.paramChild != nil {
			cur = cur.paramChild
			found = true
			// 添加参数值
			mi.addValue(cur.paramName, s)
		}

		// 4. 尝试通配符匹配 - 最低优先级
		if !found && cur.starChild != nil {
			cur = cur.starChild
			found = true
			// 通配符特殊处理 - 收集剩余路径
			if cur.paramName != "" {
				// 如果是最后一段，直接使用当前段
				if len(segs) == 1 {
					mi.addValue(cur.paramName, s)
				} else {
					// 否则，收集当前及后续所有段
					remainingPath := strings.Join(segs[len(segs)-1:], "/")
					mi.addValue(cur.paramName, remainingPath)
					// 由于已经处理了所有剩余段，可以直接退出循环
					break
				}
			}
		}

		// 如果没有找到匹配项，返回失败
		if !found {
			return &matchInfo{}, false
		}
	}

	// 确保找到的节点有处理函数
	if cur == nil || cur.handler == nil {
		return &matchInfo{}, false
	}

	// Having traversed all segments, assign the last node and collected middleware to `mi`.
	mi.n = cur
	mi.mils = r.findMils(root, segs)

	// 添加到缓存
	r.addToCache(method, path, mi)

	// Return the populated matchInfo indicating a successful match.
	return mi, true
}

// tryFastMatch 尝试进行静态路径的快速匹配
// 对于静态路径，不需要进行完整的树遍历，可以直接通过全路径快速查找
func (r *router) tryFastMatch(root *node, path string) (*node, bool) {
	// 这里可以实现一个静态路径的快速查找表
	// 简化版本：仅对于完全静态的路径使用
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}

	// 循环检查是否有完全匹配的子节点
	cur := root
	if cur == nil {
		return nil, false
	}

	if path == "/" {
		return cur, cur.handler != nil
	}

	// 去掉前导斜杠
	if path[0] == '/' {
		path = path[1:]
	}

	segments := strings.Split(path, "/")

	for _, seg := range segments {
		if cur.children == nil {
			return nil, false
		}

		child, ok := cur.children[seg]
		if !ok {
			// 如果没有找到静态子节点，说明可能需要参数匹配
			return nil, false
		}

		cur = child
	}

	// 找到节点后检查是否有处理函数
	return cur, cur.handler != nil
}

// addToCache 将路由匹配结果添加到缓存
func (r *router) addToCache(method, path string, mi *matchInfo) {
	if !r.cacheEnabled {
		return
	}

	r.routeCacheMux.Lock()
	defer r.routeCacheMux.Unlock()

	// 检查缓存大小
	if len(r.routeCache) >= r.maxCacheSize {
		// 简单的缓存淘汰策略：当达到最大缓存大小时清空缓存
		r.clearCache()
	}

	key := routeKey{method: method, path: path}
	r.routeCache[key] = mi
}

// findMils 在路由树中查找与路径段关联的所有中间件
// 这个方法需要遍历整个路径，收集每个匹配节点关联的中间件
func (r *router) findMils(root *node, segs []string) []Middleware {
	var mils []Middleware

	// 首先添加根节点的中间件
	if len(root.mils) > 0 {
		mils = append(mils, root.mils...)
	}

	// 开始在当前节点
	cur := root

	// 遍历路径段，层层查找中间件
	for _, seg := range segs {
		// 检查静态匹配
		if child, ok := cur.children[seg]; ok {
			cur = child
		} else if cur.regChild != nil && cur.regChild.regExpr.Match([]byte(seg)) {
			// 正则匹配
			cur = cur.regChild
		} else if cur.paramChild != nil {
			// 参数匹配
			cur = cur.paramChild
		} else if cur.starChild != nil {
			// 通配符匹配
			cur = cur.starChild
			break // 通配符匹配后停止遍历
		} else {
			// 没有找到匹配节点，返回已收集的中间件
			return mils
		}

		// 添加当前节点的中间件
		if len(cur.mils) > 0 {
			mils = append(mils, cur.mils...)
		}
	}

	return mils
}
