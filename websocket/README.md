# Mist WebSocket模块

Mist WebSocket模块提供了在Mist框架中轻松集成WebSocket功能的能力，支持实时通信应用如聊天、通知和游戏。

## 主要特性

- 基于[gorilla/websocket](https://github.com/gorilla/websocket)构建的高性能WebSocket实现
- 优雅的连接管理和自动清理机制
- 内置心跳检测和保活机制
- 支持房间/频道系统，便于构建分组通信
- 丰富的广播API支持不同的消息发送模式
- 通过Hub实现中心化消息处理
- 完整的错误处理和超时控制
- 丰富的示例代码，包括聊天室和通知系统

## 快速开始

### 1. 基本用法：创建Echo服务器

```go
package main

import (
    "github.com/dormoron/mist"
    "github.com/dormoron/mist/websocket"
)

func main() {
    server := mist.InitHTTPServer()
    
    // 创建一个简单的Echo WebSocket处理程序
    server.GET("/ws/echo", websocket.WebSocket(nil, func(conn *websocket.Connection) {
        // 持续接收消息
        for {
            msg, err := conn.Receive()
            if err != nil {
                // 连接已关闭或出错
                return
            }
            
            // 简单地将消息回显给发送者
            if err := conn.Send(msg.Type, msg.Data); err != nil {
                return
            }
        }
    }))
    
    server.Start(":8080")
}
```

### 2. 高级用法：创建聊天室

```go
package main

import (
    "encoding/json"
    "time"
    
    "github.com/dormoron/mist"
    "github.com/dormoron/mist/websocket"
    "github.com/google/uuid"
)

func main() {
    server := mist.InitHTTPServer()
    
    // 创建WebSocket Hub管理连接和房间
    hub := websocket.NewHub()
    
    // 注册WebSocket路由
    server.GET("/ws/chat", websocket.WebSocket(nil, func(conn *websocket.Connection) {
        // 为连接生成唯一ID
        connID := uuid.New().String()
        
        // 注册连接到Hub
        hub.Register(connID, conn)
        
        // 持续处理消息...
        // 查看完整示例请参考examples.go中的ExampleChatServer函数
    }))
    
    server.Start(":8080")
}
```

## 核心组件

### Connection

表示单个WebSocket连接，提供消息发送和接收功能。

```go
// 发送文本消息
conn.SendText("Hello!")

// 发送二进制数据
conn.SendBinary([]byte{1, 2, 3})

// 接收消息
msg, err := conn.Receive()
```

### Hub

管理多个WebSocket连接，支持房间和广播功能。

```go
// 将连接注册到Hub
hub.Register(userID, conn)

// 将连接加入房间
hub.JoinRoom("chat", userID)

// 向房间广播消息
hub.BroadcastTextToRoom("chat", "有人加入了聊天室")

// 向所有连接广播消息
hub.BroadcastText("服务器维护通知")
```

### WebSocket助手函数

创建WebSocket处理函数，简化HTTP到WebSocket的升级过程。

```go
server.GET("/ws", websocket.WebSocket(config, func(conn *websocket.Connection) {
    // 处理WebSocket连接...
}))
```

## 配置选项

通过`Config`结构体自定义WebSocket行为：

```go
config := websocket.DefaultConfig()
config.WriteBufferSize = 8192
config.ReadBufferSize = 8192
config.MaxMessageSize = 1024 * 1024 // 1MB
config.PingInterval = 10 * time.Second
```

## 安全性考虑

- 默认配置已启用Origin检查，但在生产环境中应自定义`CheckOrigin`函数
- 建议使用TLS (wss://)保护WebSocket通信
- 对消息大小设置合理的限制(MaxMessageSize)以防止DoS攻击
- 实现认证机制验证连接请求

## 性能优化

- 连接池使用map实现，支持快速查找
- 读写操作采用goroutine隔离，避免阻塞
- 内置消息缓冲区，防止慢客户端影响性能
- 自动关闭不活跃的连接，避免资源泄漏

## 客户端兼容性

Mist WebSocket实现兼容所有支持WebSocket协议(RFC 6455)的客户端，包括：

- 现代浏览器(Chrome, Firefox, Safari, Edge)
- JavaScript客户端库(socket.io, WebSocket API)
- 移动应用WebSocket客户端
- 各种语言的WebSocket客户端库

## 完整示例

参见`examples.go`文件中的完整聊天室和通知系统示例。 