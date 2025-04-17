# Mist Web 框架

Mist 是一个轻量级、高性能的 Go Web 框架，专注于简洁易用的 API 设计和灵活的路由系统，同时提供全面的中间件支持和性能优化。

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 主要特性

### 核心功能

- **高性能路由系统**: 基于前缀树实现的高效路由匹配，支持路由缓存
- **灵活的路由模式**: 支持静态路由、参数路由、正则表达式路由和通配符路由
- **简洁 API 设计**: 易于学习和使用的 API 接口
- **路由分组**: 简化相关路由的管理和中间件应用
- **类型安全**: 提供类型安全的参数处理和值提取

### 性能优化

- **HTTP/3 支持**: 完整实现基于 QUIC 的 HTTP/3，支持平台特定优化
- **自适应路由缓存**: 智能缓存机制，根据访问模式动态调整缓存内容
- **上下文对象池**: 优化对象复用，减少 GC 压力
- **零拷贝响应**: 高效文件传输，减少内存和 CPU 开销
- **内存使用监控**: 实时监控内存使用，支持异常告警

### 中间件支持

- **身份验证**: JWT、Basic Auth、OAuth 等认证方式
- **安全增强**: CORS、CSRF、XSS 防护、安全 HTTP 头
- **请求限制**: 请求体大小限制、速率限制、并发连接限制
- **监控与追踪**: Prometheus 指标、OpenTelemetry 集成
- **会话管理**: 安全的会话处理，支持多种存储后端
- **错误处理**: 集中式错误处理，支持自定义错误响应
- **缓存**: 响应缓存，支持多级缓存策略

## 快速开始

### 安装

```bash
go get github.com/dormoron/mist
```

### 基础示例

```go
package main

import (
    "github.com/dormoron/mist"
)

func main() {
    // 初始化服务器
    server := mist.InitHTTPServer()
    
    // 注册路由
    server.GET("/", func(ctx *mist.Context) {
        ctx.RespondWithJSON(200, map[string]string{
            "message": "Hello, Mist!",
        })
    })
    
    // 添加中间件
    server.Use(middlewares.Recovery())
    server.Use(middlewares.AccessLog())
    
    // 启动服务器
    server.Start(":8080")
}
```

### 路由系统示例

```go
// 参数路由
server.GET("/users/:id", func(ctx *mist.Context) {
    id := ctx.PathValue("id").String()
    ctx.RespondWithJSON(200, map[string]string{
        "user_id": id,
    })
})

// 正则表达式路由
server.GET("/posts/{id:[0-9]+}", func(ctx *mist.Context) {
    id := ctx.PathValue("id").Int()
    ctx.RespondWithJSON(200, map[string]string{
        "post_id": fmt.Sprintf("%d", id),
    })
})

// 路由分组
api := server.Group("/api")
api.Use(middlewares.Auth.JWT())

// 用户相关路由
users := api.Group("/users")
users.GET("/", listUsers)
users.POST("/", createUser)
users.GET("/:id", getUser)
users.PUT("/:id", updateUser)
users.DELETE("/:id", deleteUser)
```

## 高级功能

### HTTP/3 支持

```go
server := mist.InitHTTPServer()
// 配置 HTTP/3
config := mist.DefaultHTTP3Config()
config.MaxIdleTimeout = 60 * time.Second

// 启动 HTTP/3 服务器
err := server.StartHTTP3(":443", "cert.pem", "key.pem", config)
if err != nil {
    log.Fatalf("HTTP/3 启动失败: %v", err)
}
```

### 零拷贝文件传输

```go
server.GET("/download/:file", func(ctx *mist.Context) {
    filePath := "path/to/files/" + ctx.PathValue("file").String()
    zr := mist.NewZeroCopyResponse(ctx.ResponseWriter)
    err := zr.ServeFile(filePath)
    if err != nil {
        ctx.RespondWithJSON(500, map[string]string{
            "error": err.Error(),
        })
    }
})
```

### 内存监控

```go
monitor := mist.NewMemoryMonitor(10*time.Second, 60)
monitor.AddAlertCallback(func(stats mist.MemStats, message string) {
    log.Printf("内存警告: %s, 已分配: %d 字节", message, stats.Alloc)
})
monitor.Start()
defer monitor.Stop()
```

## 中间件使用示例

### 速率限制

```go
// 创建速率限制中间件
limiter := middlewares.RateLimit.NewFixedWindow(middlewares.RateLimit.Config{
    MaxRequests: 100,        // 每窗口最大请求数
    Window:      time.Minute, // 窗口大小
    KeyFunc: func(ctx *mist.Context) string {
        return ctx.ClientIP() // 基于客户端IP限流
    },
})

// 应用到所有路由
server.Use(limiter)

// 或仅应用到特定路由组
apiGroup := server.Group("/api")
apiGroup.Use(limiter)
```

### CORS 配置

```go
// 使用默认 CORS 配置
server.Use(middlewares.CORS.Default())

// 或使用自定义配置
corsConfig := middlewares.CORS.Config{
    AllowOrigins:     []string{"https://example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}
server.Use(middlewares.CORS.New(corsConfig))
```

### 请求体大小限制

```go
// 全局限制请求体大小为1MB
server.Use(middlewares.BodyLimit.New("1MB"))

// 或使用自定义配置
config := middlewares.BodyLimit.Config{
    MaxSize:        2 * 1024 * 1024, // 2MB
    WhitelistPaths: []string{"/upload"},
}
server.Use(middlewares.BodyLimit.NewWithConfig(config))
```

## 安全特性

Mist 框架提供多方面安全增强：

- **账户锁定策略**: 防止暴力破解攻击
- **会话安全增强**: 会话指纹绑定，令牌轮换，多级超时策略
- **安全 HTTP 头**: 自动设置 CSP、HSTS、X-Frame-Options 等安全头
- **密码安全**: 密码强度检测，历史记录检查
- **内容类型保护**: 自动检测和设置正确的 Content-Type

## 贡献指南

我们欢迎各种形式的贡献，包括但不限于：

- 提交问题和功能请求
- 提交代码改进
- 改进文档
- 分享使用经验

贡献代码的步骤：

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

Mist 框架使用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。

## 联系方式

- 项目维护者: [dormoron](https://github.com/dormoron)
- 项目主页: [GitHub](https://github.com/dormoron/mist)

---

Mist Framework - 为高性能 Go Web 应用而生

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

