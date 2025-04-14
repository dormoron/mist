// Package config 提供统一的配置管理系统
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Provider 定义配置提供者接口
type Provider interface {
	// Get 获取配置项的值
	Get(key string) (interface{}, bool)

	// GetString 获取字符串配置
	GetString(key string) string

	// GetInt 获取整数配置
	GetInt(key string) int

	// GetFloat 获取浮点数配置
	GetFloat(key string) float64

	// GetBool 获取布尔配置
	GetBool(key string) bool

	// GetDuration 获取时间段配置
	GetDuration(key string) time.Duration

	// GetStringSlice 获取字符串切片配置
	GetStringSlice(key string) []string

	// GetStringMap 获取字符串映射配置
	GetStringMap(key string) map[string]interface{}

	// Set 设置配置项的值
	Set(key string, value interface{})

	// Has 检查配置项是否存在
	Has(key string) bool

	// AllSettings 获取所有配置
	AllSettings() map[string]interface{}

	// AllKeys 获取所有配置键
	AllKeys() []string

	// AddChangeListener 添加配置变更监听器
	AddChangeListener(listener func(key string))

	// RemoveChangeListener 移除配置变更监听器
	RemoveChangeListener(listener func(key string))

	// Unmarshal 将配置反序列化到结构体
	Unmarshal(key string, v interface{}) error
}

// Configuration 是配置管理器的实现
type Configuration struct {
	// 配置数据
	data map[string]interface{}

	// 环境变量前缀
	envPrefix string

	// 配置文件路径
	configFile string

	// 配置文件格式
	fileFormat string

	// 是否已加载配置
	loaded bool

	// 配置文件监视器
	watcher *fsnotify.Watcher

	// 配置变更监听器列表
	listeners []func(string)

	// 互斥锁保护并发访问
	mu sync.RWMutex
}

// 配置选项函数类型
type Option func(*Configuration)

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) Option {
	return func(c *Configuration) {
		c.envPrefix = prefix
	}
}

// WithConfigFile 设置配置文件路径
func WithConfigFile(file string) Option {
	return func(c *Configuration) {
		c.configFile = file

		// 自动检测文件格式
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".yaml" || ext == ".yml" {
			c.fileFormat = "yaml"
		} else if ext == ".json" {
			c.fileFormat = "json"
		} else if ext == ".toml" {
			c.fileFormat = "toml"
		} else {
			c.fileFormat = "unknown"
		}
	}
}

// WithFormat 明确设置配置文件格式
func WithFormat(format string) Option {
	return func(c *Configuration) {
		c.fileFormat = format
	}
}

// New 创建一个新的配置管理器
func New(options ...Option) (*Configuration, error) {
	config := &Configuration{
		data:      make(map[string]interface{}),
		listeners: make([]func(string), 0),
	}

	// 应用选项
	for _, option := range options {
		option(config)
	}

	// 创建文件监视器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("无法创建文件监视器: %w", err)
	}
	config.watcher = watcher

	// 加载配置
	if err := config.Load(); err != nil {
		return nil, err
	}

	// 开始监视配置文件变更
	if config.configFile != "" {
		go config.watchConfigFile()
	}

	return config, nil
}

// Load 加载配置
func (c *Configuration) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 首先加载配置文件
	if c.configFile != "" {
		if err := c.loadConfigFile(); err != nil {
			return err
		}
	}

	// 然后加载环境变量，环境变量优先级高于配置文件
	c.loadEnvironmentVariables()

	c.loaded = true
	return nil
}

// loadConfigFile 从文件加载配置
func (c *Configuration) loadConfigFile() error {
	file, err := os.Open(c.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，忽略错误
			return nil
		}
		return fmt.Errorf("无法打开配置文件: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("无法读取配置文件: %w", err)
	}

	// 根据文件格式解析配置文件
	var config map[string]interface{}

	switch c.fileFormat {
	case "yaml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("无法解析YAML配置文件: %w", err)
		}

	case "json":
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("无法解析JSON配置文件: %w", err)
		}

	case "toml":
		if err := toml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("无法解析TOML配置文件: %w", err)
		}

	default:
		return fmt.Errorf("不支持的配置文件格式: %s", c.fileFormat)
	}

	// 合并配置数据
	c.mergeConfig(config)

	return nil
}

// mergeConfig 合并配置
func (c *Configuration) mergeConfig(config map[string]interface{}) {
	// 简单的浅层合并
	for k, v := range config {
		c.data[k] = v
	}
}

// loadEnvironmentVariables 加载环境变量
func (c *Configuration) loadEnvironmentVariables() {
	// 获取所有环境变量
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, c.envPrefix) {
			continue
		}

		// 分割键值对
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// 移除前缀并转换为小写
		key := strings.ToLower(strings.TrimPrefix(parts[0], c.envPrefix))
		// 替换下划线为点，以支持层级配置
		key = strings.ReplaceAll(key, "_", ".")

		// 设置配置值
		c.data[key] = parts[1]
	}
}

// watchConfigFile 监视配置文件变更
func (c *Configuration) watchConfigFile() {
	// 添加文件到监视列表
	if err := c.watcher.Add(filepath.Dir(c.configFile)); err != nil {
		// 监视失败，记录错误但不中断程序
		fmt.Printf("无法监视配置文件: %v\n", err)
		return
	}

	// 监听事件
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}

			// 只关心配置文件的写入和创建事件
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 && event.Name == c.configFile {
				// 重新加载配置
				if err := c.Load(); err != nil {
					fmt.Printf("重新加载配置文件失败: %v\n", err)
					continue
				}

				// 通知所有监听器
				c.notifyListeners("")
			}

		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("配置文件监视错误: %v\n", err)
		}
	}
}

// notifyListeners 通知配置变更监听器
func (c *Configuration) notifyListeners(key string) {
	c.mu.RLock()
	listeners := make([]func(string), len(c.listeners))
	copy(listeners, c.listeners)
	c.mu.RUnlock()

	for _, listener := range listeners {
		listener(key)
	}
}

// Get 获取配置值
func (c *Configuration) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 直接查找键
	if value, ok := c.data[key]; ok {
		return value, true
	}

	// 尝试查找嵌套键
	parts := strings.Split(key, ".")
	current := c.data

	for i, part := range parts {
		if v, ok := current[part]; ok {
			if i == len(parts)-1 {
				return v, true
			}

			if nested, ok := v.(map[string]interface{}); ok {
				current = nested
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return nil, false
}

// GetString 获取字符串配置
func (c *Configuration) GetString(key string) string {
	value, ok := c.Get(key)
	if !ok {
		return ""
	}

	// 尝试转换为字符串
	str, ok := value.(string)
	if !ok {
		// 尝试其他类型转换
		return fmt.Sprintf("%v", value)
	}

	return str
}

// GetInt 获取整数配置
func (c *Configuration) GetInt(key string) int {
	value, ok := c.Get(key)
	if !ok {
		return 0
	}

	// 根据不同类型转换
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}

	return 0
}

// GetFloat 获取浮点数配置
func (c *Configuration) GetFloat(key string) float64 {
	value, ok := c.Get(key)
	if !ok {
		return 0
	}

	// 根据不同类型转换
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}

	return 0
}

// GetBool 获取布尔配置
func (c *Configuration) GetBool(key string) bool {
	value, ok := c.Get(key)
	if !ok {
		return false
	}

	// 根据不同类型转换
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	}

	return false
}

// GetDuration 获取时间段配置
func (c *Configuration) GetDuration(key string) time.Duration {
	value, ok := c.Get(key)
	if !ok {
		return 0
	}

	// 根据不同类型转换
	switch v := value.(type) {
	case time.Duration:
		return v
	case int:
		return time.Duration(v) * time.Second
	case int64:
		return time.Duration(v) * time.Second
	case float64:
		return time.Duration(v) * time.Second
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}

	return 0
}

// GetStringSlice 获取字符串切片配置
func (c *Configuration) GetStringSlice(key string) []string {
	value, ok := c.Get(key)
	if !ok {
		return nil
	}

	// 根据不同类型转换
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		// 将interface{}切片转换为字符串切片
		result := make([]string, len(v))
		for i, val := range v {
			result[i] = fmt.Sprintf("%v", val)
		}
		return result
	case string:
		// 如果是以逗号分隔的字符串，分割它
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			// 去除空白
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			return parts
		}
		return []string{v}
	}

	return nil
}

// GetStringMap 获取字符串映射配置
func (c *Configuration) GetStringMap(key string) map[string]interface{} {
	value, ok := c.Get(key)
	if !ok {
		return nil
	}

	// 根据不同类型转换
	if m, ok := value.(map[string]interface{}); ok {
		return m
	}

	return nil
}

// Set 设置配置值
func (c *Configuration) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 直接设置键值对
	c.data[key] = value

	// 通知监听器
	go c.notifyListeners(key)
}

// Has 检查配置项是否存在
func (c *Configuration) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// AllSettings 获取所有配置
func (c *Configuration) AllSettings() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建配置的深拷贝
	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}

	return result
}

// AllKeys 获取所有配置键
func (c *Configuration) AllKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}

	return keys
}

// AddChangeListener 添加配置变更监听器
func (c *Configuration) AddChangeListener(listener func(key string)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.listeners = append(c.listeners, listener)
}

// RemoveChangeListener 移除配置变更监听器
func (c *Configuration) RemoveChangeListener(listener func(key string)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 查找并移除监听器
	for i, l := range c.listeners {
		if fmt.Sprintf("%p", l) == fmt.Sprintf("%p", listener) {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			break
		}
	}
}

// Unmarshal 将配置反序列化到结构体
func (c *Configuration) Unmarshal(key string, v interface{}) error {
	value, ok := c.Get(key)
	if !ok {
		return fmt.Errorf("配置键不存在: %s", key)
	}

	// 使用mapstructure进行解码
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      v,
		TagName:     "config",
		ErrorUnused: false,
	})

	if err != nil {
		return fmt.Errorf("创建解码器失败: %w", err)
	}

	return decoder.Decode(value)
}

// Close 关闭配置管理器，释放资源
func (c *Configuration) Close() error {
	if c.watcher != nil {
		return c.watcher.Close()
	}
	return nil
}
