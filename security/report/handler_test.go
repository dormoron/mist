package report

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMemoryHandler(t *testing.T) {
	handler := NewMemoryHandler(10)

	// 创建测试报告
	report := &SecurityReport{
		Type:        ReportTypeCSP,
		Time:        time.Now(),
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0 (TestAgent)",
		BlockedURI:  "https://evil.example.com/script.js",
		ViolatedDir: "script-src",
		Severity:    3,
	}

	// 测试处理报告
	if err := handler.HandleReport(report); err != nil {
		t.Errorf("处理报告失败: %v", err)
	}

	// 测试获取报告
	reports, err := handler.GetRecentReports(5)
	if err != nil {
		t.Errorf("获取报告失败: %v", err)
	}

	if len(reports) != 1 {
		t.Errorf("期望获取1个报告, 实际获取了 %d 个", len(reports))
	}

	// 测试按类型获取报告
	cspReports, err := handler.GetReportsByType(ReportTypeCSP, 5)
	if err != nil {
		t.Errorf("按类型获取报告失败: %v", err)
	}

	if len(cspReports) != 1 {
		t.Errorf("期望获取1个CSP报告, 实际获取了 %d 个", len(cspReports))
	}

	// 测试获取摘要
	summary, err := handler.GetReportsSummary()
	if err != nil {
		t.Errorf("获取摘要失败: %v", err)
	}

	if summary[ReportTypeCSP] != 1 {
		t.Errorf("期望CSP报告计数为1, 实际为 %d", summary[ReportTypeCSP])
	}

	// 测试最大报告数
	for i := 0; i < 15; i++ {
		newReport := &SecurityReport{
			Type:        ReportTypeXSS,
			Time:        time.Now(),
			IPAddress:   "192.168.1.2",
			UserAgent:   "Mozilla/5.0 (TestAgent)",
			BlockedURI:  "https://evil.example.com/xss.js",
			ViolatedDir: "script-src",
			Severity:    4,
		}
		handler.HandleReport(newReport)
	}

	allReports, _ := handler.GetRecentReports(20)
	if len(allReports) > 10 {
		t.Errorf("报告数量超过最大限制: %d > 10", len(allReports))
	}
}

func TestParseCSPReport(t *testing.T) {
	// 创建一个模拟的CSP报告
	cspReport := `{
		"csp-report": {
			"document-uri": "https://example.com/page.html",
			"referrer": "https://google.com/",
			"violated-directive": "script-src",
			"effective-directive": "script-src",
			"original-policy": "script-src 'self'; report-uri /report/csp",
			"blocked-uri": "https://evil.com/malicious.js",
			"status-code": 0,
			"source-file": "https://example.com/page.html"
		}
	}`

	// 创建请求
	req := httptest.NewRequest(http.MethodPost, "/report/csp", bytes.NewBufferString(cspReport))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	req.Header.Set("X-Forwarded-For", "203.0.113.195")

	// 解析报告
	report, err := ParseCSPReport(req)
	if err != nil {
		t.Fatalf("解析CSP报告失败: %v", err)
	}

	// 验证报告字段
	if report.Type != ReportTypeCSP {
		t.Errorf("报告类型错误: 期望 %s, 得到 %s", ReportTypeCSP, report.Type)
	}

	if report.BlockedURI != "https://evil.com/malicious.js" {
		t.Errorf("被阻止的URI错误: 期望 %s, 得到 %s", "https://evil.com/malicious.js", report.BlockedURI)
	}

	if report.ViolatedDir != "script-src" {
		t.Errorf("违反的指令错误: 期望 %s, 得到 %s", "script-src", report.ViolatedDir)
	}

	if report.IPAddress != "203.0.113.195" {
		t.Errorf("IP地址错误: 期望 %s, 得到 %s", "203.0.113.195", report.IPAddress)
	}

	if report.UserAgent != "Mozilla/5.0 (Test)" {
		t.Errorf("User-Agent错误: 期望 %s, 得到 %s", "Mozilla/5.0 (Test)", report.UserAgent)
	}

	// 验证严重程度
	if report.Severity != 3 {
		t.Errorf("严重程度错误: 期望 %d, 得到 %d", 3, report.Severity)
	}
}

func TestParseXSSReport(t *testing.T) {
	// 创建一个模拟的XSS报告
	xssReport := `{
		"blocked-url": "https://example.com/page.html?q=<script>alert(1)</script>",
		"source-url": "https://example.com/search",
		"filter-type": "xss"
	}`

	// 创建请求
	req := httptest.NewRequest(http.MethodPost, "/report/xss", bytes.NewBufferString(xssReport))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	req.Header.Set("X-Real-IP", "203.0.113.195")

	// 解析报告
	report, err := ParseXSSReport(req)
	if err != nil {
		t.Fatalf("解析XSS报告失败: %v", err)
	}

	// 验证报告字段
	if report.Type != ReportTypeXSS {
		t.Errorf("报告类型错误: 期望 %s, 得到 %s", ReportTypeXSS, report.Type)
	}

	if report.IPAddress != "203.0.113.195" {
		t.Errorf("IP地址错误: 期望 %s, 得到 %s", "203.0.113.195", report.IPAddress)
	}

	if report.UserAgent != "Mozilla/5.0 (Test)" {
		t.Errorf("User-Agent错误: 期望 %s, 得到 %s", "Mozilla/5.0 (Test)", report.UserAgent)
	}

	// 验证严重程度(XSS应该是4)
	if report.Severity != 4 {
		t.Errorf("严重程度错误: 期望 %d, 得到 %d", 4, report.Severity)
	}

	// 验证原始数据已被正确解析
	var rawData map[string]interface{}
	if err := json.Unmarshal(report.RawData, &rawData); err != nil {
		t.Errorf("原始数据解析失败: %v", err)
	}

	if rawData["filter-type"] != "xss" {
		t.Errorf("报告数据错误: 期望filter-type为'xss', 得到 %v", rawData["filter-type"])
	}
}

func TestReportServer(t *testing.T) {
	// 创建内存处理器
	handler := NewMemoryHandler(100)

	// 创建报告服务器
	server := NewReportServer(handler)

	// 创建一个HTTP测试服务器
	testServer := httptest.NewServer(server)
	defer testServer.Close()

	// 构建CSP报告
	cspReport := `{
		"csp-report": {
			"document-uri": "https://example.com/page.html",
			"referrer": "https://google.com/",
			"violated-directive": "script-src",
			"effective-directive": "script-src",
			"original-policy": "script-src 'self'; report-uri /report/csp",
			"blocked-uri": "https://evil.com/malicious.js",
			"status-code": 0
		}
	}`

	// 发送CSP报告请求
	resp, err := http.Post(
		testServer.URL+"/report/csp",
		"application/json",
		strings.NewReader(cspReport),
	)
	if err != nil {
		t.Fatalf("发送CSP报告请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("响应状态码错误: 期望 %d, 得到 %d", http.StatusNoContent, resp.StatusCode)
	}

	// 验证报告是否被存储
	reports, _ := handler.GetReportsByType(ReportTypeCSP, 10)
	if len(reports) != 1 {
		t.Errorf("期望存储1个CSP报告, 实际存储了 %d 个", len(reports))
	}

	// 构建XSS报告
	xssReport := `{
		"blocked-url": "https://example.com/page.html?q=<script>alert(1)</script>",
		"source-url": "https://example.com/search",
		"filter-type": "xss"
	}`

	// 发送XSS报告请求
	resp, err = http.Post(
		testServer.URL+"/report/xss",
		"application/json",
		strings.NewReader(xssReport),
	)
	if err != nil {
		t.Fatalf("发送XSS报告请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("响应状态码错误: 期望 %d, 得到 %d", http.StatusNoContent, resp.StatusCode)
	}

	// 验证报告是否被存储
	reports, _ = handler.GetReportsByType(ReportTypeXSS, 10)
	if len(reports) != 1 {
		t.Errorf("期望存储1个XSS报告, 实际存储了 %d 个", len(reports))
	}

	// 测试不支持的路径
	resp, err = http.Post(
		testServer.URL+"/report/unknown",
		"application/json",
		strings.NewReader("{}"),
	)
	if err != nil {
		t.Fatalf("发送未知报告请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码(应为400 Bad Request)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("响应状态码错误: 期望 %d, 得到 %d", http.StatusBadRequest, resp.StatusCode)
	}
}
