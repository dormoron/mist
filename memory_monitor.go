package mist

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MemStats 保存内存统计信息
type MemStats struct {
	// 常规内存统计信息
	Alloc        uint64    // 当前已分配的内存字节数
	TotalAlloc   uint64    // 累计分配的总内存字节数
	Sys          uint64    // 从系统获取的总内存字节数
	NumGC        uint32    // GC运行次数
	PauseTotalNs uint64    // GC暂停总时间
	LastGC       time.Time // 最后一次GC的时间

	// Goroutine信息
	NumGoroutine int // 当前goroutine数量

	// 内存使用趋势
	PrevAlloc   uint64  // 上次检查时的内存使用量
	AllocGrowth int64   // 内存使用增长量（可为负数）
	GrowthRate  float64 // 内存增长率

	// 采样时间
	SampleTime time.Time // 采样时间点
}

// MemoryMonitor 是一个内存使用监控器
type MemoryMonitor struct {
	// 内部状态
	enabled  bool
	interval time.Duration
	stopChan chan struct{}
	mutex    sync.RWMutex

	// 采样数据
	currentStats MemStats
	samples      []MemStats
	maxSamples   int

	// 告警阈值
	alertThreshold float64         // 内存增长率告警阈值
	callbacks      []AlertCallback // 告警回调函数

	// 上次GC触发信息
	lastGCCount uint32
	forcedGCs   uint64
}

// AlertCallback 是内存告警回调函数类型
type AlertCallback func(stats MemStats, message string)

// NewMemoryMonitor 创建一个新的内存监控器
func NewMemoryMonitor(interval time.Duration, maxSamples int) *MemoryMonitor {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	if maxSamples <= 0 {
		maxSamples = 60 // 默认保存10分钟数据（按10秒间隔）
	}

	return &MemoryMonitor{
		interval:       interval,
		maxSamples:     maxSamples,
		samples:        make([]MemStats, 0, maxSamples),
		stopChan:       make(chan struct{}),
		alertThreshold: 0.2, // 默认20%增长率触发告警
		callbacks:      make([]AlertCallback, 0),
	}
}

// Start 启动内存监控
func (m *MemoryMonitor) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.enabled {
		return
	}

	// 重新初始化stopChan，避免使用已关闭的通道
	m.stopChan = make(chan struct{})
	m.enabled = true

	// 采集初始样本
	m.collectSample()

	// 启动定期监控任务
	go m.monitor()
}

// Stop 停止内存监控
func (m *MemoryMonitor) Stop() {
	m.mutex.Lock()

	if !m.enabled {
		m.mutex.Unlock()
		return
	}

	m.enabled = false
	stopCh := m.stopChan // 保存一个引用，避免解锁后通道被修改
	m.mutex.Unlock()

	// 在解锁后关闭通道，避免死锁
	close(stopCh)
}

// AddAlertCallback 添加告警回调函数
func (m *MemoryMonitor) AddAlertCallback(callback AlertCallback) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callbacks = append(m.callbacks, callback)
}

// SetAlertThreshold 设置告警阈值
func (m *MemoryMonitor) SetAlertThreshold(threshold float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if threshold > 0 {
		m.alertThreshold = threshold
	}
}

// GetCurrentStats 获取当前内存统计信息
func (m *MemoryMonitor) GetCurrentStats() MemStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.currentStats
}

// GetSamples 获取历史采样数据
func (m *MemoryMonitor) GetSamples() []MemStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]MemStats, len(m.samples))
	copy(result, m.samples)
	return result
}

// ForceGC 强制触发垃圾回收
func (m *MemoryMonitor) ForceGC() {
	runtime.GC()
	atomic.AddUint64(&m.forcedGCs, 1)
}

// monitor 执行周期性内存监控
func (m *MemoryMonitor) monitor() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectSample()
			m.checkMemoryGrowth()
		case <-m.stopChan:
			return
		}
	}
}

// collectSample 收集内存样本
func (m *MemoryMonitor) collectSample() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	currentStats := MemStats{
		Alloc:        memStats.Alloc,
		TotalAlloc:   memStats.TotalAlloc,
		Sys:          memStats.Sys,
		NumGC:        memStats.NumGC,
		PauseTotalNs: memStats.PauseTotalNs,
		LastGC:       time.Unix(0, int64(memStats.LastGC)),
		NumGoroutine: runtime.NumGoroutine(),
		SampleTime:   time.Now(),
	}

	// 计算内存增长
	if len(m.samples) > 0 {
		prevStats := m.samples[len(m.samples)-1]
		currentStats.PrevAlloc = prevStats.Alloc
		currentStats.AllocGrowth = int64(currentStats.Alloc) - int64(prevStats.Alloc)

		// 计算增长率
		if prevStats.Alloc > 0 {
			currentStats.GrowthRate = float64(currentStats.AllocGrowth) / float64(prevStats.Alloc)
		}
	}

	// 添加到样本列表，保持最大样本数限制
	m.samples = append(m.samples, currentStats)
	if len(m.samples) > m.maxSamples {
		// 移除最老的样本
		m.samples = m.samples[1:]
	}

	// 更新当前状态
	m.currentStats = currentStats
}

// checkMemoryGrowth 检查内存增长情况
func (m *MemoryMonitor) checkMemoryGrowth() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.samples) < 2 {
		return
	}

	// 获取最新样本
	currentStats := m.samples[len(m.samples)-1]

	// 检查GC运行情况
	if m.lastGCCount != currentStats.NumGC {
		m.lastGCCount = currentStats.NumGC
	}

	// 检查是否需要触发告警
	if len(m.callbacks) > 0 && currentStats.GrowthRate > m.alertThreshold {
		message := "内存使用量增长率超过阈值"

		// 调用所有回调函数
		for _, callback := range m.callbacks {
			if callback != nil {
				go callback(currentStats, message)
			}
		}
	}
}

// GetMemoryUsageReport 获取内存使用报告
func (m *MemoryMonitor) GetMemoryUsageReport() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := m.currentStats

	// 创建报告
	report := map[string]interface{}{
		"current": map[string]interface{}{
			"alloc_mb":       float64(stats.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(stats.TotalAlloc) / 1024 / 1024,
			"sys_mb":         float64(stats.Sys) / 1024 / 1024,
			"num_gc":         stats.NumGC,
			"goroutines":     stats.NumGoroutine,
			"sample_time":    stats.SampleTime,
		},
		"growth": map[string]interface{}{
			"rate":       stats.GrowthRate,
			"bytes":      stats.AllocGrowth,
			"is_growing": stats.AllocGrowth > 0,
		},
		"forced_gc": atomic.LoadUint64(&m.forcedGCs),
	}

	// 添加简单的趋势数据
	if len(m.samples) > 0 {
		trendPoints := min(len(m.samples), 10)
		trend := make([]interface{}, trendPoints)

		for i := 0; i < trendPoints; i++ {
			idx := len(m.samples) - trendPoints + i
			sample := m.samples[idx]

			trend[i] = map[string]interface{}{
				"alloc_mb":    float64(sample.Alloc) / 1024 / 1024,
				"goroutines":  sample.NumGoroutine,
				"sample_time": sample.SampleTime,
				"growth_rate": sample.GrowthRate,
			}
		}

		report["trend"] = trend
	}

	return report
}

// min returns the smaller of x or y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
