# Mist框架生产环境最佳实践指南

本文档提供了在生产环境中使用Mist框架的最佳实践和建议。遵循这些指南将帮助您构建更安全、高效和可靠的Web应用程序。

## 目录

1. [服务器配置](#服务器配置)
2. [安全性](#安全性)
3. [性能优化](#性能优化)
4. [日志记录](#日志记录)
5. [监控和指标](#监控和指标)
6. [分布式部署](#分布式部署)
7. [容错和恢复](#容错和恢复)
8. [会话管理](#会话管理)
9. [Docker和容器化](#Docker和容器化)
10. [CI/CD集成](#CICD集成)

## 服务器配置

### 优雅关闭

始终在生产环境中实现优雅关闭，以确保正在处理的请求能够完成：

```go
server := mist.InitHTTPServer()
// 配置路由等...

// 在单独的goroutine中启动服务器
go func() {
    if err := server.Start(":8080"); err != nil {
        log.Printf("服务器启动错误: %v", err)
    }
}()

// 监听关闭信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

log.Println("正在关闭服务器...")
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    log.Fatalf("服务器强制关闭: %v", err)
}
log.Println("服务器已优雅关闭")
```

### 超时配置

为防止慢请求耗尽资源，配置适当的超时：

```go
config := mist.DefaultServerConfig()
config.ReadTimeout = 30 * time.Second
config.WriteTimeout = 30 * time.Second
config.IdleTimeout = 120 * time.Second

server := mist.InitHTTPServer(mist.WithServerConfig(config))
```

### TLS配置

在生产环境中使用HTTPS是必须的：

```go
// 方法1: 使用StartTLS方法
server.StartTLS(":443", "cert.pem", "key.pem")

// 方法2: 使用自定义TLS配置
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
    CipherSuites: []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
}

// 自定义HTTP服务器配置
config := mist.DefaultServerConfig()
server := mist.InitHTTPServer(mist.WithServerConfig(config))

// 手动配置TLS
server.httpServer.TLSConfig = tlsConfig
server.StartTLS(":443", "cert.pem", "key.pem")
```

## 安全性

### 使用安全头部中间件

添加安全相关的HTTP头部：

```go
import "github.com/dormoron/mist/middlewares/secureheader"

server := mist.InitHTTPServer()

// 使用默认安全头
server.Use(secureheader.WithSecureHeaders())

// 或自定义安全头
server.Use(secureheader.WithCustomSecureHeaders(func(opts *secureheader.Options) {
    opts.ContentSecurityPolicy = "default-src 'self'"
    opts.XFrameOptions = "DENY"
}))
```

### CSRF保护

为所有修改数据的请求添加CSRF保护：

```go
import "github.com/dormoron/mist/middlewares/csrf"

server := mist.InitHTTPServer()

// 使用默认CSRF保护
server.Use(csrf.WithDefaultCSRF())

// 或自定义CSRF配置
csrfOpts := csrf.DefaultOptions()
csrfOpts.CookieSecure = true
csrfOpts.CookieSameSite = http.SameSiteStrictMode
server.Use(csrf.NewMiddleware(csrfOpts))
```

### 输入验证

始终验证所有用户输入：

```go
server.POST("/api/users", func(ctx *mist.Context) {
    var user User
    if err := ctx.BindJSON(&user); err != nil {
        ctx.AbortWithStatus(http.StatusBadRequest)
        ctx.RespData = []byte(`{"error":"无效的请求数据"}`)
        return
    }
    
    // 验证用户数据
    if err := validateUser(user); err != nil {
        ctx.AbortWithStatus(http.StatusBadRequest)
        ctx.RespData = []byte(`{"error":"` + err.Error() + `"}`)
        return
    }
    
    // 处理有效数据...
})
```

## 性能优化

### 启用路由缓存

对于生产环境，启用路由缓存以提高性能：

```go
server := mist.InitHTTPServer()
// 默认情况下路由缓存是启用的，缓存大小为1000
// 如需调整缓存大小：
server.SetRouterCacheSize(5000) 
```

### 启用路由统计

监控路由性能：

```go
server := mist.InitHTTPServer()
server.EnableRouteStats()

// 添加一个端点来获取统计信息
server.GET("/admin/stats", func(ctx *mist.Context) {
    stats := server.GetAllRouteStats()
    data, _ := json.Marshal(stats)
    ctx.RespData = data
    ctx.RespStatusCode = 200
})
```

### 使用缓存中间件

缓存频繁访问且不经常变化的响应：

```go
import "github.com/dormoron/mist/middlewares/cache"

// 创建缓存，最多1000项，TTL为5分钟
responseCache, _ := cache.New(1000, 5*time.Minute)

// 对特定路由使用缓存
server.GET("/api/products", getProducts, responseCache.Middleware(cache.URLKeyGenerator()))
```

### 限制并发请求数

防止服务器过载：

```go
import "github.com/dormoron/mist/middlewares/activelimit/locallimit"

// 最多允许1000个并发请求
limiter := locallimit.InitMiddlewareBuilder(1000)

// 添加到全局中间件
server.Use(limiter.Build())
```

## 日志记录

### 配置结构化日志

使用结构化日志提高可搜索性：

```go
// 设置日志级别
mist.GetDefaultLogger().SetLevel(mist.LevelInfo)

// 使用结构化字段
mist.WithField("user_id", userId).
    WithField("action", "login").
    Info("用户已登录")
```

### 设置日志轮转

配置日志轮转以避免单个日志文件过大：

```go
// 假设已集成了第三方日志轮转库
config := mist.LogRotateConfig{
    Filename:   "/var/log/myapp/server.log",
    MaxSize:    100, // MB
    MaxAge:     30,  // 天
    MaxBackups: 10,
    LocalTime:  true,
    Compress:   true,
}

// 设置日志轮转
mist.SetupLogRotation(config)
```

## 监控和指标

### 健康检查端点

添加健康检查端点：

```go
import "github.com/dormoron/mist/middlewares/healthcheck"

// 创建健康检查中间件
health := healthcheck.InitMiddleware("/health")

// 注册数据库健康检查
health.RegisterComponent("database", func() (healthcheck.Status, map[string]any) {
    if err := db.Ping(); err != nil {
        return healthcheck.StatusDown, map[string]any{
            "error": err.Error(),
        }
    }
    return healthcheck.StatusUp, map[string]any{
        "latency_ms": 10,
    }
})

// 添加到全局中间件
server.Use(health.Build())
```

### Prometheus指标

集成Prometheus指标：

```go
import "github.com/dormoron/mist/middlewares/prometheus"

// 创建Prometheus中间件
promMiddleware := prometheus.New(prometheus.Config{
    Namespace: "myapp",
    Subsystem: "http",
})

// 注册自定义指标
requestsCounter := promMiddleware.AddCounter(prometheus.CounterOpts{
    Name: "requests_total",
    Help: "Total number of HTTP requests",
})

// 添加到全局中间件
server.Use(promMiddleware.Build())

// 添加Prometheus指标端点
server.GET("/metrics", promMiddleware.Handler())
```

## 分布式部署

### 分布式会话

使用Redis存储会话以支持水平扩展：

```go
import "github.com/dormoron/mist/middlewares/session"

// 创建Redis会话存储
sessionMiddleware, err := session.NewRedisStore(
    "redis:6379",       // Redis地址
    "password",         // Redis密码
    0,                  // Redis数据库
    "myapp:sessions:",  // 键前缀
    session.WithMaxAge(3600),
    session.WithCookieSecure(true),
)
if err != nil {
    log.Fatalf("创建会话中间件失败: %v", err)
}

// 添加到全局中间件
server.Use(sessionMiddleware.Build())

// 启动会话清理
go session.SessionCleanupTask(sessionMiddleware.manager, 30*time.Minute)()
```

### 使用负载均衡器

配置多个实例以及负载均衡：

```
# Nginx负载均衡配置示例
upstream myapp {
    server app1:8080;
    server app2:8080;
    server app3:8080;
}

server {
    listen 80;
    server_name myapp.example.com;

    location / {
        proxy_pass http://myapp;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 容错和恢复

### 使用恢复中间件

添加恢复中间件以防止崩溃：

```go
import "github.com/dormoron/mist/middlewares/recovery"

// 使用默认恢复中间件
server.Use(recovery.NewRecoveryMiddleware())

// 或使用JSON恢复中间件
server.Use(recovery.JSONRecoveryMiddleware())

// 或自定义恢复中间件
builder := recovery.InitMiddlewareBuilder(
    http.StatusInternalServerError,
    []byte(`{"error":"服务器内部错误"}`),
)
builder.SetPrintStack(true)
builder.SetLogFunc(func(ctx *mist.Context, err any) {
    mist.WithField("request_id", ctx.RequestID()).
        WithField("error", fmt.Sprintf("%v", err)).
        Error("服务器异常")
})
server.Use(builder.Build())
```

### 限流中间件

防止请求过载：

```go
import (
    "github.com/dormoron/mist/middlewares/ratelimit"
    "github.com/dormoron/mist/internal/ratelimit"
)

// 每IP每分钟最多100个请求
limiter := ratelimit.InitMiddlewareBuilder(
    ratelimit.NewTokenBucket(100, 100, time.Minute),
    5, // 重试间隔秒数
)
server.Use(limiter.Build())
```

## 会话管理

### 安全的会话配置

确保会话安全：

```go
import "github.com/dormoron/mist/middlewares/session"

sessionMiddleware, _ := session.NewMemoryStore(
    session.WithCookieName("app_session"),
    session.WithCookieSecure(true),
    session.WithCookieHTTPOnly(true),
    session.WithCookieSameSite(http.SameSiteStrictMode),
    session.WithMaxAge(3600), // 1小时
)
server.Use(sessionMiddleware.Build())
```

## Docker和容器化

### 示例Dockerfile

```dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o myapp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/myapp /app/

# 创建非root用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 运行应用
CMD ["/app/myapp"]
```

### docker-compose.yml示例

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=db
      - REDIS_HOST=redis
      - LOG_LEVEL=info
    depends_on:
      - db
      - redis
    restart: always
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M

  db:
    image: postgres:14
    volumes:
      - db-data:/var/lib/postgresql/data
    environment:
      - POSTGRES_PASSWORD=secret
      - POSTGRES_USER=app
      - POSTGRES_DB=appdb
    restart: always

  redis:
    image: redis:7
    volumes:
      - redis-data:/data
    restart: always

volumes:
  db-data:
  redis-data:
```

## CI/CD集成

### 示例GitHub Actions工作流

```yaml
name: Build and Deploy

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.20'
    - name: Test
      run: go test -v ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Build Docker image
      run: docker build -t myapp:${{ github.sha }} .
    - name: Login to Docker Hub
      uses: docker/login-action@v2
      with:
        username: ${{ secrets.DOCKER_HUB_USERNAME }}
        password: ${{ secrets.DOCKER_HUB_ACCESS_TOKEN }}
    - name: Push Docker image
      run: |
        docker tag myapp:${{ github.sha }} username/myapp:latest
        docker push username/myapp:latest

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - name: Deploy to server
      uses: appleboy/ssh-action@master
      with:
        host: ${{ secrets.HOST }}
        username: ${{ secrets.USERNAME }}
        key: ${{ secrets.SSH_KEY }}
        script: |
          cd /app
          docker-compose pull
          docker-compose up -d
```

## 总结

通过遵循这些生产环境最佳实践，您可以确保Mist框架应用程序具有更高的安全性、可靠性和性能。根据您的具体需求和业务场景，这些建议可能需要进一步调整。始终保持代码和依赖库的更新，并定期检查安全漏洞。 