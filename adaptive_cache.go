package mist

import (
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveCache 是一个自适应的路由缓存结构
// 该缓存会根据路由的访问频率和响应时间动态调整缓存空间分配
type AdaptiveCache struct {
	// 缓存数据
	data sync.Map

	// 缓存统计
	hits          uint64
	misses        uint64
	evictions     uint64
	size          int64
	maxSize       int64
	enabled       bool
	lock          sync.RWMutex
	cleanupTicker *time.Ticker

	// 访问统计
	accessStats map[string]*accessStat // 访问统计数据
	statsMutex  sync.RWMutex           // 统计信息的互斥锁

	// 配置
	config AdaptiveCacheConfig
}

// AdaptiveCacheConfig 定义缓存的配置选项
type AdaptiveCacheConfig struct {
	MaxSize            int64         // 最大缓存条目数量
	CleanupInterval    time.Duration // 清理间隔时间
	MinAccessCount     int64         // 最小访问次数，低于此值会被优先考虑淘汰
	AccessTimeWeight   float64       // 访问时间权重(0-1)
	FrequencyWeight    float64       // 频率权重(0-1)
	ResponseTimeWeight float64       // 响应时间权重(0-1)
	AdaptiveMode       bool          // 是否启用自适应模式
}

// accessStat 记录路由的访问统计信息
type accessStat struct {
	key            string        // 缓存键
	accessCount    int64         // 访问次数
	lastAccessTime time.Time     // 最后访问时间
	totalTime      time.Duration // 总响应时间
	creationTime   time.Time     // 创建时间
	averageTime    time.Duration // 平均响应时间
	weight         float64       // 计算出的权重值，用于缓存替换策略
}

// NewAdaptiveCache 创建一个新的自适应缓存
func NewAdaptiveCache(config AdaptiveCacheConfig) *AdaptiveCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	// 设置权重默认值
	if config.AccessTimeWeight <= 0 {
		config.AccessTimeWeight = 0.3
	}
	if config.FrequencyWeight <= 0 {
		config.FrequencyWeight = 0.5
	}
	if config.ResponseTimeWeight <= 0 {
		config.ResponseTimeWeight = 0.2
	}

	if config.MinAccessCount <= 0 {
		config.MinAccessCount = 5
	}

	cache := &AdaptiveCache{
		data:        sync.Map{},
		accessStats: make(map[string]*accessStat),
		enabled:     true,
		maxSize:     config.MaxSize,
		config:      config,
	}

	// 启动定期清理任务
	cache.cleanupTicker = time.NewTicker(config.CleanupInterval)
	go cache.cleanup()

	return cache
}

// Get 从缓存获取值
func (c *AdaptiveCache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	value, found := c.data.Load(key)
	if found {
		atomic.AddUint64(&c.hits, 1)
		c.recordAccess(key, 0) // 记录访问，无响应时间
		return value, true
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// Set 将值设置到缓存
func (c *AdaptiveCache) Set(key string, value interface{}, responseTime time.Duration) {
	if !c.enabled {
		return
	}

	// 检查是否需要先清理一些空间
	if c.shouldCleanup() {
		c.evict()
	}

	c.data.Store(key, value)
	atomic.AddInt64(&c.size, 1)
	c.recordAccess(key, responseTime)
}

// Delete 从缓存删除值
func (c *AdaptiveCache) Delete(key string) {
	c.data.Delete(key)
	atomic.AddInt64(&c.size, -1)

	// 删除访问统计
	c.statsMutex.Lock()
	delete(c.accessStats, key)
	c.statsMutex.Unlock()
}

// Clear 清空缓存
func (c *AdaptiveCache) Clear() {
	c.data = sync.Map{}
	atomic.StoreInt64(&c.size, 0)

	c.statsMutex.Lock()
	c.accessStats = make(map[string]*accessStat)
	c.statsMutex.Unlock()

	atomic.StoreUint64(&c.hits, 0)
	atomic.StoreUint64(&c.misses, 0)
	atomic.StoreUint64(&c.evictions, 0)
}

// Enable 启用缓存
func (c *AdaptiveCache) Enable() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.enabled = true
}

// Disable 禁用缓存
func (c *AdaptiveCache) Disable() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.enabled = false
	c.Clear()
}

// Stats 返回缓存统计信息
func (c *AdaptiveCache) Stats() (hits, misses, evictions uint64, size, capacity int64) {
	hits = atomic.LoadUint64(&c.hits)
	misses = atomic.LoadUint64(&c.misses)
	evictions = atomic.LoadUint64(&c.evictions)
	size = atomic.LoadInt64(&c.size)
	capacity = c.maxSize
	return
}

// recordAccess 记录路由访问情况
func (c *AdaptiveCache) recordAccess(key string, responseTime time.Duration) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	now := time.Now()
	stat, exists := c.accessStats[key]

	if !exists {
		stat = &accessStat{
			key:            key,
			accessCount:    0,
			lastAccessTime: now,
			totalTime:      0,
			creationTime:   now,
			averageTime:    0,
		}
		c.accessStats[key] = stat
	}

	// 更新统计信息
	stat.accessCount++
	stat.lastAccessTime = now
	stat.totalTime += responseTime

	// 计算平均响应时间
	if stat.accessCount > 0 {
		stat.averageTime = time.Duration(int64(stat.totalTime) / stat.accessCount)
	}

	// 计算权重
	c.updateWeight(stat)
}

// updateWeight 更新访问权重
func (c *AdaptiveCache) updateWeight(stat *accessStat) {
	if !c.config.AdaptiveMode {
		// 非自适应模式，简单LRU
		stat.weight = float64(time.Since(stat.lastAccessTime))
		return
	}

	// 计算各指标的归一化值
	var timeScore, freqScore, respScore float64

	// 时间分数 - 最近访问的路由得分低(保留)
	timeScore = float64(time.Since(stat.lastAccessTime)) / float64(time.Hour)
	if timeScore > 1.0 {
		timeScore = 1.0
	}

	// 频率分数 - 访问频率高的路由得分低(保留)
	if stat.accessCount >= c.config.MinAccessCount {
		// 对数缩放，访问越多，分数越低
		freqScore = 1.0 / (1.0 + float64(stat.accessCount)/10.0)
	} else {
		// 访问次数太少，较高淘汰分数
		freqScore = 0.8
	}

	// 响应时间分数 - 响应快的路由得分低(保留)
	avgTimeMS := float64(stat.averageTime) / float64(time.Millisecond)
	if avgTimeMS > 0 {
		// 对数缩放，响应时间越短，分数越低
		respScore = float64(avgTimeMS) / 1000.0
		if respScore > 1.0 {
			respScore = 1.0
		}
	} else {
		respScore = 0.5 // 无响应时间记录，中等分数
	}

	// 组合权重 - 低分意味着高价值(应该保留)
	stat.weight = (timeScore * c.config.AccessTimeWeight) +
		(freqScore * c.config.FrequencyWeight) +
		(respScore * c.config.ResponseTimeWeight)
}

// shouldCleanup 检查是否需要清理缓存
func (c *AdaptiveCache) shouldCleanup() bool {
	return atomic.LoadInt64(&c.size) >= c.maxSize
}

// evict 淘汰一些缓存条目
func (c *AdaptiveCache) evict() {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	// 如果没有访问统计，直接返回
	if len(c.accessStats) == 0 {
		return
	}

	// 找出权重最高的条目(最应该被淘汰的)
	var maxKey string
	var maxWeight float64 = -1

	for k, stat := range c.accessStats {
		if stat.weight > maxWeight {
			maxWeight = stat.weight
			maxKey = k
		}
	}

	// 从缓存中删除
	if maxKey != "" {
		c.data.Delete(maxKey)
		delete(c.accessStats, maxKey)
		atomic.AddInt64(&c.size, -1)
		atomic.AddUint64(&c.evictions, 1)
	}
}

// cleanup 定期清理过期和低价值缓存
func (c *AdaptiveCache) cleanup() {
	for range c.cleanupTicker.C {
		if !c.enabled {
			continue
		}

		// 如果缓存使用率超过90%，主动清理低价值条目
		if float64(atomic.LoadInt64(&c.size)) > float64(c.maxSize)*0.9 {
			c.evictBatch(c.maxSize / 10) // 每次清理约10%
		}
	}
}

// evictBatch 批量淘汰缓存条目
func (c *AdaptiveCache) evictBatch(count int64) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	// 如果条目太少，不进行批量淘汰
	if int64(len(c.accessStats)) <= count {
		return
	}

	// 收集并排序所有条目
	type weightedEntry struct {
		key    string
		weight float64
	}

	entries := make([]weightedEntry, 0, len(c.accessStats))
	for k, stat := range c.accessStats {
		entries = append(entries, weightedEntry{
			key:    k,
			weight: stat.weight,
		})
	}

	// 根据权重排序(降序，权重高的更应该被淘汰)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].weight < entries[j].weight {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 淘汰权重最高的一批条目
	evicted := int64(0)
	for _, entry := range entries {
		if evicted >= count {
			break
		}

		// 从缓存和统计中删除
		c.data.Delete(entry.key)
		delete(c.accessStats, entry.key)
		atomic.AddInt64(&c.size, -1)
		atomic.AddUint64(&c.evictions, 1)
		evicted++
	}
}

// Close 关闭缓存和清理任务
func (c *AdaptiveCache) Close() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
	c.Clear()
}
