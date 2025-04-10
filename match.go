package mist

// matchInfo 保存匹配路由所需的必要信息。它封装了匹配的节点、从URL提取的路径参数以及应该应用于该路由的中间件列表。
// 这个结构体通常在路由系统中使用，负责在成功匹配路由后承载处理HTTP请求所需的累积数据。
//
// 字段:
//   - n (*node): 指向匹配的'node'的指针，该节点表示在路由树中已经与传入请求路径匹配的端点。
//     这个'node'包含处理请求所需的必要信息，如关联的处理程序或其他路由信息。
//   - pathParams (map[string]string): 一个存储路径参数的键值对映射，其中键是参数的名称（在路径中定义），
//     值是从请求URL匹配的实际字符串。例如，对于路由模式"/users/:userID/posts/:postID"，
//     如果传入的请求路径匹配该模式，则此映射将包含"userID"和"postID"的条目。
//   - mils ([]Middleware): 一个'Middleware'函数的切片，按照切片中包含的顺序为匹配的路由执行。
//     中间件函数用于在请求到达最终处理函数之前执行操作，如请求日志记录、认证和输入验证等。
//
// 用法:
// 'matchInfo'结构体在路由匹配过程中被填充。一旦请求路径与路由树匹配，就创建一个'matchInfo'实例，
// 并填充相应的节点、提取的路径参数以及与匹配路由相关的任何中间件。然后将此实例传递给请求处理逻辑，
// 引导请求通过各种中间件层的处理，最终到达将生成响应的适当处理程序。
type matchInfo struct {
	// n 是路由树中与匹配路由对应的节点。它提供了访问处理请求所需的任何其他特定于路由的信息。
	n *node

	// pathParams 存储在URL路径中标识的参数，如"/users/:id"中的"id"，映射到从传入请求解析的实际值。
	pathParams map[string]string

	// mils 是要为匹配的路由按顺序执行的中间件函数集合。这些函数可以修改请求上下文、执行检查或进行其他预处理任务。
	mils []Middleware
}

// addValue 是一个方法，用于向matchInfo结构体的pathParams映射中添加键值对。
// 这个方法用于累积从匹配的URL路径中提取的参数，并将它们存储起来，以便在请求处理过程中后续使用。
//
// 参数:
//   - key: 一个字符串，表示URL参数的名称（例如，"userID"）。
//   - value: 一个字符串，表示从请求URL中提取的URL参数的值（例如，对于userID，值可能是"42"）。
//
// addValue函数执行以下步骤:
//
//  1. 检查matchInfo结构体内的pathParams映射是否为nil，这表示尚未添加任何参数。
//     如果是nil，则初始化pathParams映射并立即向其中添加键值对。这是必要的，因为不能向nil映射添加键；必须先初始化它。
//  2. 如果pathParams映射已经初始化，则添加或覆盖键的条目，赋予新值。
//     这确保了对于给定键，映射中存储的是最近处理的值。
//
// 用法:
// addValue方法通常在路由匹配过程中调用，期间解析与路由模式中的参数对应的路径段，并累积它们的值。
// 每次处理一个段并提取参数值时，都会使用addValue来保存该值和相应的参数名称。
//
// 示例:
// 对于像"/users/:userID"这样的URL模式，在处理像"/users/42"这样的请求路径时，
// 该方法将被调用为addValue("userID", "42")，向pathParams映射添加参数"userID"及其值"42"。
func (m *matchInfo) addValue(key string, value string) {
	// 如果尚未初始化pathParams映射，则进行初始化，以避免nil映射赋值导致的panic。
	if m.pathParams == nil {
		m.pathParams = map[string]string{key: value}
	}
	// 向pathParams映射添加或更新键值对，表示URL参数及其值。
	m.pathParams[key] = value
}
