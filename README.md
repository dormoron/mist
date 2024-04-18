# Mist

Mist是一个使用Go语言编写的轻量级、高效的Web框架，它的设计目标是使Web服务的开发变得更加简单快捷。

## 特点

- 轻量级设计，易于理解和使用
- 优秀的路由性能
- 中间件支持，方便扩展
- 强大的错误处理能力

## 快速开始

安装Mist：

```bash
go get -u github.com/your_username/mist
```
创建一个简单的服务器实例：
``` go
package main

import (
    "github.com/your_username/mist"
    "net/http"
)

func main() {
    server := mist.InitHTTPServer()
    
    server.GET("/", func(ctx *Context) {
		ctx.RespJSON(http.StatusOK,"hello")
	})
	
    server.Start(":8080")
}
```
运行你的服务器：
```bash
go run main.go
```
打开浏览器并访问 http://localhost:8080 来查看效果。

## 贡献
欢迎任何形式的贡献，包括报告bug，提出新功能，以及直接向代码库提交代码。

## 许可证
Mist是在MIT许可下发行的。有关详细信息，请查阅 [LICENSE](LICENSE) 文件。

