package mist

// Middleware 表示Go中的一个函数类型，定义了中间件函数的结构。
// 在Web服务器或其他请求处理应用程序的上下文中，中间件用于在请求到达最终请求处理程序之前处理请求，
// 允许进行预处理，如认证、日志记录或在请求的主要处理之前或之后应执行的任何其他操作。
//
// 该类型被定义为一个函数，它接受一个HandleFunc作为参数（通常称为'next'）并返回另一个HandleFunc。
// 括号内的HandleFunc是链中中间件将调用的下一个函数，而返回的HandleFunc是该函数的修改或"包装"版本。
//
// 典型的中间件将执行一些操作，然后调用'next'将控制权传递给后续的中间件或最终处理程序，
// 可能在'next'返回后执行一些操作，最后返回'next'的结果。通过这样做，它形成了一个请求流经的中间件函数链。
//
// Middleware类型设计得灵活且可组合，使得有序的中间件函数序列的构建变得简单和模块化。
//
// 参数:
//   - 'next': 要用额外行为包装的HandleFunc。这是通常会处理请求的函数或者是链中的下一个中间件。
//
// 返回值:
// - 一个HandleFunc，表示将中间件的行为添加到'next'函数后的结果。
//
// 用法:
//   - 中间件函数通常与路由器或服务器一起使用，以处理HTTP请求。
//   - 它们被链接在一起，使得请求在最终被主处理函数处理之前通过一系列中间件。
//
// 注意事项:
//   - 在设计中间件时，应确保不会无意中跳过必要的'next'处理程序。
//     除非是有意的（例如，阻止未授权请求的授权中间件），中间件通常应该调用'next'。
//   - 小心处理中间件中的错误。决定是在中间件本身内处理和记录错误，还是将它们传递给其他机制处理。
//   - 中间件函数应避免更改请求，除非这是其明确职责的一部分，
//     例如设置上下文值或修改与中间件特定功能相关的头部。
type Middleware func(next HandleFunc) HandleFunc
