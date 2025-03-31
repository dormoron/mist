// Package apidoc 提供了API文档生成工具
package apidoc

import (
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/dormoron/mist"
)

// RouteInfo 表示路由信息
type RouteInfo struct {
	// 请求方法
	Method string `json:"method"`
	// 路由路径
	Path string `json:"path"`
	// 路由说明
	Description string `json:"description,omitempty"`
	// 请求参数
	Params []ParamInfo `json:"params,omitempty"`
	// 请求体格式
	RequestBody interface{} `json:"request_body,omitempty"`
	// 响应格式
	ResponseBody interface{} `json:"response_body,omitempty"`
}

// ParamInfo 表示参数信息
type ParamInfo struct {
	// 参数名
	Name string `json:"name"`
	// 参数位置 (path/query/header)
	In string `json:"in"`
	// 参数类型
	Type string `json:"type"`
	// 是否必需
	Required bool `json:"required"`
	// 参数描述
	Description string `json:"description,omitempty"`
}

// GroupInfo 表示分组信息
type GroupInfo struct {
	// 分组名称
	Name string `json:"name"`
	// 分组前缀
	Prefix string `json:"prefix"`
	// 分组说明
	Description string `json:"description,omitempty"`
	// 分组下的路由
	Routes []RouteInfo `json:"routes"`
}

// APIDoc 表示API文档
type APIDoc struct {
	// API标题
	Title string `json:"title"`
	// API版本
	Version string `json:"version"`
	// API描述
	Description string `json:"description,omitempty"`
	// API分组
	Groups []GroupInfo `json:"groups,omitempty"`
	// API路由（不属于任何分组）
	Routes []RouteInfo `json:"routes,omitempty"`

	// 内部使用，收集路由信息
	routes []RouteInfo
}

// New 创建一个新的APIDoc实例
func New(title, version, description string) *APIDoc {
	return &APIDoc{
		Title:       title,
		Version:     version,
		Description: description,
		routes:      make([]RouteInfo, 0),
	}
}

// AddRoute 添加路由信息
func (doc *APIDoc) AddRoute(method, path, description string, params []ParamInfo, requestBody, responseBody interface{}) {
	doc.routes = append(doc.routes, RouteInfo{
		Method:       method,
		Path:         path,
		Description:  description,
		Params:       params,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
	})
}

// AddRouteInfo 添加路由信息
func (doc *APIDoc) AddRouteInfo(info RouteInfo) {
	doc.routes = append(doc.routes, info)
}

// Organize 整理API文档，根据路径前缀分组
func (doc *APIDoc) Organize() {
	// 按路径排序
	sort.Slice(doc.routes, func(i, j int) bool {
		return doc.routes[i].Path < doc.routes[j].Path
	})

	// 提取所有一级路径
	prefixMap := make(map[string][]RouteInfo)
	noGroupRoutes := make([]RouteInfo, 0)

	for _, route := range doc.routes {
		parts := strings.Split(strings.Trim(route.Path, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			prefix := "/" + parts[0]
			prefixMap[prefix] = append(prefixMap[prefix], route)
		} else {
			noGroupRoutes = append(noGroupRoutes, route)
		}
	}

	// 创建分组
	groups := make([]GroupInfo, 0)
	for prefix, routes := range prefixMap {
		groups = append(groups, GroupInfo{
			Name:   strings.Title(strings.TrimPrefix(prefix, "/")),
			Prefix: prefix,
			Routes: routes,
		})
	}

	// 按前缀排序
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Prefix < groups[j].Prefix
	})

	doc.Groups = groups
	doc.Routes = noGroupRoutes
}

// GenerateHandler 生成API文档处理函数
func (doc *APIDoc) GenerateHandler() mist.HandleFunc {
	doc.Organize()

	return func(ctx *mist.Context) {
		ctx.RespondWithJSON(http.StatusOK, doc)
	}
}

// ExtractRoutes 从HTTP服务器中提取路由信息（实验性功能）
// 注意：这个功能依赖于反射和内部结构，可能不稳定
func (doc *APIDoc) ExtractRoutes(server *mist.HTTPServer) {
	// 获取server中的router字段
	serverValue := reflect.ValueOf(server).Elem()
	routerValue := serverValue.FieldByName("router")

	if !routerValue.IsValid() {
		return
	}

	// 获取router中的trees字段
	treesValue := routerValue.FieldByName("trees")
	if !treesValue.IsValid() {
		return
	}

	// 遍历HTTP方法和对应的路由树
	treesIter := treesValue.MapRange()
	for treesIter.Next() {
		method := treesIter.Key().String()
		root := treesIter.Value().Interface()

		// 提取路由信息
		doc.extractRoutesFromNode(method, "", root)
	}
}

// extractRoutesFromNode 从节点中提取路由信息（实验性功能）
func (doc *APIDoc) extractRoutesFromNode(method, parentPath string, node interface{}) {
	// 获取节点的字段
	nodeValue := reflect.ValueOf(node).Elem()

	// 获取path字段
	pathValue := nodeValue.FieldByName("path")
	if !pathValue.IsValid() {
		return
	}

	path := pathValue.String()
	if path == "" {
		return
	}

	// 组合完整路径
	fullPath := parentPath
	if path != "/" || fullPath == "" {
		fullPath = parentPath + path
	}

	// 检查是否有处理函数
	handlerValue := nodeValue.FieldByName("handler")
	if handlerValue.IsValid() && !handlerValue.IsNil() {
		// 这是一个有效的路由
		doc.AddRoute(method, fullPath, "", nil, nil, nil)
	}

	// 遍历子节点
	childrenValue := nodeValue.FieldByName("children")
	if !childrenValue.IsValid() {
		return
	}

	childrenIter := childrenValue.MapRange()
	for childrenIter.Next() {
		child := childrenIter.Value().Interface()
		doc.extractRoutesFromNode(method, fullPath, child)
	}

	// 检查参数子节点
	paramChildValue := nodeValue.FieldByName("paramChild")
	if paramChildValue.IsValid() && !paramChildValue.IsNil() {
		child := paramChildValue.Interface()
		doc.extractRoutesFromNode(method, fullPath, child)
	}

	// 检查通配符子节点
	wildcardChildValue := nodeValue.FieldByName("wildcardChild")
	if wildcardChildValue.IsValid() && !wildcardChildValue.IsNil() {
		child := wildcardChildValue.Interface()
		doc.extractRoutesFromNode(method, fullPath, child)
	}

	// 检查正则子节点
	regChildValue := nodeValue.FieldByName("regChild")
	if regChildValue.IsValid() && !regChildValue.IsNil() {
		child := regChildValue.Interface()
		doc.extractRoutesFromNode(method, fullPath, child)
	}
}
