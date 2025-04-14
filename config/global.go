package config

import (
	"log"
	"path/filepath"
	"sync"
)

var (
	// 全局配置实例
	globalConfig Provider

	// 单例锁
	once sync.Once
)

// Init 初始化全局配置
func Init(options ...Option) error {
	var err error

	once.Do(func() {
		var config *Configuration
		config, err = New(options...)
		if err != nil {
			return
		}

		globalConfig = config
	})

	return err
}

// Get 获取全局配置实例
func Get() Provider {
	if globalConfig == nil {
		// 如果全局配置未初始化，使用默认配置
		err := Init()
		if err != nil {
			log.Printf("初始化默认配置失败: %v", err)
		}
	}

	return globalConfig
}

// AutoInit 自动初始化配置
// 自动检测当前目录下的配置文件和环境变量
func AutoInit(appName string) error {
	// 尝试加载多种格式的配置文件
	configFiles := []string{
		filepath.Join(".", "config.yaml"),
		filepath.Join(".", "config.yml"),
		filepath.Join(".", "config.json"),
		filepath.Join(".", "config.toml"),
		filepath.Join(".", "configs", "config.yaml"),
		filepath.Join(".", "configs", "config.yml"),
		filepath.Join(".", "configs", "config.json"),
		filepath.Join(".", "configs", "config.toml"),
		filepath.Join(".", "conf", "config.yaml"),
		filepath.Join(".", "conf", "config.yml"),
		filepath.Join(".", "conf", "config.json"),
		filepath.Join(".", "conf", "config.toml"),
	}

	// 查找第一个存在的配置文件
	var configFile string
	for _, file := range configFiles {
		if fileExists(file) {
			configFile = file
			break
		}
	}

	// 如果没有找到配置文件，使用默认配置
	if configFile == "" {
		return Init(WithEnvPrefix(appName + "_"))
	}

	// 初始化配置
	return Init(
		WithConfigFile(configFile),
		WithEnvPrefix(appName+"_"),
	)
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	info, err := filepath.Glob(path)
	return err == nil && len(info) > 0
}
