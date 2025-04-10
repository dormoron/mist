# Mist框架性能调优指南

本文档提供了优化Mist框架性能的详细指南，帮助您在生产环境中获得最佳性能。

## 目录

1. [服务器配置优化](#服务器配置优化)
2. [路由优化](#路由优化)
3. [中间件优化](#中间件优化)
4. [内存管理](#内存管理)
5. [并发控制](#并发控制)
6. [数据库连接优化](#数据库连接优化)
7. [静态资源处理](#静态资源处理)
8. [监控与分析](#监控与分析)
9. [HTTP/2和HTTP/3优化](#HTTP/2和HTTP/3优化)
10. [负载测试](#负载测试)

## 服务器配置优化

### 超时设置

合理配置超时参数可以防止慢请求耗尽服务器资源：

```go
config := mist.DefaultServerConfig()
// 根据应用特性调整超时参数
config.ReadTimeout = 5 * time.Second       // 减少读取超时以快速释放僵尸连接
config.WriteTimeout = 10 * time.Second     // 避免慢客户端占用资源
config.IdleTimeout = 60 * time.Second      // 空闲连接保持时间
config.ReadHeaderTimeout = 2 * time.Second // 头部读取超时
config.RequestTimeout = 30 * time.Second   // 单个请求的处理超时

server := mist.InitHTTPServer(mist.WithServerConfig(config))
```

### 最大连接数

控制最大并发连接数可以防止服务器过载：

```go
import "github.com/dormoron/mist/middlewares/activelimit/locallimit"

// 限制最大并发请求数为5000
limiter := locallimit.InitMiddlewareBuilder(5000)
server.Use(limiter.Build())
```

### Keep-Alive设置

在高并发场景下优化Keep-Alive设置可以减少连接建立开销：

```go
server.httpServer.SetKeepAlivesEnabled(true) // 启用Keep-Alive
// IdleTimeout参数控制Keep-Alive连接的最大空闲时间
```

## 路由优化

### 启用路由缓存

对于有大量路由定义的应用，启用路由缓存可以显著提高路由匹配性能：

```go
// 启用路由缓存并设置大小
server.SetRouterCacheSize(10000) // 默认为1000
```

### 监控路由性能

使用内置的路由统计功能找出性能瓶颈：

```go
// 启用路由统计
server.EnableRouteStats()

// 添加监控端点
server.GET("/metrics/routes", func(ctx *mist.Context) {
    stats := server.GetAllRouteStats()
    data, _ := json.Marshal(stats)
    ctx.RespData = data
    ctx.RespStatusCode = 200
})
```

### 优化正则表达式路由

尽量避免复杂的正则表达式路由，优先使用静态路由和简单参数路由：

```go
// 推荐 - 简单参数路由
server.GET("/users/:id", handleUser)

// 避免 - 复杂正则表达式
server.GET("/users/{id:[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}}", handleUser)
```

## 中间件优化

### 中间件顺序

合理安排中间件顺序可以提高性能：

1. 请求计数/限流中间件（首位）
2. 恢复中间件（提前捕获panic）
3. 安全相关中间件
4. 日志/监控中间件
5. 压缩/缓存中间件（靠后位置）

```go
// 优化的中间件顺序
server.Use(activelimit.Middleware())  // 1. 限流
server.Use(recovery.Middleware())     // 2. 恢复
server.Use(auth.Middleware())         // 3. 认证
server.Use(logger.Middleware())       // 4. 日志
server.Use(cache.Middleware())        // 5. 缓存
```

### 按需应用中间件

只在需要的路由上应用中间件，避免全局使用重型中间件：

```go
// 只对API路由应用认证中间件
apiGroup := server.Group("/api")
apiGroup.Use(auth.Middleware())

// 只对静态资源应用缓存中间件
server.GET("/static/*filepath", handleStatic, cache.Middleware())
```

### 使用中间件构建器

使用构建器模式创建高度优化的中间件：

```go
// 创建轻量级日志中间件
logger := logger.InitMiddlewareBuilder().
    WithLogLevel(logger.LevelInfo).
    WithoutRequestBody().  // 不记录请求体以减少内存使用
    WithSamplingRate(0.1). // 只采样10%的请求
    Build()

server.Use(logger)
```

## 内存管理

### 使用对象池

对于频繁创建的对象，使用对象池减少GC压力：

```go
// 在处理大量JSON请求时使用解码器池
var decoderPool = sync.Pool{
    New: func() interface{} {
        return json.NewDecoder(nil)
    },
}

// 使用池中的解码器
decoder := decoderPool.Get().(*json.Decoder)
decoder.Reset(req.Body)
defer decoderPool.Put(decoder)
```

### 控制请求体大小

限制请求体大小以防止内存耗尽攻击：

```go
// 使用中间件限制请求体大小
server.Use(func(next mist.HandleFunc) mist.HandleFunc {
    return func(ctx *mist.Context) {
        if ctx.Request.ContentLength > 10*1024*1024 { // 10MB限制
            ctx.AbortWithStatus(http.StatusRequestEntityTooLarge)
            return
        }
        next(ctx)
    }
})
```

### 使用流式处理

处理大型响应时使用流式处理而非一次性加载到内存：

```go
server.GET("/large-file", func(ctx *mist.Context) {
    file, err := os.Open("large.file")
    if err != nil {
        ctx.AbortWithStatus(500)
        return
    }
    defer file.Close()
    
    ctx.Header("Content-Type", "application/octet-stream")
    ctx.ResponseWriter.WriteHeader(200)
    
    // 流式复制，而非一次性读取到内存
    io.Copy(ctx.ResponseWriter, file)
})
```

## 并发控制

### 使用上下文取消

在长时间运行的处理程序中正确使用上下文取消：

```go
server.GET("/long-running", func(ctx *mist.Context) {
    // 创建派生上下文，5秒超时
    derivedCtx, cancel := context.WithTimeout(ctx.Context(), 5*time.Second)
    defer cancel()
    
    // 使用派生上下文调用耗时服务
    result, err := longRunningService.Process(derivedCtx, params)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            ctx.AbortWithStatus(http.StatusGatewayTimeout)
            return
        }
        ctx.AbortWithStatus(http.StatusInternalServerError)
        return
    }
    
    ctx.RespData = result
    ctx.RespStatusCode = 200
})
```

### 协程池

使用协程池处理并发任务，避免无限制创建goroutine：

```go
// 创建工作池
pool := &sync.Pool{
    New: func() interface{} {
        return make(chan struct{}, 100) // 限制并发数为100
    },
}

server.POST("/process-tasks", func(ctx *mist.Context) {
    sem := pool.Get().(chan struct{})
    defer pool.Put(sem)
    
    var tasks []Task
    if err := ctx.BindJSON(&tasks); err != nil {
        ctx.AbortWithStatus(400)
        return
    }
    
    var wg sync.WaitGroup
    results := make([]Result, len(tasks))
    
    for i, task := range tasks {
        wg.Add(1)
        
        // 使用信号量控制并发
        sem <- struct{}{}
        
        go func(idx int, t Task) {
            defer wg.Done()
            defer func() { <-sem }()
            
            // 处理任务
            results[idx] = processTask(t)
        }(i, task)
    }
    
    wg.Wait()
    ctx.RespondWithJSON(200, results)
})
```

## 数据库连接优化

### 连接池配置

优化数据库连接池配置：

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

func InitDB() *sql.DB {
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
    if err != nil {
        panic(err)
    }
    
    // 设置最大连接数
    db.SetMaxOpenConns(150)
    
    // 设置最大空闲连接数
    db.SetMaxIdleConns(50)
    
    // 设置连接最大生命周期
    db.SetConnMaxLifetime(30 * time.Minute)
    
    // 设置连接最大空闲时间
    db.SetConnMaxIdleTime(10 * time.Minute)
    
    return db
}
```

### 批量操作

使用批量查询和更新减少数据库往返：

```go
// 单次批量插入多个记录
func BulkInsert(users []User) error {
    query := "INSERT INTO users (name, email) VALUES "
    vals := []interface{}{}
    
    for i, user := range users {
        if i > 0 {
            query += ","
        }
        query += "(?, ?)"
        vals = append(vals, user.Name, user.Email)
    }
    
    _, err := db.Exec(query, vals...)
    return err
}
```

## 静态资源处理

### 启用压缩

为静态资源启用压缩：

```go
import "github.com/dormoron/mist/middlewares/compress"

// 创建压缩中间件
compressor := compress.InitMiddlewareBuilder().
    WithLevel(compress.LevelDefault).
    WithMinSize(1024). // 仅压缩大于1KB的响应
    Build()

// 应用到静态文件路由
server.GET("/static/*filepath", handleStatic, compressor)
```

### 缓存控制

设置适当的缓存头：

```go
server.GET("/static/*filepath", func(ctx *mist.Context) {
    // 设置缓存控制头
    ctx.Header("Cache-Control", "public, max-age=86400") // 1天
    ctx.Header("Expires", time.Now().Add(24*time.Hour).Format(http.TimeFormat))
    
    // 处理静态资源...
})
```

## 监控与分析

### 使用Prometheus监控

集成Prometheus监控中间件：

```go
import "github.com/dormoron/mist/middlewares/prometheus"

// 创建Prometheus监控中间件
metrics := prometheus.InitMiddlewareBuilder(
    "mist", "api", "requests", "API request metrics",
).Build()

// 全局应用监控
server.Use(metrics)

// 添加Prometheus指标暴露端点
server.GET("/metrics", prometheusHandler)
```

### 性能分析

使用pprof进行性能分析：

```go
import _ "net/http/pprof"

func main() {
    // 启动主应用
    go func() {
        server := mist.InitHTTPServer()
        // 配置路由...
        server.Start(":8080")
    }()
    
    // 启动pprof服务
    http.ListenAndServe(":8081", nil)
}
```

## HTTP/2和HTTP/3优化

### 启用HTTP/2

配置服务器支持HTTP/2：

```go
config := mist.DefaultServerConfig()
config.EnableHTTP2 = true // 默认已启用

server := mist.InitHTTPServer(mist.WithServerConfig(config))
server.StartTLS(":443", "cert.pem", "key.pem")
```

### 启用HTTP/3 (实验性)

配置服务器支持HTTP/3：

```go
// 需要先引入HTTP/3实现
import "github.com/quic-go/quic-go/http3"

config := mist.DefaultServerConfig()
config.EnableHTTP3 = true
config.HTTP3IdleTimeout = 30 * time.Second
config.QuicMaxIncomingStreams = 100

server := mist.InitHTTPServer(mist.WithServerConfig(config))
server.StartHTTP3(":443", "cert.pem", "key.pem")
```

## 负载测试

### 使用hey进行负载测试

```bash
# 安装hey
go install github.com/rakyll/hey@latest

# 运行负载测试（200个并发，总共10000个请求）
hey -n 10000 -c 200 http://localhost:8080/api/endpoint
```

### 使用wrk进行负载测试

```bash
# 运行30秒负载测试，100个并发连接
wrk -t12 -c100 -d30s http://localhost:8080/api/endpoint
```

### 逐步增加负载

在生产环境中，应该从低负载开始测试，逐步增加负载，并监控系统资源：

1. 开始于低并发（如10-50）
2. 监控CPU、内存、连接数
3. 每次增加50-100并发
4. 找到性能下降的拐点
5. 调整服务器配置，重复测试

## 性能调优清单

- [ ] 配置合适的超时参数
- [ ] 启用并配置路由缓存
- [ ] 优化中间件顺序和使用
- [ ] 实施对象池和内存优化
- [ ] 控制并发和连接数
- [ ] 优化数据库连接和查询
- [ ] 配置静态资源缓存和压缩
- [ ] 实施监控和指标收集
- [ ] 启用HTTP/2（和可选的HTTP/3）
- [ ] 进行全面的负载测试

通过应用上述优化措施，您的Mist应用应该能够处理更高的并发负载，同时保持低延迟和稳定性。记住，性能调优是一个迭代过程，需要测量、分析和调整。 