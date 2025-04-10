# 安全报告处理模块

这个模块提供了一个全面的安全报告处理系统，用于接收、存储和分析各种安全相关的报告，如CSP违规、XSS尝试等。

## 支持的报告类型

- **CSP (Content Security Policy)**: 内容安全策略违规报告
- **XSS (Cross-Site Scripting)**: 跨站脚本尝试
- **HPKP (HTTP Public Key Pinning)**: HTTP公钥固定违规
- **COEP (Cross-Origin Embedder Policy)**: 跨源嵌入策略违规
- **CORP (Cross-Origin Resource Policy)**: 跨源资源策略违规
- **COOP (Cross-Origin Opener Policy)**: 跨源打开者策略违规
- **Feature Policy**: 特性策略违规
- **NEL (Network Error Logging)**: 网络错误日志

## 主要组件

1. **SecurityReport**: 表示单个安全报告的结构体，包含类型、时间、原始数据等信息。
2. **Handler**: 处理和管理报告的接口。
3. **MemoryHandler**: `Handler`接口的内存实现，用于存储和检索报告。
4. **ReportServer**: HTTP处理器，用于接收和处理安全报告。

## 使用示例

### 创建一个基本的安全报告服务器

```go
package main

import (
	"log"
	"net/http"
	"security/report"
)

func main() {
	// 创建一个内存报告处理器，最多保存100条报告
	handler := report.NewMemoryHandler(100)

	// 创建报告服务器
	reportServer := report.NewReportServer(handler)

	// 为各种安全报告设置处理路由
	http.Handle("/report/csp", reportServer)
	http.Handle("/report/xss", reportServer)
	
	log.Println("安全报告服务器启动在 http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
```

### 为前端启用CSP报告

在您的HTML页面或HTTP响应头中添加CSP策略和报告URI：

```html
<meta http-equiv="Content-Security-Policy" content="default-src 'self'; report-uri /report/csp">
```

或者作为HTTP头：

```
Content-Security-Policy: default-src 'self'; report-uri /report/csp
```

### 查询报告数据

```go
// 获取最近10条报告
reports, err := handler.GetRecentReports(10)

// 获取CSP类型的报告
cspReports, err := handler.GetReportsByType(report.ReportTypeCSP, 10)

// 获取报告摘要（各类型的报告数量）
summary, err := handler.GetReportsSummary()
```

## 实现自定义处理器

您可以通过实现`Handler`接口来创建自定义的报告处理器，例如将报告保存到数据库中：

```go
type DatabaseHandler struct {
	db *sql.DB
}

func (h *DatabaseHandler) HandleReport(report *SecurityReport) error {
	// 将报告保存到数据库
}

func (h *DatabaseHandler) GetRecentReports(limit int) ([]*SecurityReport, error) {
	// 从数据库检索最近的报告
}

// 实现其他方法...
```

## 安全最佳实践

1. **限制报告端点的请求大小**：防止DOS攻击。
2. **验证报告来源**：考虑添加某种形式的API密钥或其他身份验证机制。
3. **定期清理旧报告**：设置报告老化策略，防止报告积累过多。
4. **监控异常活动**：如果短时间内收到大量报告，可能表明正在发生攻击。 