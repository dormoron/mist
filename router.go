package mist

import (
	"fmt"
	"github.com/dormoron/mist/internal/errs"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	trees map[string]*node
}

func initRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

func (r *router) registerRoute(method string, path string, handler HandleFunc, ms ...Middleware) {
	if path == "" {
		panic("web: 路由是空字符串")
	}
	if path[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}

	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}

	root, ok := r.trees[method]
	// 这是一个全新的 HTTP 方法，创建根节点
	if !ok {
		// 创建根节点
		root = &node{path: "/"}
		r.trees[method] = root
	}
	if path == "/" {
		if root.handler != nil {
			panic("web: 路由冲突[/]")
		}
		root.handler = handler
		root.route = "/"
		root.mils = ms
		return
	}

	segs := strings.Split(path[1:], "/")
	// 开始一段段处理
	for _, s := range segs {
		if s == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path))
		}
		root = root.childOrCreate(s)
	}
	if root.handler != nil {
		panic(fmt.Sprintf("web: 路由冲突[%s]", path))
	}
	root.handler = handler
	root.route = path
	root.mils = ms
}

// findRoute 查找对应的节点
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		return &matchInfo{n: root, mils: root.mils}, true
	}

	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi := &matchInfo{}
	for _, s := range segs {
		var child *node
		child, ok = root.childOf(s)
		if !ok {
			if root.typ == nodeTypeAny {
				mi.n = root
				return mi, true
			}
			return nil, false
		}
		if child.paramName != "" {
			mi.addValue(child.paramName, s)
		}
		root = child
	}
	mi.n = root
	mi.mils = r.findMils(root, segs)
	return mi, true
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
type node struct {
	typ nodeType

	route string

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	mils []Middleware

	matchedMils []Middleware

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

func (r *router) findMils(root *node, segs []string) []Middleware {
	queue := []*node{root}
	res := make([]Middleware, 0, 16)
	for i := 0; i < len(segs); i++ {
		seg := segs[i]
		var children []*node
		for _, cur := range queue {
			if len(cur.mils) > 0 {
				res = append(res, cur.mils...)
			}
			children = append(children, cur.childrenOf(seg)...)
		}
		queue = children
	}

	for _, cur := range queue {
		if len(cur.mils) > 0 {
			res = append(res, cur.mils...)
		}
	}
	return res
}

func (n *node) childrenOf(path string) []*node {
	res := make([]*node, 0, 4)
	var static *node
	if n.children != nil {
		static = n.children[path]
	}
	if n.starChild != nil {
		res = append(res, n.starChild)
	}
	if n.paramChild != nil {
		res = append(res, n.paramChild)
	}
	if static != nil {
		res = append(res, static)
	}
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
