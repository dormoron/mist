package blocklist

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist"
)

// IPRecord 表示IP访问记录
type IPRecord struct {
	// IP地址
	IP string `json:"ip"`
	// LastActivity 最后活动时间
	LastActivity time.Time `json:"last_activity"`
	// FailedAttempts 失败尝试次数
	FailedAttempts int `json:"failed_attempts"`
	// BlockedUntil 封禁解除时间
	BlockedUntil time.Time `json:"blocked_until"`
	// BlockCount 封禁次数，用于递增封禁时长
	BlockCount int `json:"block_count"`
	// CountryCode 国家/地区代码
	CountryCode string `json:"country_code,omitempty"`
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
	// Storage 存储实现，默认为内存存储
	Storage Storage
	// UseProgressiveBlocking 是否使用递增封禁时长
	UseProgressiveBlocking bool
	// ProgressiveBlockingFactor 递增封禁时长因子
	ProgressiveBlockingFactor float64
	// MaxBlockDuration 最大封禁时长
	MaxBlockDuration time.Duration
	// GeoRestriction 地理位置限制，可选
	GeoRestriction *GeoRestriction
	// AllowPrivateIPs 是否允许私有IP地址
	AllowPrivateIPs bool
}

// Manager IP黑名单管理器
type Manager struct {
	config  BlocklistConfig
	storage Storage
	mu      sync.RWMutex
	done    chan struct{}
}

// NewManager 创建一个新的黑名单管理器
func NewManager(options ...func(*BlocklistConfig)) *Manager {
	// 默认配置
	config := BlocklistConfig{
		MaxFailedAttempts:         5,
		BlockDuration:             15 * time.Minute,
		ClearInterval:             5 * time.Minute,
		RecordExpiry:              24 * time.Hour,
		OnBlocked:                 defaultOnBlocked,
		WhitelistIPs:              []string{},
		UseProgressiveBlocking:    true,
		ProgressiveBlockingFactor: 2.0,
		MaxBlockDuration:          7 * 24 * time.Hour, // 最长封禁1周
		AllowPrivateIPs:           true,
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

	// 如果没有提供存储，使用内存存储
	if config.Storage == nil {
		config.Storage = NewMemoryStorage()
	}

	manager := &Manager{
		config:  config,
		storage: config.Storage,
		done:    make(chan struct{}),
	}

	// 启动清理协程
	go manager.cleanupLoop()

	return manager
}

// 默认的处理函数
func defaultOnBlocked(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("您的IP已被暂时封禁，请稍后再试"))
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
	// 获取所有记录
	records, err := m.storage.ListIPRecords()
	if err != nil {
		// 记录错误但继续运行
		return
	}

	now := time.Now()
	for _, record := range records {
		// 如果封禁已过期且最后活动时间超过记录过期时间，则删除记录
		if record.BlockedUntil.Before(now) && record.LastActivity.Add(m.config.RecordExpiry).Before(now) {
			m.storage.DeleteIPRecord(record.IP)
		}
	}
}

// Stop 停止黑名单管理器
func (m *Manager) Stop() {
	close(m.done)
	m.storage.Close()
}

// RecordSuccess 记录成功的尝试，重置失败计数
func (m *Manager) RecordSuccess(ip string) {
	// 如果IP在白名单中，则不记录
	if m.config.whitelistMap[ip] {
		return
	}

	// 如果是私有IP且配置允许，则不记录
	if !m.config.AllowPrivateIPs && isPrivateIP(ip) {
		return
	}

	record, err := m.storage.GetIPRecord(ip)
	if err != nil {
		// 记录错误但继续运行
		return
	}

	if record == nil {
		// 如果记录不存在，创建新记录
		record = &IPRecord{
			IP:             ip,
			LastActivity:   time.Now(),
			FailedAttempts: 0,
		}
	} else {
		// 更新记录
		record.LastActivity = time.Now()
		record.FailedAttempts = 0
	}

	m.storage.SaveIPRecord(record)
}

// RecordFailure 记录失败的尝试
func (m *Manager) RecordFailure(ip string) bool {
	// 如果IP在白名单中，则不记录
	if m.config.whitelistMap[ip] {
		return false
	}

	// 如果是私有IP且配置允许，则不记录
	if !m.config.AllowPrivateIPs && isPrivateIP(ip) {
		return false
	}

	now := time.Now()
	record, err := m.storage.GetIPRecord(ip)
	if err != nil {
		// 记录错误但继续运行
		return false
	}

	if record == nil {
		// 如果记录不存在，创建新记录
		record = &IPRecord{
			IP:             ip,
			LastActivity:   now,
			FailedAttempts: 1,
		}
		m.storage.SaveIPRecord(record)
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
		// 如果使用递增封禁时长
		if m.config.UseProgressiveBlocking {
			// 增加封禁次数
			record.BlockCount++
			// 计算封禁时长
			blockDuration := time.Duration(float64(m.config.BlockDuration) *
				pow(m.config.ProgressiveBlockingFactor, float64(record.BlockCount-1)))

			// 确保不超过最大封禁时长
			if blockDuration > m.config.MaxBlockDuration {
				blockDuration = m.config.MaxBlockDuration
			}

			record.BlockedUntil = now.Add(blockDuration)
		} else {
			record.BlockedUntil = now.Add(m.config.BlockDuration)
		}

		// 重置失败计数
		record.FailedAttempts = 0
		m.storage.SaveIPRecord(record)
		return true
	}

	m.storage.SaveIPRecord(record)
	return false
}

// 计算a的b次方
func pow(a, b float64) float64 {
	result := 1.0
	for i := 0; i < int(b); i++ {
		result *= a
	}
	return result
}

// IsBlocked 检查IP是否被封禁
func (m *Manager) IsBlocked(ip string) bool {
	// 如果IP在白名单中，则不封禁
	if m.config.whitelistMap[ip] {
		return false
	}

	// 如果是私有IP且配置允许，则不封禁
	if !m.config.AllowPrivateIPs && isPrivateIP(ip) {
		return false
	}

	// 检查地理位置限制
	if m.config.GeoRestriction != nil {
		restricted, err := m.config.GeoRestriction.IsIPRestricted(ip)
		if err == nil && restricted {
			return true
		}
	}

	record, err := m.storage.GetIPRecord(ip)
	if err != nil || record == nil {
		return false
	}

	return record.BlockedUntil.After(time.Now())
}

// BlockIP 手动封禁IP
func (m *Manager) BlockIP(ip string, duration time.Duration) {
	// 如果IP在白名单中，则不封禁
	if m.config.whitelistMap[ip] {
		return
	}

	// 如果是私有IP且配置允许，则不封禁
	if !m.config.AllowPrivateIPs && isPrivateIP(ip) {
		return
	}

	// 获取记录，如果不存在则创建
	record, err := m.storage.GetIPRecord(ip)
	if err != nil {
		// 记录错误但继续运行
		return
	}

	if record == nil {
		record = &IPRecord{
			IP:             ip,
			LastActivity:   time.Now(),
			FailedAttempts: m.config.MaxFailedAttempts,
			BlockCount:     1,
		}
	} else {
		// 增加封禁次数
		record.BlockCount++
	}

	// 设置封禁时间
	record.BlockedUntil = time.Now().Add(duration)
	m.storage.SaveIPRecord(record)
}

// UnblockIP 手动解除IP封禁
func (m *Manager) UnblockIP(ip string) {
	record, err := m.storage.GetIPRecord(ip)
	if err != nil || record == nil {
		return
	}

	// 重置记录
	record.FailedAttempts = 0
	record.BlockedUntil = time.Time{}
	m.storage.SaveIPRecord(record)
}

// GetIPInfo 获取IP详细信息
func (m *Manager) GetIPInfo(ip string) (*IPRecord, error) {
	record, err := m.storage.GetIPRecord(ip)
	if err != nil {
		return nil, err
	}

	// 如果记录存在且配置了地理位置限制，获取国家信息
	if record != nil && m.config.GeoRestriction != nil && record.CountryCode == "" {
		countryCode, err := m.config.GeoRestriction.GetCountryCode(ip)
		if err == nil {
			record.CountryCode = countryCode
			m.storage.SaveIPRecord(record)
		}
	}

	return record, nil
}

// ListBlockedIPs 列出所有被封禁的IP
func (m *Manager) ListBlockedIPs() ([]*IPRecord, error) {
	return m.storage.ListBlockedIPs()
}

// Middleware 创建IP黑名单中间件
func (m *Manager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			// 检查IP是否被封禁
			if m.IsBlocked(ip) {
				m.config.OnBlocked(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MistBlocklistConfig Mist框架的黑名单配置
type MistBlocklistConfig struct {
	// 原始黑名单配置
	Config *BlocklistConfig
	// 当IP被封禁时的处理函数（适用于Mist框架）
	OnBlocked func(*mist.Context)
}

// MistMiddleware 创建适用于Mist框架的中间件
func (m *Manager) MistMiddleware(options ...func(*MistBlocklistConfig)) mist.Middleware {
	cfg := &MistBlocklistConfig{
		Config: &m.config,
		OnBlocked: func(ctx *mist.Context) {
			ctx.RespondWithJSON(http.StatusForbidden, "您的IP已被暂时封禁，请稍后再试")
		},
	}

	// 应用自定义选项
	for _, option := range options {
		option(cfg)
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			ip := getClientIP(ctx.Request)

			// 检查IP是否被封禁
			if m.IsBlocked(ip) {
				cfg.OnBlocked(ctx)
				return
			}

			next(ctx)
		}
	}
}

// WithMistOnBlocked 设置Mist框架的IP被封禁时的处理函数
func WithMistOnBlocked(handler func(*mist.Context)) func(*MistBlocklistConfig) {
	return func(c *MistBlocklistConfig) {
		c.OnBlocked = handler
	}
}

// getClientIP 获取客户端真实IP
func getClientIP(r *http.Request) string {
	// 先尝试从X-Forwarded-For获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(ip, ",")
		if len(ips) > 0 && ips[0] != "" {
			return strings.TrimSpace(ips[0])
		}
	}

	// 再尝试从X-Real-IP获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 最后从RemoteAddr获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // 如果解析失败，直接返回原始值
	}
	return ip
}

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

// WithOnBlocked 设置封禁时的处理函数
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

// WithStorage 设置存储实现
func WithStorage(storage Storage) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.Storage = storage
	}
}

// WithProgressiveBlocking 设置是否使用递增封禁时长
func WithProgressiveBlocking(enable bool, factor float64, maxDuration time.Duration) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.UseProgressiveBlocking = enable
		if factor > 0 {
			c.ProgressiveBlockingFactor = factor
		}
		if maxDuration > 0 {
			c.MaxBlockDuration = maxDuration
		}
	}
}

// WithGeoRestriction 设置地理位置限制
func WithGeoRestriction(geo *GeoRestriction) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.GeoRestriction = geo
	}
}

// WithAllowPrivateIPs 设置是否允许私有IP
func WithAllowPrivateIPs(allow bool) func(*BlocklistConfig) {
	return func(c *BlocklistConfig) {
		c.AllowPrivateIPs = allow
	}
}

// isPrivateIP 检查IP是否为私有地址
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 检查IPv4私有地址范围
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 127.0.0.0/8
		if ip4[0] == 127 {
			return true
		}
	} else {
		// IPv6 localhost
		if ip.Equal(net.ParseIP("::1")) {
			return true
		}
		// IPv6 unique local address (fc00::/7)
		return len(ip) == 16 && (ip[0] == 0xfc || ip[0] == 0xfd)
	}

	return false
}
