package cookie

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// 前缀类型常量
const (
	PrefixNone   = iota // 无前缀
	PrefixSecure        // __Secure- 前缀
	PrefixHost          // __Host- 前缀
)

// 分区类型常量
const (
	PartitionNone        = iota // 不分区
	PartitionPartitioned        // 分区Cookie (Partitioned)
)

// 错误定义
var (
	ErrInvalidCookieConfig  = errors.New("无效的Cookie配置")
	ErrInsecureCookieConfig = errors.New("不安全的Cookie配置")
)

// CookieOptions 定义Cookie选项
type CookieOptions struct {
	// 基本信息
	Name     string
	Value    string
	Path     string
	Domain   string
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite

	// 增强特性
	Prefix    int  // Cookie前缀类型 (PrefixNone, PrefixSecure, PrefixHost)
	Partition bool // 是否为分区Cookie
}

// DefaultCookieOptions 返回默认的安全Cookie选项
func DefaultCookieOptions() CookieOptions {
	return CookieOptions{
		Path:      "/",
		Secure:    true,
		HttpOnly:  true,
		SameSite:  http.SameSiteStrictMode,
		Prefix:    PrefixHost,
		Partition: false,
		MaxAge:    3600, // 1小时
	}
}

// Validate 验证Cookie选项是否有效
func (opts *CookieOptions) Validate() error {
	// 基本验证
	if opts.Name == "" {
		return errors.New("Cookie名称不能为空")
	}

	// __Host- 前缀验证
	if opts.Prefix == PrefixHost {
		if !opts.Secure {
			return errors.New("__Host- 前缀的Cookie必须设置Secure标志")
		}
		if opts.Path != "/" {
			return errors.New("__Host- 前缀的Cookie必须设置Path=/")
		}
		if opts.Domain != "" {
			return errors.New("__Host- 前缀的Cookie不能设置Domain")
		}
	}

	// __Secure- 前缀验证
	if opts.Prefix == PrefixSecure && !opts.Secure {
		return errors.New("__Secure- 前缀的Cookie必须设置Secure标志")
	}

	// 分区Cookie验证
	if opts.Partition && !opts.Secure {
		return errors.New("分区Cookie必须设置Secure标志")
	}

	return nil
}

// 安全性等级评估
func (opts *CookieOptions) SecurityLevel() int {
	level := 0

	// 基本安全特性
	if opts.Secure {
		level += 1
	}
	if opts.HttpOnly {
		level += 1
	}
	if opts.SameSite == http.SameSiteStrictMode {
		level += 2
	} else if opts.SameSite == http.SameSiteLaxMode {
		level += 1
	}

	// 高级安全特性
	if opts.Prefix == PrefixHost {
		level += 3
	} else if opts.Prefix == PrefixSecure {
		level += 2
	}

	if opts.Partition {
		level += 2
	}

	// Cookie生命周期
	if opts.MaxAge > 0 && opts.MaxAge < 86400 { // 不超过1天
		level += 1
	}

	return level
}

// SecurityLevelString 返回安全等级的描述
func (opts *CookieOptions) SecurityLevelString() string {
	level := opts.SecurityLevel()

	if level >= 8 {
		return "极高"
	} else if level >= 6 {
		return "高"
	} else if level >= 4 {
		return "中"
	} else if level >= 2 {
		return "低"
	}
	return "极低"
}

// ToCookie 将选项转换为http.Cookie
func (opts *CookieOptions) ToCookie() (*http.Cookie, error) {
	// 验证配置
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// 应用前缀
	name := opts.Name
	switch opts.Prefix {
	case PrefixSecure:
		name = "__Secure-" + name
	case PrefixHost:
		name = "__Host-" + name
	}

	// 创建Cookie
	cookie := &http.Cookie{
		Name:     name,
		Value:    opts.Value,
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   opts.MaxAge,
		Secure:   opts.Secure,
		HttpOnly: opts.HttpOnly,
		SameSite: opts.SameSite,
	}

	// 设置过期时间
	if !opts.Expires.IsZero() {
		cookie.Expires = opts.Expires
	}

	// 添加分区属性 (通过属性列表)
	if opts.Partition {
		if cookie.Raw != "" {
			cookie.Raw += "; Partitioned"
		} else {
			cookie.Raw = "Partitioned"
		}
	}

	return cookie, nil
}

// SetCookie 安全地设置Cookie
func SetCookie(w http.ResponseWriter, opts CookieOptions) error {
	cookie, err := opts.ToCookie()
	if err != nil {
		return err
	}
	http.SetCookie(w, cookie)
	return nil
}

// ParseCookieName 解析Cookie名，返回前缀类型和去除前缀的名称
func ParseCookieName(name string) (prefix int, originalName string) {
	if strings.HasPrefix(name, "__Host-") {
		return PrefixHost, strings.TrimPrefix(name, "__Host-")
	} else if strings.HasPrefix(name, "__Secure-") {
		return PrefixSecure, strings.TrimPrefix(name, "__Secure-")
	}
	return PrefixNone, name
}

// ValidateCookieSecurity 验证Cookie安全性
func ValidateCookieSecurity(cookie *http.Cookie) error {
	prefix, _ := ParseCookieName(cookie.Name)

	// __Host- 前缀验证
	if prefix == PrefixHost {
		if !cookie.Secure {
			return errors.New("__Host- 前缀的Cookie必须设置Secure标志")
		}
		if cookie.Path != "/" {
			return errors.New("__Host- 前缀的Cookie必须设置Path=/")
		}
		if cookie.Domain != "" {
			return errors.New("__Host- 前缀的Cookie不能设置Domain")
		}
	}

	// __Secure- 前缀验证
	if prefix == PrefixSecure && !cookie.Secure {
		return errors.New("__Secure- 前缀的Cookie必须设置Secure标志")
	}

	return nil
}

// IsPartitioned 检查Cookie是否为分区Cookie
func IsPartitioned(cookie *http.Cookie) bool {
	if cookie.Raw == "" {
		return false
	}
	return strings.Contains(cookie.Raw, "Partitioned")
}

// CreateSecureCookie 创建安全Cookie
func CreateSecureCookie(name, value string, maxAge int) *http.Cookie {
	opts := DefaultCookieOptions()
	opts.Name = name
	opts.Value = value
	opts.MaxAge = maxAge

	cookie, _ := opts.ToCookie()
	return cookie
}

// CreateHostPrefixCookie 创建使用__Host-前缀的Cookie
func CreateHostPrefixCookie(name, value string, maxAge int) *http.Cookie {
	opts := DefaultCookieOptions()
	opts.Name = name
	opts.Value = value
	opts.MaxAge = maxAge
	opts.Prefix = PrefixHost

	cookie, _ := opts.ToCookie()
	return cookie
}

// CreateSecurePrefixCookie 创建使用__Secure-前缀的Cookie
func CreateSecurePrefixCookie(name, value string, maxAge int) *http.Cookie {
	opts := DefaultCookieOptions()
	opts.Name = name
	opts.Value = value
	opts.MaxAge = maxAge
	opts.Prefix = PrefixSecure

	cookie, _ := opts.ToCookie()
	return cookie
}

// CreatePartitionedCookie 创建分区Cookie
func CreatePartitionedCookie(name, value string, maxAge int) *http.Cookie {
	opts := DefaultCookieOptions()
	opts.Name = name
	opts.Value = value
	opts.MaxAge = maxAge
	opts.Partition = true

	cookie, _ := opts.ToCookie()
	return cookie
}
