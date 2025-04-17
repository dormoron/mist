package mist

import (
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist/internal/errs"
)

// routeKey 是路由缓存的键结构
type routeKey struct {
	method string
	path   string
}

// routeStats 路由统计信息
type routeStats struct {
	hits             int64         // 命中次数
	totalTime        time.Duration // 总处理时间
	maxTime          time.Duration // 最长处理时间
	minTime          time.Duration // 最短处理时间
	lastAccessTime   time.Time     // 最后访问时间
	creationTime     time.Time     // 创建时间
	lastResponseCode int           // 最后响应码
}

// router 是一个数据结构，用于存储和检索Web服务器或类似应用程序的路由信息，
// 这些应用程序需要URL路径分段和模式匹配。
// 此结构是路由机制的核心，负责将URL路径与其对应的处理程序关联起来。
//
// 'router'结构包含一个名为'trees'的map，其中键是HTTP方法，
// 如"GET"，"POST"，"PUT"等。对于每个HTTP方法，都有一个关联的'node'，
// 它是树形数据结构的根。这棵树用于以层次方式存储路由，
// 其中URL路径的每个部分都表示为树中的一个节点。
//
// 通过将URL路径分解为段并以树形结构组织它们，'router'
// 结构允许快速高效地将URL路径与它们各自的处理程序匹配，
// 即使随着路由数量的增长。这种组织方式改进了线性
// 搜索方法，将路由复杂度从O(n)降低到O(k)，其中k是URL中的路径段数。
//
// 因此，'trees' map是高效处理HTTP请求匹配和调度的关键组件，
// 支持动态URL模式并提高应用程序的性能和可扩展性。
//
// 'router'使用示例:
// - 创建新的router实例:
//
//	r := &router{
//	    trees: make(map[string]*node),
//	}
//
// - 向router添加路由:
//
//	r.addRoute("GET", "/home", homeHandler)
//	r.addRoute("POST", "/users", usersHandler)
//	...
//
// - 使用router将请求的方法和路径匹配到适当的处理程序:
//
//	handler, params := r.match("GET", "/home")
//	if handler != nil {
//	  handler.ServeHTTP(w, r)
//	}
//
// 注意事项:
//   - 为了最大效率，trees应该只在应用程序的初始化阶段或在没有请求处理时进行修改，
//     因为对树结构的修改可能会导致竞争条件或不一致的路由。
//   - 大多数路由树支持动态路由参数的扩展(如'/users/:userID')，
//     在设计节点及其匹配算法时应考虑到这一点。
//   - 错误处理，如检测重复路由、无效模式或不支持的HTTP方法，
//     应根据应用程序的需要考虑和实现。
type router struct {
	trees map[string]*node

	// 自适应路由缓存
	routeCache   *AdaptiveCache // 使用自适应缓存替代原有的sync.Map
	cacheEnabled bool

	// 路由统计
	routeStatsMu sync.RWMutex
	routeStats   map[string]*routeStats
	enableStats  bool
}

// initRouter 是一个工厂函数，初始化并返回'router'结构的新实例。
// 此函数通过设置其内部数据结构为路由器的使用做准备，这些数据结构对于
// 在Web应用程序中注册和匹配URL路由到其关联的处理程序是必要的。
//
// 当被调用时，它构造一个带有初始化的'trees' map的'router'实例，这对于
// 存储不同HTTP方法的路由树的根节点至关重要。'trees' map以HTTP方法
// 作为字符串键，如"GET", "POST", "PUT"等，而值是指向'node'结构实例的指针，
// 这些实例代表该HTTP方法的树的根。
//
// 由于路由逻辑需要为每个HTTP方法设置一个不同的树，以允许高效的路由匹配
// 并适应每种方法下可能存在的唯一路径，'trees' map初始化为空，
// 没有根节点。根节点通常是在通过单独的函数或方法向路由器注册新路由时添加的，
// 示例中未显示。
//
// 'initRouter'函数的使用通常在Web服务器或应用程序的设置阶段，
// 在开始服务请求之前建立路由。返回的'router'实例
// 准备好添加路由，随后用于根据路径和方法将传入的HTTP请求路由到正确的
// 处理程序。
//
// 使用示例:
// - 在应用程序启动时初始化新的路由器实例
//
//		r := initRouter()
//		r.addRoute("GET", "/", homeHandler)
//		r.addRoute("POST", "/users", usersHandler)
//		...
//		http.ListenAndServe(":8080", r)
//
//	  - 应用程序的主函数或设置函数通常包括对'initRouter'的调用，然后是路由
//	    注册代码，最终启动配置好的服务器。
//
// 注意事项:
//   - 重要的是，在路由器开始处理任何请求之前完成此初始化，以确保
//     线程安全。如果应用程序在开始服务请求后需要修改路由器，
//     应该采用适当的同步机制。
//   - 'initRouter'函数抽象了初始化细节，确保满足'router'结构的所有必需不变量，
//     通过集中路由器设置逻辑改进了代码可读性和安全性。
func initRouter() router {
	// 创建自适应缓存配置
	cacheConfig := AdaptiveCacheConfig{
		MaxSize:            10000,
		CleanupInterval:    5 * time.Minute,
		MinAccessCount:     5,
		AccessTimeWeight:   0.3,
		FrequencyWeight:    0.5,
		ResponseTimeWeight: 0.2,
		AdaptiveMode:       true,
	}

	return router{
		trees:        map[string]*node{},
		routeCache:   NewAdaptiveCache(cacheConfig),
		cacheEnabled: true,
		routeStats:   make(map[string]*routeStats),
		enableStats:  false,
	}
}

// EnableCache 启用路由缓存
func (r *router) EnableCache(maxSize int) {
	r.cacheEnabled = true

	// 创建新配置
	cacheConfig := AdaptiveCacheConfig{
		MaxSize:            int64(maxSize),
		CleanupInterval:    5 * time.Minute,
		MinAccessCount:     5,
		AccessTimeWeight:   0.3,
		FrequencyWeight:    0.5,
		ResponseTimeWeight: 0.2,
		AdaptiveMode:       true,
	}

	// 如果旧缓存存在，关闭它
	if r.routeCache != nil {
		r.routeCache.Close()
	}

	r.routeCache = NewAdaptiveCache(cacheConfig)
}

// DisableCache 禁用路由缓存
func (r *router) DisableCache() {
	r.cacheEnabled = false
	if r.routeCache != nil {
		r.routeCache.Disable()
	}
}

// CacheStats 返回缓存统计信息
func (r *router) CacheStats() (hits, misses uint64, size int) {
	if r.routeCache == nil {
		return 0, 0, 0
	}

	hits, misses, _, size64, _ := r.routeCache.Stats()
	return hits, misses, int(size64)
}

// Group 创建并返回一个附加到被调用路由器上的新routerGroup。
// 该方法通过检查前缀是否以正斜杠开始且不以正斜杠结束（除非是根组"/"）来确保提供的前缀符合格式要求。
// 如果前缀无效，此方法会panic，以防止路由器配置错误。
//
// 参数:
//   - prefix: 新路由组的路径前缀。它应该以正斜杠开始，
//     除了根组外，不应以正斜杠结束。
//   - ms: 零个或多个中间件函数，这些函数将应用于创建的routerGroup内的每个路由。
//
// 返回:
// *routerGroup: 指向新创建的routerGroup的指针，带有给定的前缀和中间件。
//
// Panic:
// 如果'prefix'不以'/'开始，或者如果'prefix'以'/'结束
// (除非'prefix'恰好是"/")，此方法将会panic。
//
// 使用示例:
// r := &router{} // 假设路由器已初始化
// group := r.Group("/api", loggingMiddleware, authMiddleware)
func (r *router) Group(prefix string, ms ...Middleware) *routerGroup {
	// 检查前缀是否为空或不以'/'开始
	if prefix == "" || prefix[0] != '/' {
		panic(errs.ErrRouterGroupFront()) // 使用预定义的错误panic，表示前缀起始不正确
	}
	// 检查前缀是否不是根'/'并以'/'结束
	if prefix != "/" && prefix[len(prefix)-1] == '/' {
		panic(errs.ErrRouterGroupBack()) // 使用预定义的错误panic，表示前缀结束不正确
	}
	// 如果前缀正确，初始化一个新的routerGroup并使用提供的详细信息返回
	return &routerGroup{prefix: prefix, router: r, middles: ms}
}

// registerRoute 在路由器的路由树中注册一个带有处理程序和中间件的新路由。它
// 验证路径格式，确保结构正确，如果有冲突可能会panic。
//
// 此方法由各种HTTP方法特定的函数（如GET、POST）调用，并在
// 内部用于设置路由及其各自的处理程序和中间件。
//
// 参数:
//   - method: 路由应响应的HTTP方法（例如，GET、POST、PUT等）。
//   - path: 匹配传入请求的URL模式。它可以包括用冒号':'标记的参数
//     （例如，'/users/:id'）或通配符'*'。
//   - handleFunc: 当路由匹配时要调用的函数。此函数处理
//     HTTP请求并生成响应。如果你只是附加中间件，它可以为nil。
//   - mils: 一个可变参数的中间件函数，应用于路由。这些函数
//     按照提供的顺序执行，在主处理函数之前。
//
// 此函数内部:
//   - 验证路径以'/'开始且不包含不必要的尾部斜杠。
//   - 将路由及其处理程序添加到路由器的内部路由树中。
//   - 将提供的中间件与路由关联。
//
// 注意:
// 如果你想为某个路径下的所有路由添加中间件，考虑使用Group
// 功能或Use方法。
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

// appendCollectMiddlewares 从给定节点向上遍历到根节点，收集所有
// 中间件，顺序从根到节点。此函数通常用于收集应用于
// 请求的所有中间件，当请求从根节点向下到特定路由时。
//
// 参数:
//   - n: 指向开始收集中间件的起始节点的指针。这通常代表
//     由特定路由匹配的节点。
//   - ms: 一个可能已经包含一些中间件的Middleware切片。来自节点的额外中间件将
//     被前置到这个切片中。
//
// 返回:
//   - []Middleware: 一个包含提供的中间件和从遍历树到根节点
//     收集的所有中间件的Middleware切片。
func appendCollectMiddlewares(n *node, ms []Middleware) []Middleware {
	// 初始化一个本地切片，从传入的中间件开始。
	// 这将允许按正确顺序收集中间件。
	middles := ms // 最终将包含收集的中间件的切片。

	// 从给定节点开始向上迭代，直到到达根节点。
	for n != nil {
		// 将当前节点的中间件前置到'middles'切片中。
		// 前置确保了中间件在以后应用时的正确顺序：从根向下到节点。
		middles = append(n.mils, middles...)
		// 向上爬到父节点进行下一次迭代。
		n = n.parent
	}
	// 返回从根到给定节点的树中收集的中间件。
	return middles
}

// findRoute 查找匹配的路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	startTime := time.Now()

	// 首先尝试从缓存获取
	mi, found := r.findRouteFromCache(method, path)
	if found {
		// 命中缓存
		return mi, true
	}

	// 缓存未命中，执行正常路由查找
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	// 先尝试快速匹配（静态路由）
	if path == "/" {
		return &matchInfo{
			n:          root,
			pathParams: nil,
			mils:       collectMiddlewares(root),
		}, true
	}

	// 执行常规匹配过程
	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi = &matchInfo{}
	mi.pathParams = make(map[string]string)

	// 使用递归查找匹配的节点
	n, found := r.matchNode(root, segs, mi.pathParams, 0)
	if !found {
		return nil, false
	}

	mi.n = n
	mi.mils = collectMiddlewares(n)

	// 匹配成功，添加到缓存
	responseTime := time.Since(startTime)
	r.addToCache(method, path, mi, responseTime)

	// 如果启用了统计，记录路由访问情况
	if r.enableStats {
		r.trackRouteStats(path, startTime, 200) // 假设状态码为200
	}

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

// findRouteFromCache 从缓存查找路由
func (r *router) findRouteFromCache(method, path string) (*matchInfo, bool) {
	if !r.cacheEnabled || r.routeCache == nil {
		return nil, false
	}

	cacheKey := method + ":" + path
	value, found := r.routeCache.Get(cacheKey)
	if !found {
		return nil, false
	}

	// 类型断言
	mi, ok := value.(*matchInfo)
	return mi, ok
}

// addToCache 添加路由到缓存
func (r *router) addToCache(method, path string, mi *matchInfo, responseTime time.Duration) {
	if !r.cacheEnabled || r.routeCache == nil {
		return
	}

	cacheKey := method + ":" + path
	r.routeCache.Set(cacheKey, mi, responseTime)
}

// matchNode 递归匹配节点
func (r *router) matchNode(currentNode *node, segments []string, params map[string]string, index int) (*node, bool) {
	// 如果已处理完所有段，检查是否有处理函数
	if index >= len(segments) {
		if currentNode.handler != nil {
			return currentNode, true
		}
		return nil, false
	}

	segment := segments[index]

	// 1. 尝试静态子节点匹配（优先级最高）
	if child, ok := currentNode.children[segment]; ok {
		if matchedNode, found := r.matchNode(child, segments, params, index+1); found {
			return matchedNode, true
		}
	}

	// 2. 尝试正则表达式子节点匹配
	if currentNode.regChild != nil && currentNode.regChild.regExpr != nil {
		if currentNode.regChild.regExpr.Match([]byte(segment)) {
			// 添加参数
			params[currentNode.regChild.paramName] = segment

			if matchedNode, found := r.matchNode(currentNode.regChild, segments, params, index+1); found {
				return matchedNode, true
			}
		}
	}

	// 3. 尝试参数子节点匹配
	if currentNode.paramChild != nil {
		// 添加参数
		params[currentNode.paramChild.paramName] = segment

		if matchedNode, found := r.matchNode(currentNode.paramChild, segments, params, index+1); found {
			return matchedNode, true
		}
	}

	// 4. 尝试通配符子节点匹配（优先级最低）
	if currentNode.starChild != nil {
		// 通配符特殊处理，收集剩余所有段
		if currentNode.starChild.paramName != "" {
			if index == len(segments)-1 {
				// 最后一段，直接使用
				params[currentNode.starChild.paramName] = segment
			} else {
				// 收集剩余所有段
				remaining := strings.Join(segments[index:], "/")
				params[currentNode.starChild.paramName] = remaining
			}
		}

		// 通配符匹配成功，如果有处理函数则返回
		if currentNode.starChild.handler != nil {
			return currentNode.starChild, true
		}
	}

	// 没有找到匹配
	return nil, false
}

// collectMiddlewares 收集中间件
func collectMiddlewares(n *node) []Middleware {
	middles := []Middleware{}

	// 收集当前节点的中间件
	middles = append(middles, n.mils...)

	// 递归收集父节点的中间件
	for n.parent != nil {
		n = n.parent
		middles = append(n.mils, middles...)
	}

	return middles
}

// EnableStats 启用路由统计
func (r *router) EnableStats() {
	r.enableStats = true
}

// DisableStats 禁用路由统计
func (r *router) DisableStats() {
	r.enableStats = false
}

// GetRouteStats 获取指定路由的统计信息
func (r *router) GetRouteStats(route string) (map[string]interface{}, bool) {
	r.routeStatsMu.RLock()
	defer r.routeStatsMu.RUnlock()

	stats, exists := r.routeStats[route]
	if !exists {
		return nil, false
	}

	var avgTime time.Duration
	if stats.hits > 0 {
		avgTime = time.Duration(int64(stats.totalTime) / stats.hits)
	}

	return map[string]interface{}{
		"hits":               stats.hits,
		"avg_time_ms":        avgTime.Milliseconds(),
		"max_time_ms":        stats.maxTime.Milliseconds(),
		"min_time_ms":        stats.minTime.Milliseconds(),
		"last_access_time":   stats.lastAccessTime,
		"creation_time":      stats.creationTime,
		"last_response_code": stats.lastResponseCode,
	}, true
}

// GetAllRouteStats 获取所有路由的统计信息
func (r *router) GetAllRouteStats() map[string]map[string]interface{} {
	r.routeStatsMu.RLock()
	defer r.routeStatsMu.RUnlock()

	result := make(map[string]map[string]interface{})

	for route, stats := range r.routeStats {
		var avgTime time.Duration
		if stats.hits > 0 {
			avgTime = time.Duration(int64(stats.totalTime) / stats.hits)
		}

		result[route] = map[string]interface{}{
			"hits":               stats.hits,
			"avg_time_ms":        avgTime.Milliseconds(),
			"max_time_ms":        stats.maxTime.Milliseconds(),
			"min_time_ms":        stats.minTime.Milliseconds(),
			"last_access_time":   stats.lastAccessTime,
			"creation_time":      stats.creationTime,
			"last_response_code": stats.lastResponseCode,
		}
	}

	return result
}

// ResetRouteStats 重置所有路由统计信息
func (r *router) ResetRouteStats() {
	r.routeStatsMu.Lock()
	defer r.routeStatsMu.Unlock()

	r.routeStats = make(map[string]*routeStats)
}

// trackRouteStats 记录路由统计信息
func (r *router) trackRouteStats(route string, start time.Time, statusCode int) {
	if !r.enableStats {
		return
	}

	elapsed := time.Since(start)

	r.routeStatsMu.Lock()
	defer r.routeStatsMu.Unlock()

	stats, exists := r.routeStats[route]
	if !exists {
		stats = &routeStats{
			hits:             1,
			totalTime:        elapsed,
			maxTime:          elapsed,
			minTime:          elapsed,
			creationTime:     time.Now(),
			lastAccessTime:   time.Now(),
			lastResponseCode: statusCode,
		}
		r.routeStats[route] = stats
	} else {
		stats.hits++
		stats.totalTime += elapsed
		stats.lastAccessTime = time.Now()
		stats.lastResponseCode = statusCode

		if elapsed > stats.maxTime {
			stats.maxTime = elapsed
		}

		if elapsed < stats.minTime {
			stats.minTime = elapsed
		}
	}
}
