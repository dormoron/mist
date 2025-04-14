package websocket

import (
	"sync"
)

// Hub 管理所有活跃的WebSocket连接和消息广播
type Hub struct {
	// 所有活跃连接，按ID索引
	connections map[string]*Connection

	// 房间映射，房间名到连接集合
	rooms map[string]map[string]*Connection

	// 互斥锁保护共享资源
	mu sync.RWMutex
}

// NewHub 创建新的Hub实例
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]*Connection),
		rooms:       make(map[string]map[string]*Connection),
	}
}

// Register 注册一个新的WebSocket连接
func (h *Hub) Register(connID string, conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connections[connID] = conn

	// 当连接关闭时自动注销
	go func() {
		<-conn.Context().Done()
		h.Unregister(connID)
	}()
}

// Unregister 注销一个WebSocket连接
func (h *Hub) Unregister(connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从所有房间中移除连接
	for roomName, connections := range h.rooms {
		if _, exists := connections[connID]; exists {
			delete(h.rooms[roomName], connID)

			// 如果房间为空，删除房间
			if len(h.rooms[roomName]) == 0 {
				delete(h.rooms, roomName)
			}
		}
	}

	// 从连接列表中移除
	if _, exists := h.connections[connID]; exists {
		delete(h.connections, connID)
	}
}

// JoinRoom 让连接加入一个房间
func (h *Hub) JoinRoom(roomName string, connID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查连接是否存在
	conn, exists := h.connections[connID]
	if !exists {
		return false
	}

	// 创建房间（如果不存在）
	if _, exists := h.rooms[roomName]; !exists {
		h.rooms[roomName] = make(map[string]*Connection)
	}

	// 将连接添加到房间
	h.rooms[roomName][connID] = conn
	return true
}

// LeaveRoom 让连接离开一个房间
func (h *Hub) LeaveRoom(roomName string, connID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查房间是否存在
	room, exists := h.rooms[roomName]
	if !exists {
		return false
	}

	// 从房间中移除连接
	if _, exists := room[connID]; exists {
		delete(h.rooms[roomName], connID)

		// 如果房间为空，删除房间
		if len(h.rooms[roomName]) == 0 {
			delete(h.rooms, roomName)
		}

		return true
	}

	return false
}

// GetConnection 获取指定ID的连接
func (h *Hub) GetConnection(connID string) (*Connection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, exists := h.connections[connID]
	return conn, exists
}

// BroadcastToAll 向所有连接广播消息
func (h *Hub) BroadcastToAll(msgType MessageType, data []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, conn := range h.connections {
		if conn.IsConnected() {
			_ = conn.Send(msgType, data)
			count++
		}
	}

	return count
}

// BroadcastToRoom 向房间内的所有连接广播消息
func (h *Hub) BroadcastToRoom(roomName string, msgType MessageType, data []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomName]
	if !exists {
		return 0
	}

	count := 0
	for _, conn := range room {
		if conn.IsConnected() {
			_ = conn.Send(msgType, data)
			count++
		}
	}

	return count
}

// BroadcastText 向所有连接广播文本消息
func (h *Hub) BroadcastText(text string) int {
	return h.BroadcastToAll(TextMessage, []byte(text))
}

// BroadcastTextToRoom 向房间内的所有连接广播文本消息
func (h *Hub) BroadcastTextToRoom(roomName string, text string) int {
	return h.BroadcastToRoom(roomName, TextMessage, []byte(text))
}

// BroadcastBinary 向所有连接广播二进制消息
func (h *Hub) BroadcastBinary(data []byte) int {
	return h.BroadcastToAll(BinaryMessage, data)
}

// BroadcastBinaryToRoom 向房间内的所有连接广播二进制消息
func (h *Hub) BroadcastBinaryToRoom(roomName string, data []byte) int {
	return h.BroadcastToRoom(roomName, BinaryMessage, data)
}

// CountConnections 计算当前活跃连接数
func (h *Hub) CountConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.connections)
}

// CountRoomConnections 计算房间内活跃连接数
func (h *Hub) CountRoomConnections(roomName string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomName]
	if !exists {
		return 0
	}

	return len(room)
}

// GetRooms 获取所有房间名称
func (h *Hub) GetRooms() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rooms := make([]string, 0, len(h.rooms))
	for room := range h.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}

// GetRoomConnections 获取房间内所有连接ID
func (h *Hub) GetRoomConnections(roomName string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomName]
	if !exists {
		return []string{}
	}

	connections := make([]string, 0, len(room))
	for connID := range room {
		connections = append(connections, connID)
	}

	return connections
}
