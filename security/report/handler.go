package report

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

// 违规报告类型
const (
	ReportTypeCSP      = "csp"      // 内容安全策略违规
	ReportTypeXSS      = "xss"      // XSS过滤违规
	ReportTypeCOOP     = "coop"     // 跨源打开者策略违规
	ReportTypeCOEP     = "coep"     // 跨源嵌入者策略违规
	ReportTypeCORP     = "corp"     // 跨源资源策略违规
	ReportTypeDocument = "document" // 文档策略违规
	ReportTypeHSTS     = "hsts"     // HSTS违规
	ReportTypeExpectCT = "expectct" // Expect-CT违规
	ReportTypeHPKP     = "hpkp"     // HPKP违规
	ReportTypeFeature  = "feature"  // 功能违规
	ReportTypeNEL      = "nel"      // NEL违规
)

// SecurityReport 表示通用安全报告结构
type SecurityReport struct {
	Type        string                 `json:"type"`               // 报告类型
	Time        time.Time              `json:"time"`               // 收到时间
	RawData     json.RawMessage        `json:"raw_data"`           // 原始JSON数据
	ReportData  map[string]interface{} `json:"report_data"`        // 解析后的报告数据
	UserAgent   string                 `json:"user_agent"`         // 用户代理
	IPAddress   string                 `json:"ip_address"`         // IP地址
	BlockedURI  string                 `json:"blocked_uri"`        // 被阻止的URI
	ViolatedDir string                 `json:"violated_directive"` // 违反的指令
	Severity    int                    `json:"severity"`           // 严重程度 (1-5)
}

// Handler 报告处理器接口
type Handler interface {
	// HandleReport 处理安全报告
	HandleReport(r *SecurityReport) error

	// GetRecentReports 获取最近的报告
	GetRecentReports(limit int) ([]*SecurityReport, error)

	// GetReportsByType 获取指定类型的报告
	GetReportsByType(reportType string, limit int) ([]*SecurityReport, error)

	// GetReportsSummary 获取报告摘要
	GetReportsSummary() (map[string]int, error)
}

// MemoryHandler 内存报告处理器实现
type MemoryHandler struct {
	reports    []*SecurityReport
	maxReports int
	mu         sync.RWMutex
}

// NewMemoryHandler 创建新的内存报告处理器
func NewMemoryHandler(maxReports int) *MemoryHandler {
	if maxReports <= 0 {
		maxReports = 1000 // 默认存储1000条报告
	}

	return &MemoryHandler{
		reports:    make([]*SecurityReport, 0, maxReports),
		maxReports: maxReports,
	}
}

// HandleReport 实现Handler接口
func (h *MemoryHandler) HandleReport(r *SecurityReport) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果报告数量超过最大值，移除最旧的
	if len(h.reports) >= h.maxReports {
		h.reports = h.reports[1:]
	}

	// 添加新报告
	h.reports = append(h.reports, r)

	return nil
}

// GetRecentReports 实现Handler接口
func (h *MemoryHandler) GetRecentReports(limit int) ([]*SecurityReport, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.reports) {
		limit = len(h.reports)
	}

	result := make([]*SecurityReport, limit)
	// 获取最近的报告(从末尾开始)
	startIdx := len(h.reports) - limit
	copy(result, h.reports[startIdx:])

	return result, nil
}

// GetReportsByType 实现Handler接口
func (h *MemoryHandler) GetReportsByType(reportType string, limit int) ([]*SecurityReport, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*SecurityReport, 0)

	// 从最新的报告开始过滤
	for i := len(h.reports) - 1; i >= 0 && len(result) < limit; i-- {
		if h.reports[i].Type == reportType {
			result = append(result, h.reports[i])
		}
	}

	return result, nil
}

// GetReportsSummary 实现Handler接口
func (h *MemoryHandler) GetReportsSummary() (map[string]int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	summary := make(map[string]int)

	for _, report := range h.reports {
		summary[report.Type]++
	}

	return summary, nil
}

// 报告解析器

// ParseCSPReport 解析CSP违规报告
func ParseCSPReport(r *http.Request) (*SecurityReport, error) {
	var rawReport struct {
		CSPReport map[string]interface{} `json:"csp-report"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&rawReport); err != nil {
		return nil, fmt.Errorf("failed to decode CSP report: %v", err)
	}

	// 提取相关数据
	blockedURI := ""
	violatedDirective := ""

	if blockVal, ok := rawReport.CSPReport["blocked-uri"]; ok {
		if blockStr, ok := blockVal.(string); ok {
			blockedURI = blockStr
		}
	}

	if directiveVal, ok := rawReport.CSPReport["violated-directive"]; ok {
		if directiveStr, ok := directiveVal.(string); ok {
			violatedDirective = directiveStr
		}
	}

	// 计算严重程度
	severity := calculateCSPSeverity(violatedDirective, blockedURI)

	// 将原始数据编码为JSON
	rawJSON, _ := json.Marshal(rawReport)

	report := &SecurityReport{
		Type:        ReportTypeCSP,
		Time:        time.Now(),
		RawData:     rawJSON,
		ReportData:  rawReport.CSPReport,
		UserAgent:   r.UserAgent(),
		IPAddress:   getIPAddress(r),
		BlockedURI:  blockedURI,
		ViolatedDir: violatedDirective,
		Severity:    severity,
	}

	return report, nil
}

// ParseXSSReport 解析XSS过滤器报告
func ParseXSSReport(r *http.Request) (*SecurityReport, error) {
	var rawReport map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&rawReport); err != nil {
		return nil, fmt.Errorf("failed to decode XSS report: %v", err)
	}

	// 将原始数据编码为JSON
	rawJSON, _ := json.Marshal(rawReport)

	// XSS报告总是较严重的
	report := &SecurityReport{
		Type:       ReportTypeXSS,
		Time:       time.Now(),
		RawData:    rawJSON,
		ReportData: rawReport,
		UserAgent:  r.UserAgent(),
		IPAddress:  getIPAddress(r),
		Severity:   4, // XSS通常是高严重性
	}

	return report, nil
}

// ReportServer 处理安全报告的HTTP服务器组件
type ReportServer struct {
	handler Handler
}

// NewReportServer 创建新的报告服务器
func NewReportServer(handler Handler) *ReportServer {
	return &ReportServer{
		handler: handler,
	}
}

// ServeHTTP 实现http.Handler接口
func (s *ReportServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 验证请求方法
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var report *SecurityReport
	var err error

	// 根据路径确定报告类型
	switch r.URL.Path {
	case "/report/csp":
		report, err = ParseCSPReport(r)
	case "/report/xss":
		report, err = ParseXSSReport(r)
	default:
		http.Error(w, "Unknown report type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 处理报告
	if err := s.handler.HandleReport(report); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回204状态码(无内容)
	w.WriteHeader(http.StatusNoContent)
}

// 辅助函数

// calculateCSPSeverity 计算CSP违规的严重程度
func calculateCSPSeverity(directive, blockedURI string) int {
	// 根据指令和被阻止的URI评估严重程度

	// 脚本相关违规通常更严重
	if directive == "script-src" || directive == "script-src-elem" {
		if blockedURI == "eval" || blockedURI == "inline" {
			return 4 // 高严重性
		}
		return 3 // 中等严重性
	}

	// 对象和框架源的违规也较严重
	if directive == "object-src" || directive == "frame-src" || directive == "frame-ancestors" {
		return 3 // 中等严重性
	}

	// 样式或图像违规通常不太严重
	if directive == "style-src" || directive == "img-src" {
		return 2 // 低严重性
	}

	// 默认严重程度
	return 2
}

// getIPAddress 获取请求的IP地址
func getIPAddress(r *http.Request) string {
	// 尝试从X-Forwarded-For头获取
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		return forwardedFor
	}

	// 尝试从X-Real-IP头获取
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// 使用RemoteAddr
	return r.RemoteAddr
}

// CSPReport 内容安全策略违规报告
type CSPReport struct {
	CSPReport struct {
		DocumentURI        string `json:"document-uri"`
		Referrer           string `json:"referrer"`
		BlockedURI         string `json:"blocked-uri"`
		ViolatedDirective  string `json:"violated-directive"`
		EffectiveDirective string `json:"effective-directive"`
		OriginalPolicy     string `json:"original-policy"`
		Disposition        string `json:"disposition"`
		StatusCode         int    `json:"status-code"`
	} `json:"csp-report"`
}

// XSSReport XSS过滤违规报告
type XSSReport struct {
	BlockedURL   string `json:"blocked-url"`
	OriginalURL  string `json:"original-url"`
	SourceLine   string `json:"source-line"`
	ColumnNumber int    `json:"column-number"`
	LineNumber   int    `json:"line-number"`
}

// COOPReport 跨源打开者策略违规报告
type COOPReport struct {
	DocumentURL      string `json:"document-url"`
	Disposition      string `json:"disposition"`
	EffectivePolicy  string `json:"effective-policy"`
	BlockingDocument string `json:"blocking-document"`
}

// HandlerOptions 报告处理器中间件选项
type HandlerOptions struct {
	// 路径前缀
	PathPrefix string

	// 处理器映射 (报告类型 -> 自定义处理器)
	Handlers map[string]Handler

	// 默认处理器
	DefaultHandler Handler

	// 是否记录完整报告到日志
	LogFullReport bool
}

// DefaultHandlerOptions 默认报告处理器中间件选项
func DefaultHandlerOptions() HandlerOptions {
	return HandlerOptions{
		PathPrefix:     "/api/security/report",
		Handlers:       make(map[string]Handler),
		DefaultHandler: NewMemoryHandler(1000),
		LogFullReport:  false,
	}
}

// NewReportHandlerMiddleware 创建新的安全报告处理中间件
func NewReportHandlerMiddleware(opts HandlerOptions) mist.Middleware {
	if opts.PathPrefix == "" {
		opts.PathPrefix = "/api/security/report"
	}

	if opts.DefaultHandler == nil {
		opts.DefaultHandler = NewMemoryHandler(1000)
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			path := ctx.Request.URL.Path

			// 检查是否是报告请求
			if ctx.Request.Method == http.MethodPost && path != "" &&
				(path == opts.PathPrefix || path == opts.PathPrefix+"/") {
				handleSecurityReport(ctx, opts)
				return
			}

			// 检查特定类型的报告端点
			for reportType := range opts.Handlers {
				reportPath := fmt.Sprintf("%s/%s", opts.PathPrefix, reportType)
				if path == reportPath {
					handleSecurityReport(ctx, opts)
					return
				}
			}

			next(ctx)
		}
	}
}

// handleSecurityReport 处理安全违规报告
func handleSecurityReport(ctx *mist.Context, opts HandlerOptions) {
	// 读取请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 确定报告类型
	reportType := ReportTypeCSP // 默认CSP报告

	// 从路径中提取类型
	path := ctx.Request.URL.Path
	if path != opts.PathPrefix && path != opts.PathPrefix+"/" {
		// 路径格式为 /api/security/report/{type}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 {
			reportType = parts[len(parts)-1]
		}
	}

	// 创建报告
	report := &SecurityReport{
		Type:       reportType,
		Time:       time.Now(),
		RawData:    body,
		ReportData: make(map[string]interface{}),
		UserAgent:  ctx.Request.UserAgent(),
		IPAddress:  ctx.ClientIP(),
	}

	// 查找适当的处理器
	var handler Handler
	if h, exists := opts.Handlers[reportType]; exists {
		handler = h
	} else {
		handler = opts.DefaultHandler
	}

	// 处理报告
	if handler != nil {
		handler.HandleReport(report)
	}

	// 记录详细报告（如果启用）
	if opts.LogFullReport {
		log.Printf("安全违规报告详情: %s", string(body))
	}

	// 返回成功状态码
	ctx.AbortWithStatus(http.StatusNoContent)
}

// 创建报告处理中间件的便捷函数
func WithDefaultReportHandler() mist.Middleware {
	return NewReportHandlerMiddleware(DefaultHandlerOptions())
}

// WithCSPReportHandler 只处理CSP报告的中间件
func WithCSPReportHandler() mist.Middleware {
	opts := DefaultHandlerOptions()
	opts.PathPrefix = "/api/security/report/csp"
	return NewReportHandlerMiddleware(opts)
}

// 提供默认的ReportTo JSON配置
func DefaultReportToJSON(endpoint string) string {
	if endpoint == "" {
		endpoint = "/api/security/report"
	}
	return fmt.Sprintf(`{"endpoints":[{"url":"%s"}],"group":"default","max_age":10886400}`, endpoint)
}
