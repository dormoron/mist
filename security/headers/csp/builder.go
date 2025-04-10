package csp

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// 哈希算法类型
const (
	HashAlgoSha256 = "sha256"
	HashAlgoSha384 = "sha384"
	HashAlgoSha512 = "sha512"
)

// CSP指令类型
const (
	// 获取指令
	DirectiveDefaultSrc    = "default-src"
	DirectiveScriptSrc     = "script-src"
	DirectiveScriptSrcElem = "script-src-elem"
	DirectiveScriptSrcAttr = "script-src-attr"
	DirectiveStyleSrc      = "style-src"
	DirectiveStyleSrcElem  = "style-src-elem"
	DirectiveStyleSrcAttr  = "style-src-attr"
	DirectiveImgSrc        = "img-src"
	DirectiveConnectSrc    = "connect-src"
	DirectiveFontSrc       = "font-src"
	DirectiveObjectSrc     = "object-src"
	DirectiveMediaSrc      = "media-src"
	DirectiveFrameSrc      = "frame-src"
	DirectiveWorkerSrc     = "worker-src"
	DirectiveManifestSrc   = "manifest-src"
	DirectiveChildSrc      = "child-src"
	DirectivePrefetchSrc   = "prefetch-src"

	// 文档指令
	DirectiveBaseURI        = "base-uri"
	DirectiveSandbox        = "sandbox"
	DirectiveFormAction     = "form-action"
	DirectiveFrameAncestors = "frame-ancestors"
	DirectiveNavigateTo     = "navigate-to"

	// 报告指令
	DirectiveReportURI = "report-uri"
	DirectiveReportTo  = "report-to"

	// 其他指令
	DirectiveRequireSriFor           = "require-sri-for"
	DirectiveUpgradeInsecureRequests = "upgrade-insecure-requests"
	DirectiveBlockAllMixedContent    = "block-all-mixed-content"
	DirectiveTrustedTypes            = "trusted-types"
	DirectiveTreatAsPublicAddress    = "treat-as-public-address"
)

// CSPBuilder 用于构建内容安全策略的生成器
type Builder struct {
	directives     map[string][]string // 指令映射
	requireSRI     map[string]bool     // 需要SRI的指令
	nonce          string              // nonce值
	reportOnly     bool                // 是否仅报告模式
	hashCache      map[string]string   // 哈希缓存
	reportEndpoint string              // 报告端点
}

// NewBuilder 创建新的CSP生成器
func NewBuilder() *Builder {
	return &Builder{
		directives: make(map[string][]string),
		requireSRI: make(map[string]bool),
		hashCache:  make(map[string]string),
	}
}

// SetNonce 设置全局nonce值
func (b *Builder) SetNonce(nonce string) *Builder {
	b.nonce = nonce
	return b
}

// SetReportOnly 设置CSP为仅报告模式
func (b *Builder) SetReportOnly(reportOnly bool) *Builder {
	b.reportOnly = reportOnly
	return b
}

// SetReportEndpoint 设置违规报告端点
func (b *Builder) SetReportEndpoint(endpoint string) *Builder {
	b.reportEndpoint = endpoint
	return b
}

// Add 添加内容安全策略指令
func (b *Builder) Add(directive string, values ...string) *Builder {
	b.directives[directive] = append(b.directives[directive], values...)
	return b
}

// AddHash 添加内容哈希
func (b *Builder) AddHash(directive, content, algorithm string) *Builder {
	hashValue, ok := b.hashCache[content]
	if !ok {
		hashValue = generateHash(content, algorithm)
		b.hashCache[content] = hashValue
	}
	return b.Add(directive, fmt.Sprintf("'%s-%s'", algorithm, hashValue))
}

// AddNonce 添加nonce到指定指令
func (b *Builder) AddNonce(directive string) *Builder {
	if b.nonce != "" {
		return b.Add(directive, fmt.Sprintf("'nonce-%s'", b.nonce))
	}
	return b
}

// AddNonceToScriptAndStyle 添加nonce到脚本和样式指令
func (b *Builder) AddNonceToScriptAndStyle() *Builder {
	if b.nonce == "" {
		return b
	}

	// 为脚本添加nonce
	if _, exists := b.directives[DirectiveScriptSrc]; exists {
		b.AddNonce(DirectiveScriptSrc)
	} else {
		b.AddNonce(DirectiveScriptSrc)
	}

	// 为样式添加nonce
	if _, exists := b.directives[DirectiveStyleSrc]; exists {
		b.AddNonce(DirectiveStyleSrc)
	} else {
		b.AddNonce(DirectiveStyleSrc)
	}

	return b
}

// RequireSRI 为特定指令要求使用SRI
func (b *Builder) RequireSRI(directive string, require bool) *Builder {
	b.requireSRI[directive] = require
	return b
}

// AllowUnsafeInline 添加unsafe-inline到指令
func (b *Builder) AllowUnsafeInline(directive string) *Builder {
	return b.Add(directive, "'unsafe-inline'")
}

// AllowUnsafeEval 添加unsafe-eval到指令
func (b *Builder) AllowUnsafeEval(directive string) *Builder {
	return b.Add(directive, "'unsafe-eval'")
}

// AddStrictDynamic 添加strict-dynamic到脚本指令
func (b *Builder) AddStrictDynamic() *Builder {
	return b.Add(DirectiveScriptSrc, "'strict-dynamic'")
}

// EnableUpgradeInsecureRequests 启用升级不安全请求
func (b *Builder) EnableUpgradeInsecureRequests() *Builder {
	b.directives[DirectiveUpgradeInsecureRequests] = nil
	return b
}

// BlockAllMixedContent 阻止所有混合内容
func (b *Builder) BlockAllMixedContent() *Builder {
	b.directives[DirectiveBlockAllMixedContent] = nil
	return b
}

// AddReporting 添加报告配置
func (b *Builder) AddReporting(endpoint string) *Builder {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		// 使用report-to (现代浏览器支持)
		b.directives[DirectiveReportTo] = []string{endpoint}
		// 同时添加report-uri (向后兼容)
		b.directives[DirectiveReportURI] = []string{endpoint}
	} else {
		// 假设是report-to组ID
		b.directives[DirectiveReportTo] = []string{endpoint}
	}
	return b
}

// String 生成内容安全策略字符串
func (b *Builder) String() string {
	// 处理report-uri和report-to
	if b.reportEndpoint != "" {
		b.AddReporting(b.reportEndpoint)
	}

	// 添加nonce到脚本和样式(如果设置了)
	if b.nonce != "" {
		b.AddNonceToScriptAndStyle()
	}

	// 提取并排序所有指令
	directives := make([]string, 0, len(b.directives))
	for directive := range b.directives {
		directives = append(directives, directive)
	}
	sort.Strings(directives)

	// 构建CSP策略
	var policies []string

	for _, directive := range directives {
		values := b.directives[directive]

		// 特殊处理不带值的指令
		if directive == DirectiveUpgradeInsecureRequests ||
			directive == DirectiveBlockAllMixedContent {
			policies = append(policies, directive)
			continue
		}

		// 检查是否需要SRI
		sri := ""
		if require, ok := b.requireSRI[directive]; ok && require {
			sri = " 'require-sri-for'"
		}

		// 构建指令字符串
		if len(values) > 0 {
			policy := directive + sri + " " + strings.Join(values, " ")
			policies = append(policies, policy)
		} else {
			policies = append(policies, directive+sri)
		}
	}

	return strings.Join(policies, "; ")
}

// ToHeader 返回CSP头名称和值
func (b *Builder) ToHeader() (string, string) {
	name := "Content-Security-Policy"
	if b.reportOnly {
		name = "Content-Security-Policy-Report-Only"
	}
	return name, b.String()
}

// CSPStrict 返回严格的CSP策略
func CSPStrict(nonce string) *Builder {
	builder := NewBuilder().
		Add(DirectiveDefaultSrc, "'self'").
		Add(DirectiveScriptSrc, "'self'").
		Add(DirectiveObjectSrc, "'none'").
		Add(DirectiveStyleSrc, "'self'").
		Add(DirectiveImgSrc, "'self'").
		Add(DirectiveMediaSrc, "'self'").
		Add(DirectiveFrameSrc, "'none'").
		Add(DirectiveFontSrc, "'self'").
		Add(DirectiveConnectSrc, "'self'").
		Add(DirectiveBaseURI, "'self'").
		Add(DirectiveFormAction, "'self'").
		Add(DirectiveFrameAncestors, "'none'").
		RequireSRI(DirectiveScriptSrc, true).
		RequireSRI(DirectiveStyleSrc, true).
		EnableUpgradeInsecureRequests()

	if nonce != "" {
		builder.SetNonce(nonce)
	}

	return builder
}

// CSPModern 返回适合现代Web应用的CSP策略
func CSPModern(nonce string) *Builder {
	builder := NewBuilder().
		Add(DirectiveDefaultSrc, "'self'").
		Add(DirectiveScriptSrc, "'self'").
		Add(DirectiveScriptSrcElem, "'self'").
		Add(DirectiveScriptSrcAttr, "'none'").
		Add(DirectiveStyleSrc, "'self'").
		Add(DirectiveStyleSrcElem, "'self'").
		Add(DirectiveStyleSrcAttr, "'none'").
		Add(DirectiveImgSrc, "'self' data:").
		Add(DirectiveFontSrc, "'self'").
		Add(DirectiveConnectSrc, "'self'").
		Add(DirectiveMediaSrc, "'self'").
		Add(DirectiveObjectSrc, "'none'").
		Add(DirectiveChildSrc, "'none'").
		Add(DirectiveFrameAncestors, "'none'").
		Add(DirectiveFormAction, "'self'").
		Add(DirectiveBaseURI, "'self'").
		Add(DirectiveManifestSrc, "'self'").
		Add(DirectiveWorkerSrc, "'self'").
		AddStrictDynamic().
		RequireSRI(DirectiveScriptSrc, true).
		RequireSRI(DirectiveStyleSrc, true).
		EnableUpgradeInsecureRequests()

	if nonce != "" {
		builder.SetNonce(nonce)
	}

	return builder
}

// CSPBasic 返回基本的CSP策略
func CSPBasic(nonce string) *Builder {
	builder := NewBuilder().
		Add(DirectiveDefaultSrc, "'self'").
		Add(DirectiveImgSrc, "'self' data:").
		Add(DirectiveScriptSrc, "'self'").
		Add(DirectiveStyleSrc, "'self'")

	if nonce == "" {
		builder.AllowUnsafeInline(DirectiveStyleSrc)
	} else {
		builder.SetNonce(nonce)
	}

	return builder
}

// 辅助函数

// generateHash 生成内容的哈希值
func generateHash(content, algorithm string) string {
	var hashBytes []byte

	switch algorithm {
	case HashAlgoSha256:
		hash := sha256.Sum256([]byte(content))
		hashBytes = hash[:]
	case HashAlgoSha384:
		hash := sha512.New384()
		hash.Write([]byte(content))
		hashBytes = hash.Sum(nil)
	case HashAlgoSha512:
		hash := sha512.Sum512([]byte(content))
		hashBytes = hash[:]
	default:
		// 默认使用SHA-256
		hash := sha256.Sum256([]byte(content))
		hashBytes = hash[:]
	}

	return base64.StdEncoding.EncodeToString(hashBytes)
}
