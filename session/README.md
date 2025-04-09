# Mist 会话管理模块

Mist框架会话管理模块提供了一个灵活、安全、高性能的会话管理解决方案，支持多种存储后端和传播机制。

## 主要功能

- 支持内存和Redis存储后端
- 基于Cookie的会话传播
- 自动垃圾回收机制
- 线程安全的会话操作
- 会话过期管理
- 完整的测试覆盖

## 快速开始

### 创建内存会话存储

```go
import (
    "github.com/dormoron/mist/session"
    "github.com/dormoron/mist/session/memory"
)

// 创建内存存储
store, err := memory.NewStore()
if err != nil {
    // 处理错误
}

// 创建会话管理器，会话有效期30分钟
manager, err := session.NewManager(store, 1800)
if err != nil {
    // 处理错误
}

// 启用自动垃圾回收，每10分钟执行一次
manager.EnableAutoGC(10 * time.Minute)
```

### 创建Redis会话存储

```go
import (
    "github.com/dormoron/mist/session"
    "github.com/dormoron/mist/session/redis"
    redisClient "github.com/redis/go-redis/v9"
)

// 创建Redis客户端
client := redisClient.NewClient(&redisClient.Options{
    Addr:     "localhost:6379",
    Password: "", // 无密码
    DB:       0,  // 默认DB
})

// 创建Redis存储
store := redis.InitStore(client,
    redis.StoreWithExpiration(30 * time.Minute),
    redis.StoreWithPrefix("mist:sess:"),
)

// 创建会话管理器
manager, err := session.NewManager(store, 1800)
if err != nil {
    // 处理错误
}
```

### 在HTTP处理器中使用会话

```go
import (
    "github.com/dormoron/mist"
)

func HandleLogin(ctx *mist.Context) {
    // 创建新会话
    sess, err := manager.InitSession(ctx)
    if err != nil {
        // 处理错误
        return
    }

    // 存储用户信息
    sess.Set(ctx.Request.Context(), "user_id", 12345)
    sess.Set(ctx.Request.Context(), "username", "testuser")
    sess.Save()

    // 响应登录成功
    ctx.JSON(200, map[string]interface{}{
        "message": "登录成功",
    })
}

func HandleProfile(ctx *mist.Context) {
    // 获取现有会话
    sess, err := manager.GetSession(ctx)
    if err != nil {
        // 会话不存在，重定向到登录页面
        ctx.Redirect(302, "/login")
        return
    }

    // 从会话中获取用户信息
    userID, err := sess.Get(ctx.Request.Context(), "user_id")
    if err != nil {
        // 处理错误
        return
    }

    username, err := sess.Get(ctx.Request.Context(), "username")
    if err != nil {
        // 处理错误
        return
    }

    // 刷新会话
    manager.RefreshSession(ctx)

    // 响应用户资料
    ctx.JSON(200, map[string]interface{}{
        "user_id":  userID,
        "username": username,
    })
}

func HandleLogout(ctx *mist.Context) {
    // 删除会话
    err := manager.RemoveSession(ctx)
    if err != nil {
        // 处理错误
        return
    }

    // 响应登出成功
    ctx.JSON(200, map[string]interface{}{
        "message": "登出成功",
    })
}
```

## 自定义配置

### 自定义Cookie选项

```go
import (
    "github.com/dormoron/mist/session/cookie"
)

// 创建自定义Cookie传播器
cookieProp := cookie.NewPropagator("custom_session",
    cookie.WithPath("/api"),
    cookie.WithDomain("example.com"),
    cookie.WithMaxAge(7200),
    cookie.WithSecure(true),
    cookie.WithHTTPOnly(true),
    cookie.WithSameSite(http.SameSiteStrictMode),
)

// 创建会话管理器并使用自定义传播器
manager := &session.Manager{
    Store:         store,
    Propagator:    cookieProp,
    CtxSessionKey: "session",
}
```

### 自定义内存存储选项

```go
// 创建具有自定义过期时间的内存存储
store := memory.InitStore(60 * time.Minute)
```

### 自定义Redis存储选项

```go
// 创建具有自定义选项的Redis存储
store := redis.InitStore(client,
    redis.StoreWithExpiration(60 * time.Minute),
    redis.StoreWithPrefix("myapp:session:"),
)
```

## 最佳实践

1. **安全设置**: 始终设置Cookie为HttpOnly和Secure（在HTTPS环境中）。

2. **适当的过期时间**: 根据应用的安全需求设置合适的会话过期时间。

3. **定期垃圾回收**: 启用自动垃圾回收以防止内存泄漏。

4. **会话数据**: 尽量只存储必要的数据在会话中，大型数据应存储在数据库中。

5. **错误处理**: 妥善处理所有会话操作中可能出现的错误。

## 类型和接口

会话管理模块定义了以下主要接口：

- `session.Store`: 会话数据存储接口
- `session.Session`: 单个会话接口
- `session.Propagator`: 会话ID传播接口

详细文档可参考每个接口的Go文档。

## 性能考虑

- 内存存储适用于单机应用或较小规模的应用
- Redis存储适用于需要会话共享的分布式应用
- 会话数据应尽量保持简洁，避免存储大量数据 