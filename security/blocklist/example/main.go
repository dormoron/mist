package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dormoron/mist/security/blocklist"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	// 解析命令行参数
	port := flag.Int("port", 8080, "服务器端口")
	flag.Parse()

	// 创建IP黑名单管理器
	blocklistManager := blocklist.NewManager(
		blocklist.WithMaxFailedAttempts(3),
		blocklist.WithBlockDuration(5*time.Minute),
		blocklist.WithWhitelistIPs([]string{"127.0.0.1"}),
		blocklist.WithOnBlocked(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "您的IP因多次失败的尝试已被暂时封禁，请稍后再试",
			})
		}),
	)

	// 创建路由器
	mux := http.NewServeMux()

	// 登录接口
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ip := getClientIP(r)

		// 如果IP已被封禁，直接返回错误
		if blocklistManager.IsBlocked(ip) {
			blocklistManager.RecordFailure(ip) // 增加失败次数
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "您的IP已被封禁，请稍后再试",
			})
			return
		}

		// 解析登录请求
		var req LoginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			blocklistManager.RecordFailure(ip) // 增加失败次数
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "无效的请求格式",
			})
			return
		}

		// 模拟身份验证（简化示例）
		if req.Username == "admin" && req.Password == "password" {
			// 登录成功，重置失败计数
			blocklistManager.RecordSuccess(ip)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Response{
				Status:  "success",
				Message: "登录成功",
			})
			return
		}

		// 登录失败，记录失败尝试
		isBlocked := blocklistManager.RecordFailure(ip)
		w.Header().Set("Content-Type", "application/json")

		if isBlocked {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "您的IP因多次失败的尝试已被封禁，请稍后再试",
			})
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "用户名或密码错误",
			})
		}
	})

	// 受保护的API接口
	protectedAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "这是受保护的API接口",
		})
	})

	// 应用IP黑名单中间件到受保护的API
	mux.Handle("/api/protected", blocklistManager.Middleware()(protectedAPI))

	// 管理接口 - 手动解除IP封禁（通常需要管理员权限）
	mux.HandleFunc("/api/admin/unblock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// 这里应该添加管理员验证逻辑
		// 简化示例，直接从请求中获取IP
		ipToUnblock := r.URL.Query().Get("ip")
		if ipToUnblock == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "请指定要解除封禁的IP",
			})
			return
		}

		blocklistManager.UnblockIP(ipToUnblock)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: fmt.Sprintf("IP %s 已解除封禁", ipToUnblock),
		})
	})

	// 状态检查接口
	mux.HandleFunc("/api/admin/status", func(w http.ResponseWriter, r *http.Request) {
		ipToCheck := r.URL.Query().Get("ip")
		if ipToCheck == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{
				Status:  "error",
				Message: "请指定要检查的IP",
			})
			return
		}

		// 检查IP状态
		isBlocked := blocklistManager.IsBlocked(ipToCheck)

		status := "正常"
		if isBlocked {
			status = "已封禁"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"ip":        ipToCheck,
			"isBlocked": isBlocked,
			"state":     status,
		})
	})

	// 启动服务器
	serverAddr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, mux))
}

// 从请求中获取客户端IP
func getClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For头获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// 尝试从X-Real-IP头获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 否则使用RemoteAddr
	return r.RemoteAddr
}
