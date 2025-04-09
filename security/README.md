# Mist框架安全模块

Mist框架安全模块提供了全面的Web应用安全功能，包括会话管理、认证、CSRF保护、限流、安全HTTP头、密码处理以及多因素认证等。

## 主要功能

- **会话管理**：支持Redis后端的会话存储、JWT令牌
- **身份认证**：基于JWT的身份验证系统 
- **CSRF保护**：防止跨站请求伪造攻击
- **请求限流**：防止API滥用和DoS攻击
- **安全HTTP头**：自动添加安全相关的HTTP头
- **密码处理**：安全的密码哈希和验证
- **安全配置**：支持不同安全级别（基础、中级、严格）的预设配置
- **多因素认证(MFA)**：支持基于TOTP的两步验证
- **IP黑名单**：防止暴力破解攻击

## 快速开始

### 设置安全级别

```go
package main

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
)

func main() {
	// 设置安全级别（基础、中级、严格）
	security.SetSecurityLevel(security.LevelIntermediate)
	
	// 创建应用实例
	app := mist.Default()
	
	// 路由和其他配置...
	
	app.Run(":8080")
}
```

### 使用CSRF保护

```go
package main

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/csrf"
)

func main() {
	app := mist.Default()
	
	// 添加CSRF中间件
	app.Use(csrf.New())
	
	// 自定义CSRF保护配置
	app.Use(csrf.New(
		csrf.WithTokenLength(64),
		csrf.WithCookieName("my_csrf_token"),
		csrf.WithHeaderName("X-My-CSRF-Token"),
	))
	
	app.Run(":8080")
}
```

### 使用限流功能

```go
package main

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/ratelimit"
)

func main() {
	app := mist.Default()
	
	// 创建内存限流器（每秒10个请求，突发20个请求）
	limiter := ratelimit.NewMemoryLimiter(10, 20)
	
	// 添加限流中间件
	app.Use(ratelimit.New(limiter,
		ratelimit.WithRate(10),
		ratelimit.WithBurst(20),
		ratelimit.WithKeyExtractor(ratelimit.IPKeyExtractor),
	))
	
	// 为特定API路由添加更严格的限流
	apiGroup := app.Group("/api")
	apiLimiter := ratelimit.NewMemoryLimiter(5, 10)
	apiGroup.Use(ratelimit.New(apiLimiter))
	
	app.Run(":8080")
}
```

### 使用安全HTTP头

```go
package main

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/headers"
)

func main() {
	app := mist.Default()
	
	// 添加安全头部中间件（使用默认配置）
	app.Use(headers.New())
	
	// 自定义安全头部配置
	app.Use(headers.New(
		headers.WithXSSProtection(true),
		headers.WithHSTS(true, 63072000, true, true),
		headers.WithContentSecurityPolicy(headers.CSPStrict()),
	))
	
	app.Run(":8080")
}
```

### 使用密码处理

```go
package main

import (
	"fmt"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/password"
)

func registerHandler(ctx *mist.Context) {
	username := ctx.PostForm("username")
	plainPassword := ctx.PostForm("password")
	
	// 检查密码强度
	strength := password.CheckPasswordStrength(plainPassword)
	if strength < password.Medium {
		ctx.AbortWithStatus(400) // 密码强度不足
		return
	}
	
	// 哈希密码
	hashedPassword, err := password.HashPassword(plainPassword)
	if err != nil {
		ctx.AbortWithStatus(500) // 密码处理错误
		return
	}
	
	// 存储用户信息...
	fmt.Println("哈希后的密码:", hashedPassword)
}

func loginHandler(ctx *mist.Context) {
	username := ctx.PostForm("username")
	plainPassword := ctx.PostForm("password")
	
	// 从数据库获取哈希后的密码...
	storedHash := getPasswordFromDatabase(username)
	
	// 验证密码
	err := password.CheckPassword(plainPassword, storedHash)
	if err != nil {
		ctx.AbortWithStatus(401) // 用户名或密码错误
		return
	}
	
	// 登录成功，创建会话...
}
```

### 会话管理

```go
package main

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
	"github.com/dormoron/mist/security/redisess"
	"github.com/redis/go-redis/v9"
)

func main() {
	app := mist.Default()
	
	// 配置Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// 创建基于Redis的会话提供者
	provider := redisess.InitSessionProvider(redisClient, "your-jwt-secret-key")
	
	// 设置全局会话提供者
	security.SetDefaultProvider(provider)
	
	// 添加登录检查中间件
	app.Use(security.CheckLoginMiddleware("/admin", "/profile", "/api"))
	
	app.Run(":8080")
}
```

### 使用多因素认证(MFA)

```go
package main

import (
	"time"
	
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/mfa"
)

func main() {
	app := mist.Default()
	
	// 创建内存存储的MFA验证状态管理器
	store := mfa.NewMemoryStore()
	
	// 创建MFA中间件
	mfaMiddleware := mfa.NewMiddleware(
		mfa.WithStore(store),
		mfa.WithValidationDuration(24 * time.Hour),
		mfa.WithRedirectURL("/mfa/validate"),
	)
	
	// 应用MFA中间件到需要保护的路由
	adminGroup := app.Group("/admin")
	adminGroup.Use(mfaMiddleware)
	
	// MFA验证处理
	app.POST("/mfa/validate", func(ctx *mist.Context) {
		userID := getUserID(ctx) // 从会话或上下文中获取用户ID
		code := ctx.PostForm("code")
		
		// 从数据库获取用户的TOTP密钥
		secretKey := getUserTOTPSecret(userID)
		
		// 创建TOTP实例
		totp := mfa.NewTOTPWithSecret(secretKey)
		
		// 验证TOTP代码
		err := mfa.Validate(ctx, userID, code, totp, store, 24*time.Hour)
		if err != nil {
			ctx.AbortWithStatus(400) // 验证码无效
			return
		}
		
		// 验证成功，重定向到原始目标页面
		ctx.Redirect(302, "/admin")
	})
	
	// 生成TOTP二维码链接
	app.GET("/mfa/setup", func(ctx *mist.Context) {
		userID := getUserID(ctx)
		
		// 创建新的TOTP实例
		totp, err := mfa.NewTOTP()
		if err != nil {
			ctx.AbortWithStatus(500)
			return
		}
		
		// 存储用户的TOTP密钥到数据库
		saveUserTOTPSecret(userID, totp.Secret)
		
		// 生成配置URI（用于QR码）
		provisioningURI := totp.ProvisioningURI(userID)
		
		// 返回URI（前端可以用它生成QR码）
		ctx.JSON(200, map[string]string{
			"uri": provisioningURI,
		})
	})
	
	app.Run(":8080")
}
```

### 使用IP黑名单防止暴力破解

```go
package main

import (
	"time"
	
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security/blocklist"
)

func main() {
	app := mist.Default()
	
	// 创建IP黑名单管理器
	blocklistManager := blocklist.NewManager(
		blocklist.WithMaxFailedAttempts(5),        // 5次失败尝试后封禁
		blocklist.WithBlockDuration(30*time.Minute), // 封禁30分钟
		blocklist.WithWhitelistIPs([]string{"127.0.0.1"}), // 白名单IP
	)
	
	// 添加IP黑名单中间件
	app.Use(blocklistManager.Middleware())
	
	// 登录处理
	app.POST("/login", func(ctx *mist.Context) {
		username := ctx.PostForm("username")
		password := ctx.PostForm("password")
		
		// 获取客户端IP
		ip := ctx.ClientIP()
		
		// 验证用户名和密码
		if validateCredentials(username, password) {
			// 登录成功，记录成功并重置失败计数
			blocklistManager.RecordSuccess(ip)
			// 处理登录成功...
		} else {
			// 登录失败，记录失败
			blocked := blocklistManager.RecordFailure(ip)
			if blocked {
				// IP已被封禁，返回特定错误
				ctx.AbortWithStatus(403) // 已被封禁
			} else {
				// 未被封禁，但登录失败
				ctx.AbortWithStatus(401) // 用户名或密码错误
			}
		}
	})
	
	// 手动封禁IP
	app.POST("/admin/block-ip", func(ctx *mist.Context) {
		ip := ctx.PostForm("ip")
		duration := 24 * time.Hour // 封禁24小时
		
		blocklistManager.BlockIP(ip, duration)
		ctx.String(200, "IP已被封禁")
	})
	
	// 手动解除IP封禁
	app.POST("/admin/unblock-ip", func(ctx *mist.Context) {
		ip := ctx.PostForm("ip")
		
		blocklistManager.UnblockIP(ip)
		ctx.String(200, "IP封禁已解除")
	})
	
	app.Run(":8080")
}
```

## 安全配置

Mist框架提供了三种预设的安全级别：

1. **LevelBasic**：基础安全级别，适合开发环境或不太敏感的应用
2. **LevelIntermediate**：中级安全级别，适合一般Web应用（默认）
3. **LevelStrict**：严格安全级别，适合处理敏感数据的应用

可以通过`security.SetSecurityLevel()`设置全局安全级别，或创建自定义配置：

```go
// 自定义安全配置
customConfig := security.DefaultSecurityConfig(security.LevelIntermediate)
customConfig.Password.MinLength = 12
customConfig.Session.AccessTokenExpiry = 30 * time.Minute
customConfig.CSRF.TokenLength = 64

// 应用自定义配置
security.SetSecurityConfig(customConfig)
```

## 最佳实践

1. 为生产环境使用至少`LevelIntermediate`安全级别
2. 确保所有敏感操作受CSRF保护
3. 为关键API端点设置适当的速率限制
4. 对所有用户密码使用`password`包提供的哈希功能
5. 启用安全HTTP头以增强前端安全性
6. 为敏感账户启用多因素认证
7. 使用IP黑名单防止暴力破解攻击
8. 定期轮换JWT密钥和会话密钥

## 完整示例

```go
package main

import (
	"time"
	
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/security"
	"github.com/dormoron/mist/security/csrf"
	"github.com/dormoron/mist/security/headers"
	"github.com/dormoron/mist/security/ratelimit"
	"github.com/dormoron/mist/security/redisess"
	"github.com/dormoron/mist/security/blocklist"
	"github.com/dormoron/mist/security/mfa"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 设置安全级别
	security.SetSecurityLevel(security.LevelStrict)
	
	// 创建应用实例
	app := mist.Default()
	
	// 配置Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	// 会话管理
	provider := redisess.InitSessionProvider(redisClient, "your-secret-key")
	security.SetDefaultProvider(provider)
	
	// IP黑名单
	blocklistManager := blocklist.NewManager(
		blocklist.WithMaxFailedAttempts(5),
		blocklist.WithBlockDuration(30*time.Minute),
	)
	
	// MFA验证
	mfaStore := mfa.NewMemoryStore()
	mfaMiddleware := mfa.NewMiddleware(
		mfa.WithStore(mfaStore),
		mfa.WithValidationDuration(24*time.Hour),
	)
	
	// 添加安全中间件
	
	// 1. IP黑名单
	app.Use(blocklistManager.Middleware())
	
	// 2. 安全HTTP头
	app.Use(headers.New(
		headers.WithContentSecurityPolicy(headers.CSPStrict()),
	))
	
	// 3. CSRF保护
	app.Use(csrf.New())
	
	// 4. 全局限流（每秒100请求）
	globalLimiter := ratelimit.NewMemoryLimiter(100, 200)
	app.Use(ratelimit.New(globalLimiter))
	
	// 5. 登录检查
	app.Use(security.CheckLoginMiddleware("/admin", "/profile", "/api"))
	
	// API路由组
	apiGroup := app.Group("/api")
	
	// API特定限流（每秒10请求）
	apiLimiter := ratelimit.NewMemoryLimiter(10, 20) 
	apiGroup.Use(ratelimit.New(apiLimiter))
	
	// 管理员路由组（需要MFA验证）
	adminGroup := app.Group("/admin")
	adminGroup.Use(mfaMiddleware)
	
	// 设置路由
	app.GET("/", func(ctx *mist.Context) {
		ctx.String(200, "Mist Framework with Security")
	})
	
	// 登录处理
	app.POST("/login", handleLogin(blocklistManager))
	
	// MFA相关路由
	app.POST("/mfa/validate", handleMFAValidation(mfaStore))
	app.GET("/mfa/setup", handleMFASetup())
	
	// 启动服务器
	app.Run(":8080")
}

// 登录处理函数
func handleLogin(bm *blocklist.Manager) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// 登录逻辑...
	}
}

// MFA验证处理函数
func handleMFAValidation(store mfa.ValidationStore) mist.HandleFunc {
	return func(ctx *mist.Context) {
		// MFA验证逻辑...
	}
}

// MFA设置处理函数
func handleMFASetup() mist.HandleFunc {
	return func(ctx *mist.Context) {
		// MFA设置逻辑...
	}
}
```

## 4. IP 黑名单（Blocklist）

Mist框架提供了IP黑名单功能，用于限制恶意IP访问系统，防止暴力破解等攻击。

### 4.1 快速开始

```go
import (
    "github.com/dormoron/mist"
    "github.com/dormoron/mist/security/blocklist"
    "github.com/dormoron/mist/security/blocklist/middleware"
    "time"
)

// 创建IP黑名单管理器
blocklistManager := blocklist.NewManager(
    blocklist.WithMaxFailedAttempts(5),          // 最大失败尝试次数
    blocklist.WithBlockDuration(15*time.Minute), // 封禁时长
    blocklist.WithWhitelistIPs([]string{"127.0.0.1"}), // 白名单IP
)

// 在Mist框架中使用（推荐方式）
app := mist.New()
// 使用默认配置（返回403 Forbidden状态码）
app.Use(middleware.New(blocklistManager))

// 使用自定义处理函数
app.Use(middleware.New(
    blocklistManager,
    middleware.WithOnBlocked(func(ctx *mist.Context) {
        ctx.AbortWithStatus(http.StatusForbidden)
        // 或者返回JSON响应
        // ctx.RespondWithJSON(http.StatusForbidden, map[string]string{
        //     "error": "您的IP已被暂时封禁，请稍后再试",
        // })
    }),
))

// 在标准HTTP服务中使用
http.Handle("/api", blocklistManager.Middleware()(yourHandler))
```

### 4.2 配置选项

```go
// 创建具有自定义配置的IP黑名单管理器
manager := blocklist.NewManager(
    // 设置最大失败尝试次数，超过后IP将被封禁
    blocklist.WithMaxFailedAttempts(3),
    
    // 设置封禁时长
    blocklist.WithBlockDuration(30*time.Minute),
    
    // 设置清理过期记录的间隔
    blocklist.WithClearInterval(10*time.Minute),
    
    // 设置IP白名单，这些IP不会被封禁
    blocklist.WithWhitelistIPs([]string{"127.0.0.1", "192.168.1.1"}),
    
    // 设置封禁时的响应处理函数（标准HTTP中间件）
    blocklist.WithOnBlocked(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusForbidden)
        w.Write([]byte("您的IP已被暂时封禁，请稍后再试"))
    }),
)

// 在Mist框架中使用自定义封禁处理函数
app.Use(manager.MistMiddleware(
    blocklist.WithMistOnBlocked(func(ctx *mist.Context) {
        // 记录IP被封禁事件
        log.Printf("IP %s 因多次失败尝试被封禁", ctx.ClientIP())
        
        // 返回自定义错误响应
        ctx.RespondWithJSON(http.StatusForbidden, map[string]string{
            "error": "您的IP因多次失败的尝试已被封禁",
            "retry_after": "30分钟后再试",
        })
    }),
))
```

### 4.3 记录登录失败和成功

在登录过程中，您可以使用以下方法记录成功和失败的登录尝试：

```go
// 处理登录请求
func handleLogin(w http.ResponseWriter, r *http.Request) {
    ip := getClientIP(r) // 获取客户端IP
    
    // 检查IP是否已被封禁
    if blocklistManager.IsBlocked(ip) {
        http.Error(w, "您的IP已被封禁，请稍后再试", http.StatusForbidden)
        return
    }
    
    // 执行验证...
    if loginSuccessful {
        // 记录成功的登录，重置失败计数
        blocklistManager.RecordSuccess(ip)
        // 继续正常登录流程...
    } else {
        // 记录失败的登录，增加失败计数
        isBlocked := blocklistManager.RecordFailure(ip)
        if isBlocked {
            http.Error(w, "您的IP因多次失败的尝试已被封禁", http.StatusForbidden)
        } else {
            http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
        }
    }
}
```

### 4.4 手动管理IP封禁

您可以手动封禁和解除封禁IP：

```go
// 手动封禁IP，指定封禁时长
blocklistManager.BlockIP("192.168.1.100", 2*time.Hour)

// 解除IP封禁
blocklistManager.UnblockIP("192.168.1.100")

// 检查IP是否被封禁
isBlocked := blocklistManager.IsBlocked("192.168.1.100")
``` 