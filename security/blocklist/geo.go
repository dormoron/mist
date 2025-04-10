package blocklist

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/dormoron/mist"
	"github.com/oschwald/geoip2-golang"
)

// 地理位置限制相关错误
var (
	// ErrGeoDBNotInitialized 表示GeoIP数据库未初始化
	ErrGeoDBNotInitialized = errors.New("GeoIP数据库未初始化")
	// ErrCountryNotFound 表示未找到国家信息
	ErrCountryNotFound = errors.New("无法确定IP的国家信息")
	// ErrIPBlocked 表示IP因地理位置限制被封禁
	ErrIPBlocked = errors.New("IP受到地理位置限制")
)

// GeoRestrictionMode 地理位置限制模式
type GeoRestrictionMode int

const (
	// AllowListMode 白名单模式 - 只允许指定国家/地区
	AllowListMode GeoRestrictionMode = iota
	// BlockListMode 黑名单模式 - 禁止指定国家/地区
	BlockListMode
)

// GeoRestriction 地理位置限制配置
type GeoRestriction struct {
	// Mode 限制模式
	Mode GeoRestrictionMode
	// Countries 国家代码列表
	Countries []string
	// countriesMap 快速查询的国家代码映射
	countriesMap map[string]bool
	// DB GeoIP2数据库
	DB *geoip2.Reader
	// mutex 互斥锁
	mutex sync.RWMutex
}

// NewGeoRestriction 创建地理位置限制实例
func NewGeoRestriction(mode GeoRestrictionMode, countries []string) *GeoRestriction {
	countriesMap := make(map[string]bool)
	for _, country := range countries {
		countriesMap[strings.ToUpper(country)] = true
	}

	return &GeoRestriction{
		Mode:         mode,
		Countries:    countries,
		countriesMap: countriesMap,
	}
}

// InitDBFromFile 从文件初始化GeoIP2数据库
func (g *GeoRestriction) InitDBFromFile(dbPath string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// 如果已有打开的数据库，先关闭
	if g.DB != nil {
		g.DB.Close()
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return err
	}

	g.DB = db
	return nil
}

// InitDBFromURL 从URL下载并初始化GeoIP2数据库
func (g *GeoRestriction) InitDBFromURL(url, savePath string) error {
	// 下载数据库文件
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 保存内容
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	// 初始化数据库
	return g.InitDBFromFile(savePath)
}

// Close 关闭GeoIP2数据库
func (g *GeoRestriction) Close() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.DB != nil {
		return g.DB.Close()
	}
	return nil
}

// GetCountryCode 获取IP地址的国家代码
func (g *GeoRestriction) GetCountryCode(ip string) (string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.DB == nil {
		return "", ErrGeoDBNotInitialized
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("无效的IP地址: %s", ip)
	}

	record, err := g.DB.Country(parsedIP)
	if err != nil {
		return "", err
	}

	if record.Country.IsoCode == "" {
		return "", ErrCountryNotFound
	}

	return record.Country.IsoCode, nil
}

// IsIPRestricted 检查IP是否受到地理位置限制
func (g *GeoRestriction) IsIPRestricted(ip string) (bool, error) {
	countryCode, err := g.GetCountryCode(ip)
	if err != nil {
		// 如果无法确定国家，根据配置决定处理方式
		if err == ErrCountryNotFound || err == ErrGeoDBNotInitialized {
			// 默认允许通过，可根据需求调整
			return false, nil
		}
		return false, err
	}

	// 国家代码转为大写以匹配
	countryCode = strings.ToUpper(countryCode)

	// 检查国家是否在列表中
	inList := g.countriesMap[countryCode]

	// 根据模式返回结果
	switch g.Mode {
	case AllowListMode:
		// 如果为白名单模式，在列表中的允许，否则限制
		return !inList, nil
	case BlockListMode:
		// 如果为黑名单模式，在列表中的限制，否则允许
		return inList, nil
	default:
		return false, nil
	}
}

// GeoMiddleware 创建基于地理位置的中间件
func (g *GeoRestriction) GeoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端IP
		ip := getClientIP(r)

		// 检查IP是否受到地理位置限制
		restricted, err := g.IsIPRestricted(ip)
		if err != nil {
			// 处理错误，默认允许通过
			next.ServeHTTP(w, r)
			return
		}

		if restricted {
			// IP受到限制，返回403错误
			http.Error(w, "Access denied based on your location", http.StatusForbidden)
			return
		}

		// 通过验证，继续处理请求
		next.ServeHTTP(w, r)
	})
}

// MistGeoMiddleware 创建基于地理位置的Mist中间件
func (g *GeoRestriction) MistGeoMiddleware(onRestricted func(*mist.Context)) mist.Middleware {
	if onRestricted == nil {
		onRestricted = func(ctx *mist.Context) {
			ctx.AbortWithStatus(http.StatusForbidden)
		}
	}

	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			// 获取客户端IP
			ip := getClientIP(ctx.Request)

			// 检查IP是否受到地理位置限制
			restricted, err := g.IsIPRestricted(ip)
			if err != nil {
				// 处理错误，默认允许通过
				next(ctx)
				return
			}

			if restricted {
				// IP受到限制，执行自定义处理
				onRestricted(ctx)
				return
			}

			// 通过验证，继续处理请求
			next(ctx)
		}
	}
}

// CountryInfo 国家信息结构体
type CountryInfo struct {
	// Code 国家代码
	Code string `json:"code"`
	// Name 国家名称
	Name string `json:"name"`
	// Continent 大陆
	Continent string `json:"continent"`
}

// GetIPInfo 获取IP详细信息
func (g *GeoRestriction) GetIPInfo(ip string) (*CountryInfo, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.DB == nil {
		return nil, ErrGeoDBNotInitialized
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址: %s", ip)
	}

	record, err := g.DB.Country(parsedIP)
	if err != nil {
		return nil, err
	}

	if record.Country.IsoCode == "" {
		return nil, ErrCountryNotFound
	}

	return &CountryInfo{
		Code:      record.Country.IsoCode,
		Name:      record.Country.Names["zh-CN"], // 中文名称，如果有
		Continent: record.Continent.Code,
	}, nil
}

// AddCountry 添加国家到限制列表
func (g *GeoRestriction) AddCountry(countryCode string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	countryCode = strings.ToUpper(countryCode)
	if !g.countriesMap[countryCode] {
		g.countriesMap[countryCode] = true
		g.Countries = append(g.Countries, countryCode)
	}
}

// RemoveCountry 从限制列表移除国家
func (g *GeoRestriction) RemoveCountry(countryCode string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	countryCode = strings.ToUpper(countryCode)
	if g.countriesMap[countryCode] {
		delete(g.countriesMap, countryCode)

		// 更新国家列表
		var newCountries []string
		for _, c := range g.Countries {
			if strings.ToUpper(c) != countryCode {
				newCountries = append(newCountries, c)
			}
		}
		g.Countries = newCountries
	}
}
