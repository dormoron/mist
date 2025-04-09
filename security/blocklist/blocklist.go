package blocklist

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

// IPRecord 表示IP访问记录
type IPRecord struct {
	// IP地址
	IP string
	// LastActivity 最后活动时间
	LastActivity time.Time
	// FailedAttempts 失败尝试次数
	FailedAttempts int
	// BlockedUntil 封禁解除时间
	BlockedUntil time.Time
}

// BlocklistConfig 黑名单配置
type BlocklistConfig struct {
	// MaxFailedAttempts 最大失败尝试次数，超过则封禁
	MaxFailedAttempts int
	// BlockDuration 封禁时长
	BlockDuration time.Duration
	// ClearInterval 清理间隔，定期清理过期的记录
	ClearInterval time.Duration
	// OnBlocked 封禁时的处理函数
	OnBlocked func(w http.ResponseWriter, r *http.Request)
	// RecordExpiry 记录过期时间，过期后失败次数重置
	RecordExpiry time.Duration
	// WhitelistIPs 白名单IP，这些IP不会被封禁
	WhitelistIPs []string
	// whitelistMap 白名单IP映射表（内部使用）
	whitelistMap map[string]bool
}

// Manager IP黑名单管理器
type Manager struct {
	records map[string]*IPRecord
	config  BlocklistConfig
	mu      sync.RWMutex
	done    chan struct{}
}

// NewManager 创建一个新的黑名单管理器
func NewManager(options ...func(*BlocklistConfig)) *Manager {
	// 默认配置
	config := BlocklistConfig{
		MaxFailedAttempts: 5,
		BlockDuration:     15 * time.Minute,
		ClearInterval:     5 * time.Minute,
		RecordExpiry:      24 * time.Hour,
		OnBlocked: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("您的IP已被暂时封禁，请稍后再试"))
		},
		WhitelistIPs: []string{},
	}

	// 应用自定义选项
	for _, option := range options {
		option(&config)
	}

	// 初始化白名单映射
	config.whitelistMap = make(map[string]bool)
	for _, ip := range config.WhitelistIPs {
		config.whitelistMap[ip] = true
	}

	manager := &Manager{
		records: make(map[string]*IPRecord),
		config:  config,
		done:    make(chan struct{}),
	}

	// 启动清理协程
	go manager.cleanupLoop()

	return manager
}

// cleanupLoop 定期清理过期的记录
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.config.ClearInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.done:
			return
		}
	}
}

// cleanup 清理过期记录
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for ip, record := range m.records {
		// 如果封禁已过期且最后活动时间超过记录过期时间，则删除记录
		if record.BlockedUntil.Before(now) && record.LastActivity.Add(m.config.RecordExpiry).Before(now) {
			delete(m.records, ip)
		}
	}
}

// Stop 停止黑名单管理器
func (m *Manager) Stop() {
	close(m.done)
}

// RecordSuccess 记录成功的尝试，重置失败计数
func (m *Manager) RecordSuccess(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果IP在白名单中，则不记录
	if m.config.whitelistMap[ip] {
		return
	}

	record, exists := m.records[ip]
	if !exists {
		// 如果记录不存在，创建新记录
		m.records[ip] = &IPRecord{
			IP:             ip,
			LastActivity:   time.Now(),
			FailedAttempts: 0,
		}
		return
	}

	// 更新记录
	record.LastActivity = time.Now()
	record.FailedAttempts = 0
}

// RecordFailure 记录失败的尝试
func (m *Manager) RecordFailure(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果IP在白名单中，则不记录
	if m.config.whitelistMap[ip] {
		return false
	}

	now := time.Now()
	record, exists := m.records[ip]
	if !exists {
		// 如果记录不存在，创建新记录
		m.records[ip] = &IPRecord{
			IP:             ip,
			LastActivity:   now,
			FailedAttempts: 1,
		}
		return false
	}

	// 如果已被封禁且封禁时间未到，则返回true
	if record.BlockedUntil.After(now) {
		return true
	}

	// 记录最后活动时间并增加失败计数
	record.LastActivity = now
	record.FailedAttempts++

	// 如果失败次数超过阈值，则封禁IP
	if record.FailedAttempts >= m.config.MaxFailedAttempts {
		record.BlockedUntil = now.Add(m.config.BlockDuration)
		return true
	}

	return false
}

// IsBlocked 检查IP是否被封禁
func (m *Manager) IsBlocked(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果IP在白名单中，则不封禁
	if m.config.whitelistMap[ip] {
		return false
	}

	record, exists := m.records[ip]
	if !exists {
		return false
	}

	return record.BlockedUntil.After(time.Now())
}

// BlockIP 手动封禁IP
func (m *Manager) BlockIP(ip string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果IP在白名单中，则不封禁
	if m.config.whitelistMap[ip] {
		return
	}

	// 获取记录，如果不存在则创建
	record, exists := m.records[ip]
	if !exists {
		record = &IPRecord{
			IP:             ip,
			LastActivity:   time.Now(),
			FailedAttempts: m.config.MaxFailedAttempts,
		}
		m.records[ip] = record
	}

	// 设置封禁时间
	record.BlockedUntil = time.Now().Add(duration)
}

// UnblockIP 手动解除IP封禁
func (m *Manager) UnblockIP(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, exists := m.records[ip]
	if !exists {
		return
	}

	// 重置记录
	record.FailedAttempts = 0
	record.BlockedUntil = time.Time{}
}

// Middleware 创建IP黑名单中间件
func (m *Manager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			// 检查IP是否被封禁
			if m.IsBlocked(ip) {
				// 调用封禁处理函数
				m.config.OnBlocked(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MistBlocklistConfig Mist框架的IP黑名单配置
type MistBlocklistConfig struct {
	// 原始黑名单配置
	Config *BlocklistConfig
	// 当IP被封禁时的处理函数（适用于Mist框架）
	OnBlocked func(*mist.Context)
}

// 全局默认Mist配置
var defaultMistConfig = MistBlocklistConfig{
	OnBlocked: func(ctx *mist.Context) {
		ctx.AbortWithStatus(http.StatusForbidden)
	},
}

// MistMiddleware 创建适用于Mist框架的中间件，支持自定义封禁处理函数
// 已废弃: 请使用 security/blocklist/middleware 包中的 New 函数
func (m *Manager) MistMiddleware(options ...func(*MistBlocklistConfig)) mist.Middleware {
	// 兼容旧版，调用新的middleware实现
	config := defaultMistConfig
	config.Config = &m.config

	// 应用自定义选项
	for _, option := range options {
		option(&config)
	}

	// 使用新的实现
	if config.OnBlocked != nil {
		// 由于无法在函数中使用import，
		// 这里需要手动实现中间件而不是使用middleware包
		return func(next mist.HandleFunc) mist.HandleFunc {
			return func(ctx *mist.Context) {
				ip := ctx.ClientIP()

				// 如果IP被封禁，中断请求
				if m.IsBlocked(ip) {
					// 调用封禁处理函数
					config.OnBlocked(ctx)
					return
				}

				// 继续处理请求
				next(ctx)
			}
		}
	}

	// 默认处理
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			ip := ctx.ClientIP()

			// 如果IP被封禁，中断请求
			if m.IsBlocked(ip) {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}

			// 继续处理请求
			next(ctx)
		}
	}
}

// WithMistOnBlocked 设置Mist框架中IP封禁时的处理函数
func WithMistOnBlocked(handler func(*mist.Context)) func(*MistBlocklistConfig) {
	return func(c *MistBlocklistConfig) {
		c.OnBlocked = handler
	}
}

// getClientIP 从请求中获取客户端IP
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

// 选项函数

// WithMaxFailedAttempts 设置最大失败尝试次数
func WithMaxFailedAttempts(max int) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.MaxFailedAttempts = max
	}
}

// WithBlockDuration 设置封禁时长
func WithBlockDuration(duration time.Duration) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.BlockDuration = duration
	}
}

// WithClearInterval 设置清理间隔
func WithClearInterval(interval time.Duration) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.ClearInterval = interval
	}
}

// WithOnBlocked 设置封禁处理函数
func WithOnBlocked(handler func(w http.ResponseWriter, r *http.Request)) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.OnBlocked = handler
	}
}

// WithRecordExpiry 设置记录过期时间
func WithRecordExpiry(expiry time.Duration) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.RecordExpiry = expiry
	}
}

// WithWhitelistIPs 设置白名单IP
func WithWhitelistIPs(ips []string) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.WhitelistIPs = ips
	}
}

// IPRangeConfig IP范围配置
type IPRangeConfig struct {
	// AllowedNetworks 允许的IP网段
	AllowedNetworks []*net.IPNet
	// DeniedNetworks 拒绝的IP网段
	DeniedNetworks []*net.IPNet
	// DefaultAllow 默认允许策略（如果为true，则除了明确拒绝的IP外都允许）
	DefaultAllow bool
	// OnDenied 拒绝访问时的处理函数
	OnDenied func(*mist.Context)
}

// IPRangeMiddleware 基于IP范围的中间件
func IPRangeMiddleware(config IPRangeConfig) mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 获取客户端IP
			ipStr := ctx.ClientIP()
			ip := net.ParseIP(ipStr)
			if ip == nil {
				// 无法解析IP，按默认策略处理
				if !config.DefaultAllow {
					if config.OnDenied != nil {
						config.OnDenied(ctx)
					} else {
						ctx.AbortWithStatus(http.StatusForbidden)
					}
				} else {
					next(ctx)
				}
				return
			}

			// 检查是否在拒绝列表中
			for _, network := range config.DeniedNetworks {
				if network.Contains(ip) {
					if config.OnDenied != nil {
						config.OnDenied(ctx)
					} else {
						ctx.AbortWithStatus(http.StatusForbidden)
					}
					return
				}
			}

			// 检查是否在允许列表中
			allowed := config.DefaultAllow
			if len(config.AllowedNetworks) > 0 {
				allowed = false
				for _, network := range config.AllowedNetworks {
					if network.Contains(ip) {
						allowed = true
						break
					}
				}
			}

			if !allowed {
				if config.OnDenied != nil {
					config.OnDenied(ctx)
				} else {
					ctx.AbortWithStatus(http.StatusForbidden)
				}
				return
			}

			next(ctx)
		}
	}
}

// ParseCIDR 解析CIDR字符串为IP网段
func ParseCIDR(cidrs []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}
	return networks, nil
}
