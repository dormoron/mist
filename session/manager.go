package session

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/mist"
	"github.com/dormoron/mist/session/cookie"
	"github.com/google/uuid"
)

// The Manager struct acts as a centralized component that orchestrates the session management
// within a web application. By embedding the Store and Propagator interfaces, it seamlessly combines
// the functionality of session storage and identifier propagation. The Manager handles the complete lifecycle
// of a session, which includes creating sessions, storing them, transmitting session identifiers in HTTP
// messages, and cleaning up as needed. This higher-level struct allows for simplified session management
// throughout the application, as it encapsulates all the necessary operations within a single entity.
//
// Fields:
//   - CtxSessionKey: A string that represents the key under which the session object is stored in
//     the context of an HTTP request. This allows middleware and handlers to retrieve the session
//     info from the context using this key, facilitating a standard way of accessing session data
//     during the processing of a request.
//   - autoGC: A bool indicating whether automatic garbage collection is enabled to
//     periodically clean up expired sessions.
//   - gcInterval: The duration between garbage collection runs.
//   - gcCtx: Context for the garbage collection goroutine, used to cancel it when needed.
//   - gcCancel: Function to cancel the garbage collection context.
//   - mutex: Mutex to protect concurrent access to the Manager's state.
//
// The inclusion of both the Store and Propagator interfaces suggests that any instance of Manager is
// capable of performing all session-related operations defined by these interfaces. This includes generating
// and managing session data (through the Store interface) and handling the session identifiers across
// HTTP requests and responses (through the Propagator interface).
//
// Here is an example of how the Manager struct could be initialized and used within an application:
//
//	func main() {
//	    // Initialize the session manager with specific implementations of Store and Propagator.
//	    sessionManager := &Manager{
//	        Store: NewRedisStore(), // assuming NewRedisStore returns an implementation of Store
//	        Propagator: NewCookiePropagator("session_id"), // assuming NewCookiePropagator returns an implementation of Propagator
//	        CtxSessionKey: "session", // the key used to store session objects in context
//	    }
//
//	    // Enable automatic garbage collection every 10 minutes
//	    sessionManager.EnableAutoGC(10 * time.Minute)
//
//	    // Set up your HTTP server, routes, middleware, etc.,
//	    // and use sessionManager to manage sessions in your application.
//	}
//
// Through Manager, all handlers and middleware in the application can interact with sessions
// using a standardized interface without worrying about the underlying storage or communication
// mechanisms, which are abstracted away by the implementations of Store and Propagator.
//
// Implementers of the Manager struct should ensure that necessary synchronizations or concurrent
// access handling are considered in their implementations of Store and Propagator to prevent race
// conditions or data inconsistencies.
type Manager struct {
	Store                // Handles storage and retrieval of session data.
	Propagator           // Manages transmission of session identifiers in HTTP messages.
	CtxSessionKey string // Key for session object storage in request context.

	// Automatic garbage collection configuration
	autoGC     bool               // Flag indicating if automatic GC is enabled
	gcInterval time.Duration      // Interval between GC runs
	gcCtx      context.Context    // Context for the GC goroutine
	gcCancel   context.CancelFunc // Function to cancel the GC context
	mutex      sync.Mutex         // Mutex to protect concurrent access to Manager state
}

// GetSession is a method that retrieves the current user's session from the HTTP request
// and caches it in the context for future use within the scope of the current request processing.
// This method provides a single entry point for session retrieval, and ensures that the session
// is loaded only once per request, thereby improving performance and reducing redundant operations.
//
// The flow is as follows:
//  1. It first checks if the UserValues map within the mist.Context is initialized. If not,
//     it initializes the map to store the session object later in the process.
//  2. The method then tries to retrieve the session from the UserValues map using the CtxSessionKey
//     defined in the Manager struct. This is to check if the session was already fetched and cached
//     earlier in the current request lifecycle.
//  3. If the session is found in the map, it is returned immediately, avoiding any further operations.
//  4. If not, the method utilizes the Propagator interface's Extract method to retrieve the session
//     identifier from the incoming HTTP request, which is typically read from a cookie or request header.
//  5. With the session identifier obtained, the method then fetches the actual session data using the
//     Store interface's Get method. This method call also passes along the context from the request to handle
//     any session-related context operations such as deadlines or cancellations.
//  6. After the session is successfully retrieved, it is stored in the UserValues map using the CtxSessionKey
//     for quick access during subsequent calls within the same request lifecycle.
//  7. Finally, the actual session data or an error (if any occurred while retrieving the session identifier
//     or the session data) is returned.
//
// If at any point there is a failure to retrieve the session identifier or the session data, an error
// is returned to the caller. This method centralizes error handling related to session retrieval, which
// simplifies the session logic elsewhere in the application.
//
// The mist.Context is assumed to be a custom HTTP context that contains both the standard library context
// and additional data fields used for managing user-specific values within a single request lifecycle.
//
// Usage:
// This method should be called by middlewares or handlers that require access to the current user's session.
// It exempts them from having to handle low-level session extraction and storage mechanisms directly.
func (m *Manager) GetSession(ctx *mist.Context) (Session, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context provided to GetSession")
	}

	// Ensure the map used to store values in the context is initialized.
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}

	// Attempt to retrieve the session from the cache in the user values map.
	val, ok := ctx.UserValues[m.CtxSessionKey]
	if ok {
		if sess, valid := val.(Session); valid {
			return sess, nil
		}
		// If the value is not a Session, remove it
		delete(ctx.UserValues, m.CtxSessionKey)
	}

	// Session not found in cache, so extract the session ID from the HTTP request.
	sessId, err := m.Propagator.Extract(ctx.Request)
	if err != nil {
		return nil, err
	}

	if sessId == "" {
		return nil, fmt.Errorf("empty session ID extracted from request")
	}

	// Retrieve the session data using the extracted session ID.
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	session, err := m.Store.Get(reqCtx, sessId)
	if err != nil {
		return nil, err
	}

	// Store the session in the map for quick access during this request lifecycle.
	ctx.UserValues[m.CtxSessionKey] = session
	return session, nil
}

// InitSession is responsible for creating a new session and associating it with the client who initiated
// the HTTP request. It is typically called when a new user visits the application and a new session needs to
// be established. The method leverages the capabilities of the embedded interfaces within the Manager struct
// to generate a unique session identifier, create a new session, and transmit this session identifier back to
// the client for future interactions.
//
// The process involves the following steps:
//  1. Generate a new unique identifier for the session using a universally unique identifier (UUID) library.
//  2. With the new session identifier, the method calls the Generate method of the embedded Store interface
//     to actually create a new session in the session store. This session creation is supposed to associate
//     the generated UUID with a new session object and store it in whatever storage mechanism the Store
//     interface implementation uses (e.g., in-memory, database, etc.). The request context is provided
//     to handle any necessary context operations such as deadlines or request cancellations.
//  3. If an error occurs during session generation (e.g., database error, context deadline exceeded), this
//     error is returned to the caller and no further steps are taken.
//  4. Should the session generation be successful, the new identifier is then propagated to the client using
//     the Inject method of the Propagator interface which is part of the Manager. This step typically involves
//     setting a cookie or an HTTP header in the response so that the client can include this identifier in
//     subsequent requests to maintain the session context.
//  5. The new session object is returned to the caller along with any error that might occur during the
//     identifier injection process (though no error is expected in creating a new session at this point,
//     errors might occur while setting an HTTP response header or cookie).
//
// It's important for the implementer to note that after this method is called, the client must include
// the session identifier in subsequent requests, and the server will need to handle this identifier to
// retrieve the associated session from the store.
//
// The mist.Context parameter provides request-specific information including the Request and
// ResponseWriter which are used to retrieve and set information related to the session. This context
// is assumed to be part of a custom processing pipeline that allows easy access and manipulation of
// HTTP request and response data.
//
// Usage:
// This method should be called when a new user session needs to be initiated. Typically, it would be
// invoked within the auth process, or when a session is not found for a request and needs
// to be created.
//
// Example:
//
//	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
//	    ctx := mist.NewContext(r, w)
//	    session, err := sessionManager.InitSession(ctx)
//	    if err != nil {
//	        // Handle error
//	    }
//	    // Session initialized successfully, store session data or modify response as needed.
//	})
func (m *Manager) InitSession(ctx *mist.Context) (Session, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context provided to InitSession")
	}

	// Generate a new UUID for the session.
	id := uuid.New().String()

	// Create a new session with the generated UUID.
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	sess, err := m.Generate(reqCtx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session: %w", err) // Return error if session generation fails.
	}

	// Propagate the new session identifier to the client using the ResponseWriter.
	if err = m.Inject(id, ctx.ResponseWriter); err != nil {
		// Try to clean up the session if we couldn't inject the ID
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.Store.Remove(cleanupCtx, id) // Best effort cleanup, ignore errors
		return nil, fmt.Errorf("failed to inject session ID: %w", err)
	}

	// Store the session in the context for quick access during this request
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}
	ctx.UserValues[m.CtxSessionKey] = sess

	return sess, nil
}

// SessionSecurityOptions 定义会话安全选项
type SessionSecurityOptions struct {
	// EnableSameSite 启用SameSite cookie策略
	EnableSameSite bool
	// SameSiteMode 设置SameSite模式 (None, Lax, Strict)
	SameSiteMode http.SameSite
	// SecureOnly 仅在HTTPS下使用
	SecureOnly bool
	// HttpOnly 阻止JavaScript访问cookie
	HttpOnly bool
	// EnableFingerprinting 启用会话指纹绑定
	EnableFingerprinting bool
	// RotateTokenOnValidation 每次认证成功后轮换会话令牌
	RotateTokenOnValidation bool
	// AbsoluteTimeout 会话绝对过期时间（无论活动与否）
	AbsoluteTimeout time.Duration
	// IdleTimeout 会话闲置超时时间
	IdleTimeout time.Duration
	// RenewTimeout 会话续期时间阈值
	RenewTimeout time.Duration
	// RequireReauthForSensitive 敏感操作需要重新验证
	RequireReauthForSensitive bool
	// ReauthTimeout 重新验证超时时间
	ReauthTimeout time.Duration
}

// DefaultSessionSecurityOptions 返回默认的会话安全选项
func DefaultSessionSecurityOptions() *SessionSecurityOptions {
	return &SessionSecurityOptions{
		EnableSameSite:            true,
		SameSiteMode:              http.SameSiteLaxMode,
		SecureOnly:                true,
		HttpOnly:                  true,
		EnableFingerprinting:      true,
		RotateTokenOnValidation:   true,
		AbsoluteTimeout:           24 * time.Hour,
		IdleTimeout:               30 * time.Minute,
		RenewTimeout:              5 * time.Minute,
		RequireReauthForSensitive: true,
		ReauthTimeout:             10 * time.Minute,
	}
}

// SessionFingerprint 存储会话指纹信息
type SessionFingerprint struct {
	IP         string
	UserAgent  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	LastAuthAt time.Time
}

// SetSecurityOptions 设置会话安全选项
func (m *Manager) SetSecurityOptions(options *SessionSecurityOptions) {
	// 如果是基于cookie的传播器，设置相关选项
	if cp, ok := m.Propagator.(*cookie.CookiePropagator); ok {
		if options.EnableSameSite {
			cp.SetSameSite(options.SameSiteMode)
		}
		cp.SetSecure(options.SecureOnly)
		cp.SetHTTPOnly(options.HttpOnly)
		cp.SetMaxAge(int(options.AbsoluteTimeout.Seconds()))
	}
}

// GenerateSessionFingerprint 生成会话指纹
func GenerateSessionFingerprint(r *http.Request) *SessionFingerprint {
	return &SessionFingerprint{
		IP:         getClientIP(r),
		UserAgent:  r.UserAgent(),
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		LastAuthAt: time.Now(),
	}
}

// VerifySessionFingerprint 验证会话指纹
func VerifySessionFingerprint(fp *SessionFingerprint, r *http.Request) bool {
	if fp == nil {
		return false
	}

	// 验证IP和UserAgent是否匹配
	// 注意：IP可能会变化，所以这个检查可以根据需要调整严格程度
	currentIP := getClientIP(r)
	currentUA := r.UserAgent()

	// 简单的匹配，可以根据需要增加更复杂的逻辑
	return fp.IP == currentIP && fp.UserAgent == currentUA
}

// UpdateSessionFingerprint 更新会话指纹
func UpdateSessionFingerprint(fp *SessionFingerprint, r *http.Request) {
	if fp == nil {
		return
	}
	fp.LastSeenAt = time.Now()
}

// UpdateAuthTime 更新最后认证时间
func UpdateAuthTime(fp *SessionFingerprint) {
	if fp == nil {
		return
	}
	fp.LastAuthAt = time.Now()
}

// NeedsReauthentication 检查是否需要重新认证
func NeedsReauthentication(fp *SessionFingerprint, timeout time.Duration) bool {
	if fp == nil {
		return true
	}

	return time.Now().Sub(fp.LastAuthAt) > timeout
}

// CheckSessionExpired 检查会话是否过期
func CheckSessionExpired(session Session, ctx context.Context, absoluteTimeout, idleTimeout time.Duration) bool {
	if session == nil {
		return true
	}

	// 获取会话创建和最后访问时间
	createdTimeVal, _ := session.Get(ctx, "created_at")
	lastSeenTimeVal, _ := session.Get(ctx, "last_seen_at")

	now := time.Now()

	// 检查绝对超时
	if createdTimeVal != nil {
		if created, ok := createdTimeVal.(time.Time); ok {
			if now.Sub(created) > absoluteTimeout {
				return true
			}
		}
	}

	// 检查闲置超时
	if lastSeenTimeVal != nil {
		if lastSeen, ok := lastSeenTimeVal.(time.Time); ok {
			if now.Sub(lastSeen) > idleTimeout {
				return true
			}
		}
	}

	return false
}

// getClientIP 从请求中获取客户端IP
func getClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For获取
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// 可能有多个IP，取第一个
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 尝试从X-Real-IP获取
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	return strings.Split(r.RemoteAddr, ":")[0]
}

// RefreshSessionWithSecurity 使用安全选项刷新会话
func (m *Manager) RefreshSessionWithSecurity(ctx *mist.Context, options *SessionSecurityOptions) error {
	if options == nil {
		options = DefaultSessionSecurityOptions()
	}

	session, err := m.GetSession(ctx)
	if err != nil {
		return err
	}

	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// 如果启用指纹验证，验证会话指纹
	if options.EnableFingerprinting {
		fpData, _ := session.Get(reqCtx, "fingerprint")
		if fpData != nil {
			fp, ok := fpData.(*SessionFingerprint)
			if ok {
				// 验证指纹
				if !VerifySessionFingerprint(fp, ctx.Request) {
					return fmt.Errorf("会话指纹验证失败，可能存在会话劫持")
				}
				// 更新指纹的最后访问时间
				UpdateSessionFingerprint(fp, ctx.Request)
				session.Set(reqCtx, "fingerprint", fp)
			}
		}
	}

	// 检查会话是否过期
	if CheckSessionExpired(session, reqCtx, options.AbsoluteTimeout, options.IdleTimeout) {
		return fmt.Errorf("会话已过期")
	}

	// 更新最后访问时间
	session.Set(reqCtx, "last_seen_at", time.Now())

	// 如果启用会话令牌轮换，则生成新的会话ID
	if options.RotateTokenOnValidation {
		newID := uuid.New().String()
		oldID := session.ID()

		// 创建会话副本，使用新ID
		newSession, err := m.Generate(reqCtx, newID)
		if err != nil {
			return err
		}

		// 复制会话数据
		keys := []string{"created_at", "last_seen_at", "fingerprint"}
		for _, key := range keys {
			val, err := session.Get(reqCtx, key)
			if err == nil && val != nil {
				newSession.Set(reqCtx, key, val)
			}
		}

		// 将新ID注入响应
		if err := m.Propagator.Inject(newID, ctx.ResponseWriter); err != nil {
			return err
		}

		// 保存新会话
		if err := newSession.Save(); err != nil {
			return err
		}

		// 删除旧会话
		m.Store.Remove(reqCtx, oldID)

		// 更新当前会话引用
		if ctx.UserValues == nil {
			ctx.UserValues = make(map[string]any, 1)
		}
		ctx.UserValues[m.CtxSessionKey] = newSession
	} else {
		// 保存更新后的会话
		if err := session.Save(); err != nil {
			return err
		}
	}

	return nil
}

// InitSessionWithSecurity 使用安全选项初始化会话
func (m *Manager) InitSessionWithSecurity(ctx *mist.Context, options *SessionSecurityOptions) (Session, error) {
	if options == nil {
		options = DefaultSessionSecurityOptions()
	}

	// 生成会话ID
	sessId := uuid.New().String()

	// 从请求上下文获取上下文
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// 生成新会话
	session, err := m.Store.Generate(reqCtx, sessId)
	if err != nil {
		return nil, err
	}

	// 设置会话基本信息
	session.Set(reqCtx, "created_at", time.Now())
	session.Set(reqCtx, "last_seen_at", time.Now())

	// 如果启用指纹验证，生成并保存指纹
	if options.EnableFingerprinting {
		fp := GenerateSessionFingerprint(ctx.Request)
		session.Set(reqCtx, "fingerprint", fp)
	}

	// 配置会话传播器
	if cp, ok := m.Propagator.(*cookie.CookiePropagator); ok {
		if options.EnableSameSite {
			cp.SetSameSite(options.SameSiteMode)
		}
		cp.SetSecure(options.SecureOnly)
		cp.SetHTTPOnly(options.HttpOnly)
		cp.SetMaxAge(int(options.AbsoluteTimeout.Seconds()))
	}

	// 注入会话ID到响应
	if err := m.Propagator.Inject(sessId, ctx.ResponseWriter); err != nil {
		return nil, err
	}

	// 保存会话
	if err := session.Save(); err != nil {
		return nil, err
	}

	// 缓存会话
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}
	ctx.UserValues[m.CtxSessionKey] = session

	return session, nil
}

// RequireReauthForSensitiveOperation 检查并要求重新认证敏感操作
func (m *Manager) RequireReauthForSensitiveOperation(ctx *mist.Context, options *SessionSecurityOptions) (bool, error) {
	if options == nil {
		options = DefaultSessionSecurityOptions()
	}

	session, err := m.GetSession(ctx)
	if err != nil {
		return true, err
	}

	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	// 获取会话指纹
	fpData, _ := session.Get(reqCtx, "fingerprint")
	if fpData == nil {
		return true, fmt.Errorf("会话缺少指纹信息")
	}

	fp, ok := fpData.(*SessionFingerprint)
	if !ok {
		return true, fmt.Errorf("会话指纹格式无效")
	}

	// 检查是否需要重新认证
	return NeedsReauthentication(fp, options.ReauthTimeout), nil
}

// RemoveSession is a method designed to delete a user's session from the session store
// and to clear any session identifiers from the client's context, effectively logging
// the user out. This can be a critical function for user security, ensuring sessions are
// properly terminated when a user logs out or when their session should be invalidated for
// other reasons, such as after changing a password, after a period of inactivity, or for
// administrative logout purposes.
//
// The flow of the session removal process operates as follows:
//  1. Attempt to retrieve the existing session from the mist.Context by calling the GetSession method
//     of the Manager struct, which retrieves session data based on a session identifier found in the
//     client's request.
//  2. If an error is encountered during session retrieval, such as when the session does not exist
//     or the session identifier is invalid, the error is immediately returned and the refresh
//     operation is aborted.
//  3. Once the session is successfully retrieved, the Manager struct's embedded Store interface is used
//     to remove the session data from the persistent session storage via the Store.Remove method. This
//     requires the context from the current HTTP request (for deadline or cancellation purposes) and the
//     session ID.
//  4. Any error during the session removal from the store is returned immediately, indicating a failure to
//     fully remove the session.
//  5. If the session data is successfully removed from the store, the manager proceeds to request that the session
//     identifier be removed from the client's session context by calling the Remove method of the Propagator
//     interface. This typically involves instructing the client (often via an HTTP cookie) to discard the
//     session identifier, thus preventing it from being included in future requests.
//  6. The final error from the Propagator Remove operation is returned, which may indicate if there
//     was a problem with instructing the client to discard the session (e.g., if the HTTP headers have
//     already been sent).
//
// This method completes the lifecycle management of a session, providing a clean and secure way to end a
// user's session when it is no longer needed or valid. Proper session termination is an important
// aspect of web application security, as it helps prevent unauthorized access via stale session IDs.
//
// The mist.Context refers to a custom context object which typically combines the standard
// Go Context with HTTP request and response handling, allowing easier manipulation of HTTP session management.
func (m *Manager) RemoveSession(ctx *mist.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context provided to RemoveSession")
	}

	// Retrieve the existing session.
	sess, err := m.GetSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session for removal: %w", err)
	}

	if sess == nil {
		return fmt.Errorf("session is nil after retrieval")
	}

	// Get the session's ID before removing it from the store
	id := sess.ID()
	if id == "" {
		return fmt.Errorf("session ID is empty")
	}

	// Remove the session from the store.
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	if err = m.Store.Remove(reqCtx, id); err != nil {
		return fmt.Errorf("failed to remove session from store: %w", err)
	}

	// Remove any session data from the context
	if ctx.UserValues != nil {
		delete(ctx.UserValues, m.CtxSessionKey)
	}

	// Remove the session identifier from the client (e.g., clear the cookie).
	if err = m.Propagator.Remove(ctx.ResponseWriter); err != nil {
		return fmt.Errorf("failed to remove session identifier from client: %w", err)
	}

	return nil
}

// NewManager creates and initializes a new session Manager with the specified session store and
// maximum age for sessions. This function is the primary way to create a Manager instance, ensuring
// that it is properly configured with the necessary components for session management.
//
// Parameters:
//   - store: An implementation of the Store interface that will be used to persist session data.
//     This could be an in-memory store for simple applications, a Redis store for distributed
//     applications, or any other implementation that satisfies the Store interface.
//   - maxAge: An integer representing the maximum age of a session in seconds. This value will be used
//     to configure the cookie option in the Propagator to control how long session identifiers remain
//     valid on the client side.
//
// Returns:
//   - *Manager: A pointer to the newly created and initialized Manager instance, ready to be used for
//     session management within your application.
//   - error: An error if something goes wrong during the creation process.
//
// The Manager's CtxSessionKey is set to "session" by default, which is the key used to store session
// data in the mist.Context's UserValues map. This can be changed after creation if a different key is desired.
//
// This function also initializes the Propagator component of the Manager with an implementation from the
// cookie package, which handles the transfer of session identifiers between the server and the client via
// cookies.
//
// Example usage:
//
//	// Create a new Redis-based session store
//	redisStore, err := redis.NewStore(&redis.Options{Addr: "localhost:6379"})
//	if err != nil {
//	    // Handle error
//	}
//
//	// Create a new Manager with the Redis store and a 30-minute maximum age
//	manager, err := NewManager(redisStore, 1800)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Use the manager to handle sessions in your HTTP handlers
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	    ctx := mist.NewContext(r, w)
//	    sess, err := manager.GetSession(ctx)
//	    // ...
//	})
func NewManager(store Store, maxAge int) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("session store cannot be nil")
	}

	cookieProp := cookie.InitPropagator(
		cookie.WithCookieName("mist_session"),
		cookie.WithCookieOption(func(cookie *http.Cookie) {
			cookie.MaxAge = maxAge
			cookie.Path = "/"
			cookie.HttpOnly = true
		}),
	)

	return &Manager{
		Store:         store,
		Propagator:    cookieProp,
		CtxSessionKey: "session",
		autoGC:        false,
	}, nil
}

// EnableAutoGC activates the automatic garbage collection of expired sessions.
// This method starts a background goroutine that periodically calls the GC method
// of the session Store to clean up expired sessions. This is important for managing
// server resources by preventing memory leaks from abandoned sessions.
//
// Parameters:
//   - interval: The time.Duration between garbage collection runs. This should be set
//     to a reasonable value based on the expected number of sessions and the server's
//     resource constraints. Too frequent GC can impact performance, while too infrequent
//     GC can lead to resource exhaustion.
//
// Returns:
//   - error: An error if the automatic GC could not be enabled.
//
// The garbage collection goroutine will continue running until DisableAutoGC is called
// or the Manager instance is garbage collected. The goroutine uses a context for cancellation,
// which is stored in the Manager's gcCtx field and can be canceled using the gcCancel function.
//
// This method is safe to call from multiple goroutines as it uses a mutex to protect the
// Manager's state. If automatic GC is already enabled, this method will stop the existing
// GC goroutine and start a new one with the provided interval.
//
// Example:
//
//	// Enable automatic GC every 10 minutes
//	err := manager.EnableAutoGC(10 * time.Minute)
//	if err != nil {
//	    // Handle error
//	}
func (m *Manager) EnableAutoGC(interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("GC interval must be positive")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// If GC is already running, stop it first
	if m.autoGC && m.gcCancel != nil {
		m.gcCancel()
	}

	// Create a new context with cancel for the GC goroutine
	m.gcCtx, m.gcCancel = context.WithCancel(context.Background())
	m.gcInterval = interval
	m.autoGC = true

	// Start the GC goroutine
	go m.gcWorker()

	return nil
}

// DisableAutoGC stops the automatic garbage collection of expired sessions.
// This method stops the background goroutine that was started by EnableAutoGC.
// After calling this method, sessions will no longer be automatically cleaned up,
// and the application will need to manually call the GC method of the Store to
// remove expired sessions.
//
// Returns:
//   - error: An error if automatic GC could not be disabled.
//
// This method is safe to call from multiple goroutines as it uses a mutex to protect
// the Manager's state. If automatic GC is not enabled, this method does nothing and
// returns nil.
//
// Example:
//
//	// Disable automatic GC
//	err := manager.DisableAutoGC()
//	if err != nil {
//	    // Handle error
//	}
func (m *Manager) DisableAutoGC() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.autoGC || m.gcCancel == nil {
		// GC is not running, nothing to do
		return nil
	}

	// Cancel the GC context to stop the goroutine
	m.gcCancel()
	m.autoGC = false
	m.gcCancel = nil

	return nil
}

// gcWorker is a private method that runs in a background goroutine to periodically
// garbage collect expired sessions. This method is started by EnableAutoGC and runs
// until the context is canceled by DisableAutoGC or when the Manager is garbage collected.
//
// The method uses a ticker to run the GC method of the Store at the interval specified
// in the Manager's gcInterval field. It also listens for context cancellation to stop
// the ticker and exit the goroutine.
//
// This method is intended to be called only internally by EnableAutoGC.
func (m *Manager) gcWorker() {
	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Create a context with timeout for the GC operation
			gcOpCtx, cancel := context.WithTimeout(m.gcCtx, 30*time.Second)

			// Run garbage collection
			err := m.Store.GC(gcOpCtx)
			if err != nil {
				// Log error but continue
				fmt.Printf("Session GC error: %v\n", err)
			}

			cancel() // Cancel the timeout context

		case <-m.gcCtx.Done():
			// Context was canceled, exit the goroutine
			return
		}
	}
}

// SetMaxAge adjusts the maximum age of session cookies. This method updates the CookieOption
// function within the underlying cookie propagator to set a new MaxAge value. The MaxAge value
// is specified in seconds and determines how long the session cookie will remain valid in the
// client's browser.
//
// Parameters:
//   - maxAge: The new maximum age of the session cookie in seconds. A positive value indicates
//     how many seconds the cookie will remain valid for. A negative value means the cookie will
//     be deleted when the browser session ends. A zero value indicates that the cookie should
//     be deleted immediately.
//
// This method is typically used to adjust session lifetimes based on application requirements
// or to implement features like "remember me" by extending the session duration when requested.
//
// Note that this method only affects the cookies set after this method is called. Existing
// cookies that have already been sent to clients will not be affected until they interact
// with the server again and receive a new cookie.
//
// Example:
//
//	// Set session cookies to expire after 7 days (604800 seconds)
//	manager.SetMaxAge(604800)
//
//	// Set session cookies to expire when the browser is closed
//	manager.SetMaxAge(-1)
//
//	// Set session cookies to be deleted immediately
//	manager.SetMaxAge(0)
func (m *Manager) SetMaxAge(maxAge int) {
	// Try to cast the Propagator to cookie.CookiePropagator
	if prop, ok := m.Propagator.(*cookie.CookiePropagator); ok {
		prop.SetMaxAge(maxAge)
	} else if prop, ok := m.Propagator.(*cookie.Propagator); ok {
		// Apply the cookie option to set MaxAge
		cookie.WithCookieOption(func(c *http.Cookie) {
			c.MaxAge = maxAge
		})(prop)
	}
}

// Create generates a new session with a randomly generated ID. This is a convenience method
// that can be used when you need to create a session without a specific ID or when you want
// to create a session outside of the normal HTTP request/response cycle.
//
// Returns:
//   - Session: The newly created session.
//   - error: An error if the session could not be created.
//
// This method uses the embedded Store's Generate method to create a new session with a randomly
// generated UUID. It uses the background context since there is no request context available.
//
// Example:
//
//	// Create a new session
//	sess, err := manager.Create()
//	if err != nil {
//	    // Handle error
//	}
//
//	// Use the session
//	err = sess.Set(context.Background(), "user_id", 123)
func (m *Manager) Create() (Session, error) {
	// Generate a new UUID for the session
	id := uuid.New().String()

	// Create a new context with background since there is no request context
	ctx := context.Background()

	// Use the Store's Generate method to create a new session
	return m.Generate(ctx, id)
}

// RunGC manually runs garbage collection on the session store. This method is useful
// when automatic garbage collection is disabled or when you want to immediately clean up
// expired sessions without waiting for the next automatic GC cycle.
//
// Returns:
//   - error: An error if the garbage collection failed.
//
// This method uses a background context with a 30-second timeout to ensure that the
// garbage collection operation doesn't run indefinitely. If the GC operation takes
// longer than 30 seconds, it will be canceled.
//
// Example:
//
//	// Manually run garbage collection
//	err := manager.RunGC()
//	if err != nil {
//	    // Handle error
//	}
func (m *Manager) RunGC() error {
	// Create a context with a 30-second timeout for the GC operation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run the GC operation with the timeout context
	return m.Store.GC(ctx)
}
