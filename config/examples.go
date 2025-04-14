package config

import (
	"fmt"
	"time"
)

// 示例1: 基本配置使用
func Example_basic() {
	// 创建一个新的配置管理器
	cfg, err := New(
		WithConfigFile("config.yaml"),
		WithEnvPrefix("APP_"),
	)
	if err != nil {
		panic(err)
	}

	// 设置一些配置值
	cfg.Set("app.name", "MyApp")
	cfg.Set("app.version", "1.0.0")
	cfg.Set("server.port", 8080)
	cfg.Set("server.timeout", 30)

	// 获取配置值
	appName := cfg.GetString("app.name")
	port := cfg.GetInt("server.port")

	fmt.Printf("App: %s, Port: %d\n", appName, port)

	// 检查配置是否存在
	if cfg.Has("app.debug") {
		fmt.Println("Debug mode is configured")
	} else {
		fmt.Println("Debug mode is not configured")
	}

	// 获取所有配置
	settings := cfg.AllSettings()
	fmt.Printf("All settings: %+v\n", settings)
}

// AppConfig 应用配置结构体
type AppConfig struct {
	Name    string `config:"name"`
	Version string `config:"version"`
	Debug   bool   `config:"debug"`
	Server  struct {
		Port    int           `config:"port"`
		Timeout time.Duration `config:"timeout"`
		Host    string        `config:"host"`
	} `config:"server"`
	Database struct {
		DSN      string `config:"dsn"`
		MaxConns int    `config:"max_conns"`
		MaxIdle  int    `config:"max_idle"`
	} `config:"database"`
	Features []string               `config:"features"`
	Options  map[string]interface{} `config:"options"`
}

// 示例2: 配置结构体映射
func Example_unmarshal() {
	// 创建配置管理器
	cfg, _ := New()

	// 设置一些配置值
	cfg.Set("app", map[string]interface{}{
		"name":    "MyApp",
		"version": "1.0.0",
		"debug":   true,
		"server": map[string]interface{}{
			"port":    8080,
			"timeout": 30,
			"host":    "localhost",
		},
		"database": map[string]interface{}{
			"dsn":       "postgres://user:pass@localhost:5432/mydb",
			"max_conns": 100,
			"max_idle":  10,
		},
		"features": []string{"auth", "api", "dashboard"},
		"options": map[string]interface{}{
			"theme":    "dark",
			"language": "en",
		},
	})

	// 映射到结构体
	var appConfig AppConfig
	if err := cfg.Unmarshal("app", &appConfig); err != nil {
		panic(err)
	}

	// 使用配置结构体
	fmt.Printf("App: %s (v%s)\n", appConfig.Name, appConfig.Version)
	fmt.Printf("Server: %s:%d (timeout: %v)\n",
		appConfig.Server.Host,
		appConfig.Server.Port,
		appConfig.Server.Timeout)
	fmt.Printf("Database: %s (max connections: %d)\n",
		appConfig.Database.DSN,
		appConfig.Database.MaxConns)

	fmt.Println("Features:")
	for _, feature := range appConfig.Features {
		fmt.Printf("  - %s\n", feature)
	}
}

// 示例3: 配置变更监听
func Example_listener() {
	// 创建配置管理器
	cfg, _ := New()

	// 添加配置变更监听器
	cfg.AddChangeListener(func(key string) {
		if key == "" {
			fmt.Println("全局配置已变更")
		} else {
			fmt.Printf("配置项已变更: %s\n", key)

			// 获取新值
			value, _ := cfg.Get(key)
			fmt.Printf("  新值: %v\n", value)
		}
	})

	// 修改配置
	cfg.Set("app.name", "NewName")
	cfg.Set("server.port", 9000)

	// 输出:
	// 配置项已变更: app.name
	//   新值: NewName
	// 配置项已变更: server.port
	//   新值: 9000
}

// 以下是配置文件示例

/*
YAML配置文件示例 (config.yaml):

app:
  name: MyApp
  version: 1.0.0
  debug: true

server:
  port: 8080
  host: localhost
  timeout: 30s

database:
  dsn: postgres://user:pass@localhost:5432/mydb
  max_conns: 100
  max_idle: 10

features:
  - auth
  - api
  - dashboard

options:
  theme: dark
  language: en

*/

/*
JSON配置文件示例 (config.json):

{
  "app": {
    "name": "MyApp",
    "version": "1.0.0",
    "debug": true
  },
  "server": {
    "port": 8080,
    "host": "localhost",
    "timeout": "30s"
  },
  "database": {
    "dsn": "postgres://user:pass@localhost:5432/mydb",
    "max_conns": 100,
    "max_idle": 10
  },
  "features": [
    "auth",
    "api",
    "dashboard"
  ],
  "options": {
    "theme": "dark",
    "language": "en"
  }
}

*/

/*
TOML配置文件示例 (config.toml):

[app]
name = "MyApp"
version = "1.0.0"
debug = true

[server]
port = 8080
host = "localhost"
timeout = "30s"

[database]
dsn = "postgres://user:pass@localhost:5432/mydb"
max_conns = 100
max_idle = 10

features = ["auth", "api", "dashboard"]

[options]
theme = "dark"
language = "en"

*/
