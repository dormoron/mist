package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dormoron/mist"
)

// 日志中间件
func loggerMiddleware(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		start := time.Now()
		method := ctx.Request.Method
		path := ctx.Request.URL.Path

		// 执行下一个处理函数
		next(ctx)

		// 计算处理时间
		duration := time.Since(start)
		// 获取状态码
		statusCode := ctx.RespStatusCode

		// 打印日志
		fmt.Printf("[%s] %s %s %d %v\n",
			time.Now().Format("2006-01-02 15:04:05"),
			method, path, statusCode, duration)
	}
}

// 鉴权中间件
func authMiddleware(next mist.HandleFunc) mist.HandleFunc {
	return func(ctx *mist.Context) {
		authHeader := ctx.Request.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			ctx.RespondWithJSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "未授权访问",
			})
			return
		}
		next(ctx)
	}
}

func main() {
	// 初始化服务器
	server := mist.InitHTTPServer()

	// 注册全局中间件
	server.Use(loggerMiddleware)

	// 基本路由测试
	server.GET("/", func(ctx *mist.Context) {
		// 直接设置响应状态和数据
		ctx.RespStatusCode = http.StatusOK
		ctx.Header("Content-Type", "text/html")
		ctx.RespData = []byte("<h1>Mist HTTP/3 服务器</h1><p>路由测试首页</p>")
	})

	// JSON API测试
	server.GET("/api/test", func(ctx *mist.Context) {
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"message": "HTTP/3服务器测试成功",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// RESTful API路由测试
	apiGroup := "/api/v1"

	// 获取用户列表
	server.GET(apiGroup+"/users", func(ctx *mist.Context) {
		users := []map[string]interface{}{
			{"id": 1, "name": "用户1", "email": "user1@example.com"},
			{"id": 2, "name": "用户2", "email": "user2@example.com"},
		}
		ctx.RespondWithJSON(http.StatusOK, users)
	})

	// 获取特定用户
	server.GET(apiGroup+"/users/{id}", func(ctx *mist.Context) {
		id := ctx.PathParams["id"]
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"id":    id,
			"name":  "用户" + id,
			"email": "user" + id + "@example.com",
		})
	})

	// 创建用户（需要认证）
	server.POST(apiGroup+"/users", func(ctx *mist.Context) {
		var userData map[string]interface{}
		if err := ctx.BindJSON(&userData); err != nil {
			ctx.RespondWithJSON(http.StatusBadRequest, map[string]interface{}{
				"error": "无效的JSON数据",
			})
			return
		}

		// 模拟创建用户
		userData["id"] = 100
		userData["created_at"] = time.Now().Format(time.RFC3339)

		ctx.RespondWithJSON(http.StatusCreated, userData)
	}, authMiddleware)

	// 正则路由测试
	server.GET("/posts/{year:\\d{4}}/{month:\\d{2}}", func(ctx *mist.Context) {
		year := ctx.PathParams["year"]
		month := ctx.PathParams["month"]

		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"year":  year,
			"month": month,
			"posts": []string{"文章1", "文章2", "文章3"},
		})
	})

	// 通配符路由
	server.GET("/files/{*filepath}", func(ctx *mist.Context) {
		filepath := ctx.PathParams["filepath"]
		ctx.RespondWithJSON(http.StatusOK, map[string]interface{}{
			"filepath": filepath,
			"message":  "请求的文件路径: " + filepath,
		})
	})

	// 生成临时的自签名证书（仅用于测试）
	certFile, keyFile, err := generateSelfSignedCert()
	if err != nil {
		log.Fatalf("生成证书失败: %v", err)
	}
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	// 输出测试路由信息
	fmt.Println("==== 路由测试指南 ====")
	fmt.Println("基本路由: https://localhost:8443/")
	fmt.Println("JSON API: https://localhost:8443/api/test")
	fmt.Println("用户列表: https://localhost:8443/api/v1/users")
	fmt.Println("单个用户: https://localhost:8443/api/v1/users/1")
	fmt.Println("创建用户(需认证): POST https://localhost:8443/api/v1/users")
	fmt.Println("   需要添加请求头: Authorization: Bearer test-token")
	fmt.Println("   请求体: {\"name\":\"新用户\",\"email\":\"new@example.com\"}")
	fmt.Println("正则路由: https://localhost:8443/posts/2023/04")
	fmt.Println("通配符路由: https://localhost:8443/files/path/to/some/file.txt")
	fmt.Println("======================")

	// 启动HTTP/3服务器（非阻塞）
	go func() {
		fmt.Printf("启动HTTP/3服务器于 https://localhost:8443\n")
		fmt.Printf("注意：这是自签名证书，浏览器会显示安全警告\n")

		if err := server.StartHTTP3(":8443", certFile, keyFile); err != nil {
			log.Printf("HTTP/3服务器错误: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("正在关闭服务器...")
}

// 生成自签名证书（仅用于测试）
func generateSelfSignedCert() (string, string, error) {
	// 这里仅为测试目的，使用预先生成的自签名证书
	tempDir, err := os.MkdirTemp("", "http3-test")
	if err != nil {
		return "", "", err
	}

	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	// 证书数据（仅用于测试，请勿在生产环境中使用）
	certData := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	keyData := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`

	if err := os.WriteFile(certFile, []byte(certData), 0644); err != nil {
		return "", "", err
	}

	if err := os.WriteFile(keyFile, []byte(keyData), 0644); err != nil {
		return "", "", err
	}

	return certFile, keyFile, nil
}
