# IP黑名单（Blocklist）示例

这个示例展示了如何使用Mist框架的IP黑名单（Blocklist）功能来保护你的API免受暴力破解攻击。

## 功能特点

- 基于失败尝试次数的IP封禁机制
- 可配置的封禁时长和最大失败次数
- IP白名单支持
- 自动清理过期记录
- 支持手动封禁和解除封禁IP
- 支持标准HTTP中间件和Mist框架中间件

## 运行示例

```bash
# 在默认端口8080上运行
go run main.go

# 在指定端口上运行
go run main.go -port=8888
```

## API接口说明

### 1. 登录接口

- **URL**: `/api/login`
- **方法**: `POST`
- **请求体**:
  ```json
  {
    "username": "admin",
    "password": "password"
  }
  ```
- **成功响应**:
  ```json
  {
    "status": "success",
    "message": "登录成功"
  }
  ```
- **失败响应**:
  ```json
  {
    "status": "error",
    "message": "用户名或密码错误"
  }
  ```
- **IP封禁响应**:
  ```json
  {
    "status": "error",
    "message": "您的IP因多次失败的尝试已被封禁，请稍后再试"
  }
  ```

### 2. 受保护的API接口

- **URL**: `/api/protected`
- **方法**: `GET`
- **成功响应**:
  ```json
  {
    "status": "success",
    "message": "这是受保护的API接口"
  }
  ```
- **IP封禁响应**:
  ```json
  {
    "status": "error",
    "message": "您的IP因多次失败的尝试已被封禁，请稍后再试"
  }
  ```

### 3. 解除IP封禁（管理接口）

- **URL**: `/api/admin/unblock?ip={ip地址}`
- **方法**: `POST`
- **成功响应**:
  ```json
  {
    "status": "success",
    "message": "IP xxx.xxx.xxx.xxx 已解除封禁"
  }
  ```

### 4. 检查IP状态（管理接口）

- **URL**: `/api/admin/status?ip={ip地址}`
- **方法**: `GET`
- **成功响应**:
  ```json
  {
    "status": "success",
    "ip": "xxx.xxx.xxx.xxx",
    "isBlocked": false,
    "state": "正常"
  }
  ```

## 测试示例

### 测试登录失败和IP封禁

```bash
# 使用错误的密码尝试登录（需要3次失败才会被封禁）
curl -X POST http://localhost:8080/api/login -d '{"username":"admin","password":"wrong"}'
curl -X POST http://localhost:8080/api/login -d '{"username":"admin","password":"wrong"}'
curl -X POST http://localhost:8080/api/login -d '{"username":"admin","password":"wrong"}'

# 第4次尝试将会收到封禁消息
curl -X POST http://localhost:8080/api/login -d '{"username":"admin","password":"wrong"}'

# 尝试访问受保护的API
curl http://localhost:8080/api/protected
```

### 解除IP封禁

```bash
# 解除本地IP的封禁
curl -X POST http://localhost:8080/api/admin/unblock?ip=127.0.0.1
```

### 检查IP状态

```bash
# 检查本地IP的状态
curl http://localhost:8080/api/admin/status?ip=127.0.0.1
```

## 在Mist框架中使用

此示例主要展示了在标准HTTP服务中使用IP黑名单功能，但Mist框架也提供了更简洁的集成方式：

```go
package main

import (
    "github.com/dormoron/mist"
    "github.com/dormoron/mist/security/blocklist"
    "github.com/dormoron/mist/security/blocklist/middleware"
    "time"
    "log"
    "net/http"
)

func main() {
    // 创建Mist应用
    app := mist.New()
    
    // 创建IP黑名单管理器
    blocklistManager := blocklist.NewManager(
        blocklist.WithMaxFailedAttempts(3),
        blocklist.WithBlockDuration(5*time.Minute),
    )
    
    // 使用默认配置
    // app.Use(middleware.New(blocklistManager))
    
    // 使用自定义封禁处理函数
    app.Use(middleware.New(
        blocklistManager,
        middleware.WithOnBlocked(func(ctx *mist.Context) {
            // 记录IP封禁事件
            log.Printf("IP %s 已被封禁", ctx.ClientIP())
            
            // 返回JSON响应
            ctx.RespondWithJSON(http.StatusForbidden, map[string]interface{}{
                "status":  "error",
                "message": "您的IP因多次失败的尝试已被暂时封禁，请稍后再试",
            })
        }),
    ))
    
    // 设置路由和处理函数...
    
    // 启动服务器
    app.Run(":8080")
}
```

## 自定义配置选项

### 标准HTTP中间件选项（用于blocklist.Manager.Middleware）

- `WithMaxFailedAttempts(max int)` - 设置最大失败尝试次数
- `WithBlockDuration(duration time.Duration)` - 设置封禁时长
- `WithClearInterval(interval time.Duration)` - 设置清理间隔时间
- `WithWhitelistIPs(ips []string)` - 设置IP白名单
- `WithOnBlocked(handler func(w http.ResponseWriter, r *http.Request))` - 设置封禁处理函数

### Mist框架中间件选项（用于middleware.New）

- `middleware.WithOnBlocked(handler func(*mist.Context))` - 设置Mist框架中的封禁处理函数 