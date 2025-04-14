package websocket

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/dormoron/mist"
	"github.com/gorilla/websocket"
)

var (
	// ErrConnectionClosed 表示WebSocket连接已关闭
	ErrConnectionClosed = errors.New("websocket: 连接已关闭")

	// ErrMessageTooLarge 表示消息大小超过限制
	ErrMessageTooLarge = errors.New("websocket: 消息太大")

	// ErrInvalidMessageType 表示消息类型无效
	ErrInvalidMessageType = errors.New("websocket: 消息类型无效")

	// ErrChannelFull 表示发送通道已满
	ErrChannelFull = errors.New("websocket: 发送通道已满")

	// ErrChannelClosed 表示通道已关闭
	ErrChannelClosed = errors.New("websocket: 通道已关闭")
)

// MessageType 定义WebSocket消息类型
type MessageType int

const (
	// TextMessage 表示文本消息
	TextMessage = MessageType(websocket.TextMessage)

	// BinaryMessage 表示二进制消息
	BinaryMessage = MessageType(websocket.BinaryMessage)

	// CloseMessage 表示关闭连接
	CloseMessage = MessageType(websocket.CloseMessage)

	// PingMessage 表示Ping消息
	PingMessage = MessageType(websocket.PingMessage)

	// PongMessage 表示Pong消息
	PongMessage = MessageType(websocket.PongMessage)
)

// Message 表示WebSocket消息
type Message struct {
	// Type 是消息类型
	Type MessageType

	// Data 是消息数据
	Data []byte
}

// Config 表示WebSocket配置
type Config struct {
	// WriteBufferSize 是写缓冲区大小
	WriteBufferSize int

	// ReadBufferSize 是读缓冲区大小
	ReadBufferSize int

	// MaxMessageSize 是最大消息大小
	MaxMessageSize int64

	// HandshakeTimeout 是握手超时时间
	HandshakeTimeout time.Duration

	// ReadTimeout 是读取超时时间
	ReadTimeout time.Duration

	// WriteTimeout 是写入超时时间
	WriteTimeout time.Duration

	// PingInterval 是Ping间隔时间
	PingInterval time.Duration

	// MessageBufferSize 是消息缓冲区大小
	MessageBufferSize int

	// CheckOrigin 是检查Origin的函数
	CheckOrigin func(r *http.Request) bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WriteBufferSize:   4096,
		ReadBufferSize:    4096,
		MaxMessageSize:    512 * 1024, // 512KB
		HandshakeTimeout:  10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		PingInterval:      30 * time.Second,
		MessageBufferSize: 256,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}
}

// Connection 表示WebSocket连接
type Connection struct {
	// 内部WebSocket连接
	conn *websocket.Conn

	// 配置
	config *Config

	// 发送消息通道
	send chan Message

	// 接收消息通道
	receive chan Message

	// 关闭通道
	done chan struct{}

	// 关闭标志
	closed bool

	// 互斥锁
	mu sync.RWMutex

	// 用户数据
	userData map[string]interface{}

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// Upgrader 是WebSocket连接升级器
type Upgrader struct {
	// 配置
	config *Config

	// 内部升级器
	upgrader websocket.Upgrader
}

// NewUpgrader 创建一个新的WebSocket升级器
func NewUpgrader(config *Config) *Upgrader {
	if config == nil {
		config = DefaultConfig()
	}

	return &Upgrader{
		config: config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:   config.ReadBufferSize,
			WriteBufferSize:  config.WriteBufferSize,
			CheckOrigin:      config.CheckOrigin,
			HandshakeTimeout: config.HandshakeTimeout,
		},
	}
}

// Upgrade 将HTTP连接升级为WebSocket连接
func (u *Upgrader) Upgrade(ctx *mist.Context) (*Connection, error) {
	// 从mist.Context获取HTTP响应写入器和请求
	w := ctx.ResponseWriter
	r := ctx.Request

	// 执行WebSocket握手，升级连接
	conn, err := u.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// 设置最大消息大小
	conn.SetReadLimit(u.config.MaxMessageSize)

	// 设置读取超时
	if u.config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(u.config.ReadTimeout))
	}

	// 创建上下文
	wsCtx, cancel := context.WithCancel(r.Context())

	// 创建WebSocket连接
	wsConn := &Connection{
		conn:     conn,
		config:   u.config,
		send:     make(chan Message, u.config.MessageBufferSize),
		receive:  make(chan Message, u.config.MessageBufferSize),
		done:     make(chan struct{}),
		closed:   false,
		userData: make(map[string]interface{}),
		ctx:      wsCtx,
		cancel:   cancel,
	}

	// 设置Pong处理函数
	conn.SetPongHandler(func(string) error {
		if wsConn.config.ReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(wsConn.config.ReadTimeout))
		}
		return nil
	})

	// 启动读写协程
	go wsConn.readPump()
	go wsConn.writePump()

	return wsConn, nil
}

// readPump 不断从WebSocket连接读取数据
func (c *Connection) readPump() {
	defer func() {
		c.Close()
	}()

	for {
		// 设置读取截止时间
		if c.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		}

		// 读取消息
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure) {
				// 记录意外错误
			}
			break
		}

		// 发送消息到接收通道
		select {
		case c.receive <- Message{Type: MessageType(messageType), Data: data}:
		case <-c.done:
			return
		}
	}
}

// writePump 不断向WebSocket连接写入数据
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// 设置写入截止时间
			if c.config.WriteTimeout > 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			}

			if !ok {
				// 通道已关闭，发送关闭消息
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 写入消息
			err := c.conn.WriteMessage(int(message.Type), message.Data)
			if err != nil {
				return
			}

		case <-ticker.C:
			// 发送Ping消息
			if c.config.WriteTimeout > 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// Send 发送消息
func (c *Connection) Send(messageType MessageType, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrConnectionClosed
	}

	// 检查消息大小
	if int64(len(data)) > c.config.MaxMessageSize {
		return ErrMessageTooLarge
	}

	// 检查消息类型
	if messageType != TextMessage && messageType != BinaryMessage {
		return ErrInvalidMessageType
	}

	// 发送消息
	select {
	case c.send <- Message{Type: messageType, Data: data}:
		return nil
	case <-c.done:
		return ErrConnectionClosed
	default:
		return ErrChannelFull
	}
}

// SendText 发送文本消息
func (c *Connection) SendText(text string) error {
	return c.Send(TextMessage, []byte(text))
}

// SendBinary 发送二进制消息
func (c *Connection) SendBinary(data []byte) error {
	return c.Send(BinaryMessage, data)
}

// Receive 接收消息
func (c *Connection) Receive() (Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return Message{}, ErrConnectionClosed
	}

	select {
	case msg, ok := <-c.receive:
		if !ok {
			return Message{}, ErrChannelClosed
		}
		return msg, nil
	case <-c.done:
		return Message{}, ErrConnectionClosed
	}
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// 关闭通道
	close(c.done)

	// 取消上下文
	c.cancel()

	// 关闭WebSocket连接
	return c.conn.Close()
}

// Context 返回连接的上下文
func (c *Connection) Context() context.Context {
	return c.ctx
}

// SetUserValue 设置用户值
func (c *Connection) SetUserValue(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.userData[key] = value
}

// GetUserValue 获取用户值
func (c *Connection) GetUserValue(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.userData[key]
	return value, exists
}

// IsConnected 返回连接是否仍然活跃
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return !c.closed
}

// WebSocket 创建一个升级HTTP连接到WebSocket连接的处理函数
func WebSocket(config *Config, handler func(*Connection)) mist.HandleFunc {
	if config == nil {
		config = DefaultConfig()
	}

	upgrader := NewUpgrader(config)

	return func(ctx *mist.Context) {
		// 升级HTTP连接到WebSocket
		conn, err := upgrader.Upgrade(ctx)
		if err != nil {
			// 如果升级失败，返回400错误
			ctx.RespStatusCode = http.StatusBadRequest
			return
		}

		// 调用处理函数
		handler(conn)
	}
}
