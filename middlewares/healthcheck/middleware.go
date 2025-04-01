package healthcheck

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dormoron/mist"
)

// Status 表示服务健康状态
type Status string

const (
	StatusUp      Status = "UP"
	StatusDown    Status = "DOWN"
	StatusUnknown Status = "UNKNOWN"
)

// ComponentCheck 表示一个组件检查函数
type ComponentCheck func() (Status, map[string]any)

// HealthResponse 表示健康检查响应
type HealthResponse struct {
	Status     Status                     `json:"status"`
	Components map[string]ComponentStatus `json:"components,omitempty"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version,omitempty"`
}

// ComponentStatus 表示组件状态
type ComponentStatus struct {
	Status  Status         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

// Middleware 健康检查中间件
type Middleware struct {
	mu              sync.RWMutex
	path            string
	version         string
	components      map[string]ComponentCheck
	statusCache     atomic.Value // 缓存的健康状态
	cacheTimeout    time.Duration
	lastCheckTime   time.Time
	readinessChecks map[string]ComponentCheck
	livenessChecks  map[string]ComponentCheck
}

// InitMiddleware 创建新的健康检查中间件
func InitMiddleware(path string) *Middleware {
	if path == "" {
		path = "/health"
	}

	m := &Middleware{
		path:            path,
		components:      make(map[string]ComponentCheck),
		readinessChecks: make(map[string]ComponentCheck),
		livenessChecks:  make(map[string]ComponentCheck),
		cacheTimeout:    5 * time.Second,
	}

	// 添加默认健康检查
	m.RegisterComponent("self", func() (Status, map[string]any) {
		return StatusUp, map[string]any{
			"message": "Service is running",
		}
	})

	// 初始化缓存
	m.statusCache.Store(HealthResponse{
		Status:    StatusUnknown,
		Timestamp: time.Now(),
	})

	return m
}

// RegisterComponent 注册组件健康检查
func (m *Middleware) RegisterComponent(name string, check ComponentCheck) *Middleware {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.components[name] = check
	return m
}

// RegisterReadinessCheck 注册就绪检查
func (m *Middleware) RegisterReadinessCheck(name string, check ComponentCheck) *Middleware {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readinessChecks[name] = check
	return m
}

// RegisterLivenessCheck 注册存活检查
func (m *Middleware) RegisterLivenessCheck(name string, check ComponentCheck) *Middleware {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessChecks[name] = check
	return m
}

// SetVersion 设置服务版本
func (m *Middleware) SetVersion(version string) *Middleware {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.version = version
	return m
}

// SetCacheTimeout 设置缓存超时时间
func (m *Middleware) SetCacheTimeout(timeout time.Duration) *Middleware {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheTimeout = timeout
	return m
}

// performChecks 执行健康检查
func (m *Middleware) performChecks(checks map[string]ComponentCheck) (HealthResponse, bool) {
	result := HealthResponse{
		Status:     StatusUp,
		Components: make(map[string]ComponentStatus),
		Timestamp:  time.Now(),
		Version:    m.version,
	}

	allUp := true

	// 执行所有组件检查
	for name, check := range checks {
		status, details := check()
		result.Components[name] = ComponentStatus{
			Status:  status,
			Details: details,
		}

		if status != StatusUp {
			allUp = false
			// 如果任何组件不健康，则整体状态为DOWN
			if result.Status != StatusDown {
				result.Status = status
			}
		}
	}

	// 如果没有组件检查，则状态为UNKNOWN
	if len(checks) == 0 {
		result.Status = StatusUnknown
		allUp = false
	}

	return result, allUp
}

// Build 构建健康检查中间件
func (m *Middleware) Build() mist.Middleware {
	return func(next mist.HandleFunc) mist.HandleFunc {
		return func(ctx *mist.Context) {
			path := ctx.Request.URL.Path

			// 处理主健康检查
			if path == m.path {
				// 检查缓存是否有效
				m.mu.RLock()
				needFreshCheck := time.Since(m.lastCheckTime) > m.cacheTimeout
				m.mu.RUnlock()

				var resp HealthResponse
				if needFreshCheck {
					m.mu.Lock()
					// 双重检查，避免多个并发请求都执行检查
					if time.Since(m.lastCheckTime) > m.cacheTimeout {
						resp, _ = m.performChecks(m.components)
						m.statusCache.Store(resp)
						m.lastCheckTime = time.Now()
					} else {
						resp = m.statusCache.Load().(HealthResponse)
					}
					m.mu.Unlock()
				} else {
					resp = m.statusCache.Load().(HealthResponse)
				}

				data, err := json.Marshal(resp)
				if err != nil {
					ctx.RespStatusCode = http.StatusInternalServerError
					ctx.RespData = []byte(`{"error":"Failed to generate health check response"}`)
					return
				}

				ctx.RespStatusCode = http.StatusOK
				if resp.Status != StatusUp {
					ctx.RespStatusCode = http.StatusServiceUnavailable
				}
				ctx.RespData = data
				ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
				return
			}

			// 处理就绪检查
			if path == m.path+"/readiness" {
				resp, allUp := m.performChecks(m.readinessChecks)
				data, err := json.Marshal(resp)
				if err != nil {
					ctx.RespStatusCode = http.StatusInternalServerError
					ctx.RespData = []byte(`{"error":"Failed to generate readiness check response"}`)
					return
				}

				ctx.RespStatusCode = http.StatusOK
				if !allUp {
					ctx.RespStatusCode = http.StatusServiceUnavailable
				}
				ctx.RespData = data
				ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
				return
			}

			// 处理存活检查
			if path == m.path+"/liveness" {
				resp, allUp := m.performChecks(m.livenessChecks)
				data, err := json.Marshal(resp)
				if err != nil {
					ctx.RespStatusCode = http.StatusInternalServerError
					ctx.RespData = []byte(`{"error":"Failed to generate liveness check response"}`)
					return
				}

				ctx.RespStatusCode = http.StatusOK
				if !allUp {
					ctx.RespStatusCode = http.StatusServiceUnavailable
				}
				ctx.RespData = data
				ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
				return
			}

			// 继续处理非健康检查路径
			next(ctx)
		}
	}
}
