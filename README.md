# Mist Web Framework

Mist 是一个轻量级、高性能的 Go Web 框架，专注于简洁易用的 API 设计和灵活的路由系统，同时提供全面的中间件支持。

## 特性

- **高性能路由系统**：基于前缀树实现的高效路由匹配
- **灵活的路由模式**：支持静态路由、参数路由、正则表达式路由和通配符路由
- **中间件机制**：支持全局中间件和路由级别中间件
- **简洁的 API 设计**：易于学习和使用的 API 接口
- **路由缓存**：通过缓存机制提升高频路由的匹配性能
- **类型安全**：提供类型安全的参数处理和值提取
- **易扩展**：框架核心设计简洁，易于扩展

## 安装

```bash
go get github.com/dormoron/mist
```

## 快速开始

创建一个简单的 HTTP 服务器：

```go
package main

import (
    "github.com/dormoron/mist"
)

func main() {
    server := mist.InitHTTPServer()
    
    // 注册路由
    server.GET("/", func(ctx *mist.Context) {
        ctx.RespData = []byte("Hello, Mist!")
        ctx.RespStatusCode = 200
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

## 路由系统

Mist 框架支持多种路由模式，满足不同场景需求：

### 静态路由

```go
server.GET("/users", func(ctx *mist.Context) {
    ctx.RespData = []byte("User List")
    ctx.RespStatusCode = 200
})
```

### 参数路由

使用`:param`格式定义路径参数：

```go
server.GET("/users/:id", func(ctx *mist.Context) {
    id := ctx.PathParams["id"]
    ctx.RespData = []byte("User ID: " + id)
    ctx.RespStatusCode = 200
})
```

### 正则表达式路由

Mist支持两种正则表达式路由格式：

```go
// 花括号格式: {paramName:pattern}
server.GET("/users/{id:[0-9]+}", func(ctx *mist.Context) {
    id := ctx.PathParams["id"]
    ctx.RespData = []byte("User ID (numeric): " + id)
    ctx.RespStatusCode = 200
})

// 冒号括号格式: :paramName(pattern)
server.GET("/posts/:id(\\d+)", func(ctx *mist.Context) {
    id := ctx.PathParams["id"]
    ctx.RespData = []byte("Post ID (numeric): " + id)
    ctx.RespStatusCode = 200
})
```

### 通配符路由

使用`*`匹配剩余路径段：

```go
server.GET("/files/*filepath", func(ctx *mist.Context) {
    filepath := ctx.PathParams["filepath"]
    ctx.RespData = []byte("File path: " + filepath)
    ctx.RespStatusCode = 200
})
```

## HTTP方法支持

Mist支持所有标准HTTP方法：

```go
server.GET("/users", listUsers)
server.POST("/users", createUser)
server.PUT("/users/:id", updateUser)
server.DELETE("/users/:id", deleteUser)
server.PATCH("/users/:id", partialUpdateUser)
server.OPTIONS("/users", usersOptions)
server.HEAD("/users", usersHead)
```

## 中间件

Mist 提供了灵活的中间件机制，支持全局中间件和路由级中间件。

### 全局中间件

全局中间件会应用到所有路由：

```go
server := mist.InitHTTPServer()

// 添加全局日志中间件
server.Use(func(next mist.HandleFunc) mist.HandleFunc {
    return func(ctx *mist.Context) {
        // 请求处理前的操作
        startTime := time.Now()
        
        // 调用下一个中间件或处理函数
        next(ctx)
        
        // 请求处理后的操作
        duration := time.Since(startTime)
        fmt.Printf("[%s] %s - %d (%v)\n", 
            ctx.Request.Method, 
            ctx.Request.URL.Path, 
            ctx.RespStatusCode, 
            duration)
    }
})
```

### 路由级中间件

路由级中间件只应用于特定路由：

```go
// 身份验证中间件
func authMiddleware(next mist.HandleFunc) mist.HandleFunc {
    return func(ctx *mist.Context) {
        token := ctx.Request.Header.Get("Authorization")
        if !isValidToken(token) {
            ctx.RespStatusCode = 401
            ctx.RespData = []byte("Unauthorized")
            return
        }
        next(ctx)
    }
}

// 添加路由级中间件
server.GET("/protected", func(ctx *mist.Context) {
    ctx.RespData = []byte("Protected resource")
    ctx.RespStatusCode = 200
}, authMiddleware)
```

## 路由分组

Mist支持路由分组，方便管理相关路由和应用中间件：

```go
// 创建API路由组
apiGroup := server.Group("/api")

// 为API组添加中间件
apiGroup.Use(authMiddleware)

// 在组中添加路由
apiGroup.GET("/users", listUsers)
apiGroup.POST("/users", createUser)

// 创建嵌套组
adminGroup := apiGroup.Group("/admin")
adminGroup.Use(adminAuthMiddleware)
adminGroup.GET("/stats", getStats)
```

## 请求和响应处理

Mist提供了便捷的请求处理和响应生成功能：

### 请求参数获取

```go
server.GET("/search", func(ctx *mist.Context) {
    // 获取查询参数
    query := ctx.QueryValue("q").StringOrDefault("")
    page := ctx.QueryValue("page").IntOrDefault(1)
    
    // 处理请求...
    
    ctx.RespStatusCode = 200
    ctx.RespData = []byte(fmt.Sprintf("Query: %s, Page: %d", query, page))
})
```

### JSON请求处理

```go
server.POST("/users", func(ctx *mist.Context) {
    var user User
    if err := ctx.ReadJSON(&user); err != nil {
        ctx.RespStatusCode = 400
        ctx.RespData = []byte("Invalid request")
        return
    }
    
    // 处理用户数据...
    
    ctx.RespStatusCode = 201
    ctx.RespData = []byte("User created")
})
```

## 性能优化

Mist框架内置多种性能优化机制：

- 基于前缀树的高效路由匹配
- 路由缓存机制，提高访问频率高的路由性能
- 优化的内存分配策略，减少GC压力
- 高效的正则表达式匹配实现

## 示例项目

查看[examples目录](./examples)获取更多实际应用示例：

- 基础Web服务器
- REST API实现
- 中间件示例
- 文件服务器
- WebSocket支持

## 贡献指南

我们欢迎所有形式的贡献，包括但不限于：

- 提交问题和功能请求
- 提交代码改进
- 改进文档
- 分享使用经验

请遵循以下步骤贡献代码：

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建Pull Request

## 许可证

Mist框架使用MIT许可证 - 详情请参阅[LICENSE](LICENSE)文件。

## 联系方式

- 项目维护者: [dormoron](https://github.com/dormoron)
- 项目主页: [GitHub](https://github.com/dormoron/mist)

---

Mist Framework - 为高性能Go Web应用而生

## 安全优化

本项目最近进行了全面的安全增强，包括以下方面：

### 1. 账户锁定策略

为防止暴力破解攻击，增加了账户锁定机制：

- 可配置最大失败尝试次数
- 自动锁定时间设置
- 基于用户ID和IP的锁定策略
- 定期清理过期锁定记录

### 2. 会话安全增强

改进了会话管理安全性：

- 支持会话指纹绑定（检测会话劫持）
- 自动会话令牌轮换
- 多级超时策略（绝对超时和闲置超时）
- 敏感操作的重新认证要求

### 3. 安全HTTP头

实现了全面的安全HTTP头设置：

- Content-Security-Policy (CSP)
- Strict-Transport-Security (HSTS)
- X-Frame-Options
- X-Content-Type-Options
- X-XSS-Protection
- Referrer-Policy
- Permissions-Policy
- Cross-Origin 相关策略

### 4. 其他安全特性

- 密码强度检测和历史记录检查
- 自动检测和设置正确的Content-Type
- 移除泄露服务器信息的HTTP头

