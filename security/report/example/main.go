package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dormoron/mist/security/report"
)

func main() {
	// 创建一个内存报告处理器，最多保存100条报告
	handler := report.NewMemoryHandler(100)

	// 创建报告服务器
	reportServer := report.NewReportServer(handler)

	// 为各种安全报告设置处理路由
	http.Handle("/report/csp", reportServer)
	http.Handle("/report/xss", reportServer)
	http.Handle("/report/hpkp", reportServer)
	http.Handle("/report/feature", reportServer)
	http.Handle("/report/nel", reportServer)
	http.Handle("/report/coep", reportServer)
	http.Handle("/report/corp", reportServer)
	http.Handle("/report/coop", reportServer)

	// 添加一个API端点来获取报告摘要
	http.HandleFunc("/api/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		summary, err := handler.GetReportsSummary()
		if err != nil {
			http.Error(w, "获取报告摘要失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{\n")
		first := true
		for reportType, count := range summary {
			if !first {
				fmt.Fprintf(w, ",\n")
			}
			fmt.Fprintf(w, "  %q: %d", reportType, count)
			first = false
		}
		fmt.Fprintf(w, "\n}\n")
	})

	// 添加一个API端点来获取最近的报告
	http.HandleFunc("/api/reports/recent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		limit := 10 // 默认限制为10条
		reports, err := handler.GetRecentReports(limit)
		if err != nil {
			http.Error(w, "获取最近报告失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[\n")
		for i, r := range reports {
			if i > 0 {
				fmt.Fprintf(w, ",\n")
			}
			fmt.Fprintf(w, "  {\n")
			fmt.Fprintf(w, "    \"type\": %q,\n", r.Type)
			fmt.Fprintf(w, "    \"time\": %q,\n", r.Time.Format(time.RFC3339))
			fmt.Fprintf(w, "    \"user_agent\": %q,\n", r.UserAgent)
			fmt.Fprintf(w, "    \"ip_address\": %q,\n", r.IPAddress)
			if r.BlockedURI != "" {
				fmt.Fprintf(w, "    \"blocked_uri\": %q,\n", r.BlockedURI)
			}
			if r.ViolatedDir != "" {
				fmt.Fprintf(w, "    \"violated_directive\": %q,\n", r.ViolatedDir)
			}
			fmt.Fprintf(w, "    \"severity\": %d\n", r.Severity)
			fmt.Fprintf(w, "  }")
		}
		fmt.Fprintf(w, "\n]\n")
	})

	// 添加一个API端点来按类型获取报告
	http.HandleFunc("/api/reports/type/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		reportType := r.URL.Path[len("/api/reports/type/"):]
		if reportType == "" {
			http.Error(w, "必须指定报告类型", http.StatusBadRequest)
			return
		}

		limit := 10 // 默认限制为10条
		reports, err := handler.GetReportsByType(reportType, limit)
		if err != nil {
			http.Error(w, "获取报告失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[\n")
		for i, r := range reports {
			if i > 0 {
				fmt.Fprintf(w, ",\n")
			}
			fmt.Fprintf(w, "  {\n")
			fmt.Fprintf(w, "    \"time\": %q,\n", r.Time.Format(time.RFC3339))
			fmt.Fprintf(w, "    \"user_agent\": %q,\n", r.UserAgent)
			fmt.Fprintf(w, "    \"ip_address\": %q,\n", r.IPAddress)
			if r.BlockedURI != "" {
				fmt.Fprintf(w, "    \"blocked_uri\": %q,\n", r.BlockedURI)
			}
			if r.ViolatedDir != "" {
				fmt.Fprintf(w, "    \"violated_directive\": %q,\n", r.ViolatedDir)
			}
			fmt.Fprintf(w, "    \"severity\": %d\n", r.Severity)
			fmt.Fprintf(w, "  }")
		}
		fmt.Fprintf(w, "\n]\n")
	})

	// 启动HTTP服务器
	log.Println("安全报告服务器启动在 http://localhost:8080")
	log.Println("- 接收CSP报告: POST /report/csp")
	log.Println("- 接收XSS报告: POST /report/xss")
	log.Println("- 查看报告摘要: GET /api/reports/summary")
	log.Println("- 查看最近报告: GET /api/reports/recent")
	log.Println("- 按类型查看报告: GET /api/reports/type/{type}")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
