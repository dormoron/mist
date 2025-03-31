package mist

import (
	"regexp"
	"strings"
	"sync"
)

// trieNode 表示前缀树中的一个节点
type trieNode struct {
	// 节点类型
	nodeType int

	// 节点路径片段
	segment string

	// 处理函数
	handler HandleFunc

	// 中间件
	middleware []Middleware

	// 子节点 - 静态路径
	children map[string]*trieNode

	// 参数子节点 - 如 :id
	paramChild *trieNode

	// 参数名
	paramName string

	// 通配符子节点 - 如 *filepath
	wildcardChild *trieNode

	// 正则表达式子节点 - 如 {id:[0-9]+}
	regexChild *trieNode

	// 正则表达式
	regex *regexp.Regexp

	// 完整路径
	fullPath string

	// 父节点
	parent *trieNode
}

// PathTrie 是一个高性能的路由前缀树实现
type PathTrie struct {
	// 方法树映射
	trees map[string]*trieNode

	// 路由缓存
	cache      map[string]*matchInfo
	cacheMutex sync.RWMutex

	// 路由缓存配置
	maxCacheSize int
	cacheEnabled bool
	cacheHits    uint64
	cacheMisses  uint64
}

// 节点类型常量
const (
	nodeTypeStatic2   = iota // 静态节点，如 /users
	nodeTypeParam2           // 参数节点，如 /:id
	nodeTypeWildcard2        // 通配符节点，如 /*filepath
	nodeTypeRegex2           // 正则表达式节点，如 /{id:[0-9]+}
)

// NewPathTrie 创建一个新的PathTrie实例
func NewPathTrie() *PathTrie {
	return &PathTrie{
		trees:        make(map[string]*trieNode),
		cache:        make(map[string]*matchInfo, 1024),
		maxCacheSize: 10000,
		cacheEnabled: true,
	}
}

// Add 向前缀树添加路由
func (t *PathTrie) Add(method, path string, handler HandleFunc, middleware ...Middleware) {
	// 输入验证
	if path == "" {
		panic("路径不能为空")
	}
	if path[0] != '/' {
		panic("路径必须以'/'开头")
	}
	if path != "/" && path[len(path)-1] == '/' {
		panic("路径不能以'/'结尾")
	}

	// 清空缓存
	if t.cacheEnabled {
		t.cacheMutex.Lock()
		t.cache = make(map[string]*matchInfo, 1024)
		t.cacheHits = 0
		t.cacheMisses = 0
		t.cacheMutex.Unlock()
	}

	// 获取或创建对应HTTP方法的根节点
	root, ok := t.trees[method]
	if !ok {
		root = &trieNode{
			nodeType: nodeTypeStatic2,
			segment:  "/",
			children: make(map[string]*trieNode),
			fullPath: "/",
		}
		t.trees[method] = root
	}

	// 根路径特殊处理
	if path == "/" {
		if root.handler != nil {
			panic("路由冲突：根路径已注册")
		}
		root.handler = handler
		root.middleware = middleware
		return
	}

	// 分割路径
	segments := strings.Split(path[1:], "/")
	currentNode := root

	// 遍历路径段
	for _, segment := range segments {
		if segment == "" {
			panic("路径不能包含连续的'/'")
		}

		// 检查是否是参数路径
		if segment[0] == ':' {
			paramName := segment[1:]
			if paramName == "" {
				panic("参数名不能为空")
			}

			// 创建或获取参数子节点
			if currentNode.paramChild == nil {
				currentNode.paramChild = &trieNode{
					nodeType:  nodeTypeParam2,
					segment:   segment,
					paramName: paramName,
					children:  make(map[string]*trieNode),
					parent:    currentNode,
				}
			}
			currentNode = currentNode.paramChild
		} else if segment[0] == '*' {
			// 通配符路径
			wildcardName := segment[1:]
			if wildcardName == "" {
				panic("通配符名不能为空")
			}

			// 创建或获取通配符子节点
			if currentNode.wildcardChild == nil {
				currentNode.wildcardChild = &trieNode{
					nodeType:  nodeTypeWildcard2,
					segment:   segment,
					paramName: wildcardName,
					children:  make(map[string]*trieNode),
					parent:    currentNode,
				}
			}
			currentNode = currentNode.wildcardChild
		} else if len(segment) > 3 && segment[0] == '{' && segment[len(segment)-1] == '}' {
			// 正则表达式路径
			// 例如：/{id:[0-9]+}
			parts := strings.SplitN(segment[1:len(segment)-1], ":", 2)
			if len(parts) != 2 {
				panic("正则表达式路径格式不正确")
			}

			paramName := parts[0]
			regexStr := parts[1]

			// 编译正则表达式
			regex, err := regexp.Compile(regexStr)
			if err != nil {
				panic("正则表达式编译失败: " + err.Error())
			}

			// 创建或获取正则子节点
			if currentNode.regexChild == nil {
				currentNode.regexChild = &trieNode{
					nodeType:  nodeTypeRegex2,
					segment:   segment,
					paramName: paramName,
					regex:     regex,
					children:  make(map[string]*trieNode),
					parent:    currentNode,
				}
			}
			currentNode = currentNode.regexChild
		} else {
			// 静态路径
			child, ok := currentNode.children[segment]
			if !ok {
				child = &trieNode{
					nodeType: nodeTypeStatic2,
					segment:  segment,
					children: make(map[string]*trieNode),
					parent:   currentNode,
				}
				currentNode.children[segment] = child
			}
			currentNode = child
		}
	}

	// 设置当前节点的完整路径
	currentNode.fullPath = path

	// 检查是否存在路由冲突
	if currentNode.handler != nil {
		panic("路由冲突：路径已注册 " + path)
	}

	// 设置处理函数和中间件
	currentNode.handler = handler
	currentNode.middleware = middleware
}

// Match 根据方法和路径查找匹配的路由
func (t *PathTrie) Match(method, path string) (*matchInfo, bool) {
	// 检查缓存
	cacheKey := method + ":" + path
	if t.cacheEnabled {
		t.cacheMutex.RLock()
		if mi, ok := t.cache[cacheKey]; ok {
			t.cacheHits++
			t.cacheMutex.RUnlock()
			return mi, true
		}
		t.cacheMisses++
		t.cacheMutex.RUnlock()
	}

	// 获取对应方法的根节点
	root, ok := t.trees[method]
	if !ok {
		return nil, false
	}

	// 根路径特殊处理
	if path == "/" {
		if root.handler == nil {
			return nil, false
		}
		mi := &matchInfo{
			n:          (*node)(nil), // 由于兼容性，设为nil
			mils:       root.middleware,
			pathParams: make(map[string]string),
		}
		t.addToCache(cacheKey, mi)
		return mi, true
	}

	// 处理路径
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")
	params := make(map[string]string)

	// 匹配路径
	matchedNode, matched := t.matchNode(root, pathSegments, params, 0)
	if !matched {
		return nil, false
	}

	// 组装匹配信息
	mi := &matchInfo{
		n:          (*node)(nil), // 由于兼容性，设为nil
		mils:       collectMiddleware(matchedNode),
		pathParams: params,
	}

	// 添加到缓存
	t.addToCache(cacheKey, mi)

	return mi, matchedNode.handler != nil
}

// matchNode 递归匹配路径段
func (t *PathTrie) matchNode(currentNode *trieNode, segments []string, params map[string]string, index int) (*trieNode, bool) {
	// 已经匹配完所有段
	if index >= len(segments) {
		return currentNode, true
	}

	segment := segments[index]

	// 1. 尝试静态匹配
	if child, ok := currentNode.children[segment]; ok {
		if matchedNode, matched := t.matchNode(child, segments, params, index+1); matched {
			return matchedNode, true
		}
	}

	// 2. 尝试参数匹配
	if currentNode.paramChild != nil {
		// 存储参数
		params[currentNode.paramChild.paramName] = segment
		if matchedNode, matched := t.matchNode(currentNode.paramChild, segments, params, index+1); matched {
			return matchedNode, true
		}
		// 回溯，移除参数
		delete(params, currentNode.paramChild.paramName)
	}

	// 3. 尝试正则匹配
	if currentNode.regexChild != nil && currentNode.regexChild.regex.MatchString(segment) {
		// 存储参数
		params[currentNode.regexChild.paramName] = segment
		if matchedNode, matched := t.matchNode(currentNode.regexChild, segments, params, index+1); matched {
			return matchedNode, true
		}
		// 回溯，移除参数
		delete(params, currentNode.regexChild.paramName)
	}

	// 4. 尝试通配符匹配
	if currentNode.wildcardChild != nil {
		// 处理通配符匹配
		// 将剩余的所有路径段合并作为通配符参数的值
		remainingPath := strings.Join(segments[index:], "/")
		params[currentNode.wildcardChild.paramName] = remainingPath
		return currentNode.wildcardChild, true
	}

	// 没有匹配
	return nil, false
}

// collectMiddleware 收集中间件
func collectMiddleware(node *trieNode) []Middleware {
	if node == nil {
		return nil
	}

	// 沿着树向上收集所有中间件
	var middleware []Middleware
	current := node

	// 从叶子节点到根节点收集中间件
	var middlewareStack [][]Middleware
	for current != nil {
		if len(current.middleware) > 0 {
			middlewareStack = append(middlewareStack, current.middleware)
		}
		current = current.parent
	}

	// 按照从根到叶的顺序添加中间件
	for i := len(middlewareStack) - 1; i >= 0; i-- {
		middleware = append(middleware, middlewareStack[i]...)
	}

	return middleware
}

// addToCache 添加匹配结果到缓存
func (t *PathTrie) addToCache(key string, mi *matchInfo) {
	if !t.cacheEnabled {
		return
	}

	t.cacheMutex.Lock()
	defer t.cacheMutex.Unlock()

	// 检查缓存大小
	if len(t.cache) >= t.maxCacheSize {
		// 简单的缓存淘汰策略：清空整个缓存
		t.cache = make(map[string]*matchInfo, 1024)
		t.cacheHits = 0
		t.cacheMisses = 0
	}

	t.cache[key] = mi
}

// EnableCache 启用缓存
func (t *PathTrie) EnableCache(maxSize int) {
	t.cacheMutex.Lock()
	defer t.cacheMutex.Unlock()

	t.cacheEnabled = true
	if maxSize > 0 {
		t.maxCacheSize = maxSize
	}
}

// DisableCache 禁用缓存
func (t *PathTrie) DisableCache() {
	t.cacheMutex.Lock()
	defer t.cacheMutex.Unlock()

	t.cacheEnabled = false
	t.cache = make(map[string]*matchInfo, 1024)
	t.cacheHits = 0
	t.cacheMisses = 0
}

// CacheStats 返回缓存统计信息
func (t *PathTrie) CacheStats() (hits, misses uint64, size int) {
	t.cacheMutex.RLock()
	defer t.cacheMutex.RUnlock()

	return t.cacheHits, t.cacheMisses, len(t.cache)
}
