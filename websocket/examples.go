package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/dormoron/mist"
	"github.com/google/uuid"
)

// ChatMessage 表示聊天消息结构
type ChatMessage struct {
	// 消息类型
	Type string `json:"type"`

	// 发送者ID
	Sender string `json:"sender"`

	// 目标房间
	Room string `json:"room,omitempty"`

	// 消息内容
	Content string `json:"content"`

	// 发送时间
	Timestamp int64 `json:"timestamp"`
}

// 示例1：创建简单的Echo服务器
func ExampleEchoServer() mist.HandleFunc {
	// 使用默认配置
	return WebSocket(nil, func(conn *Connection) {
		// 持续接收消息
		for {
			msg, err := conn.Receive()
			if err != nil {
				// 连接已关闭或出错
				return
			}

			// 简单地将消息回显给发送者
			if err := conn.Send(msg.Type, msg.Data); err != nil {
				// 发送失败
				return
			}
		}
	})
}

// 示例2：创建聊天室服务器
func ExampleChatServer() *mist.HTTPServer {
	server := mist.InitHTTPServer()

	// 创建一个WebSocket Hub
	hub := NewHub()

	// 处理WebSocket连接
	server.GET("/ws/chat", WebSocket(nil, func(conn *Connection) {
		// 为连接生成唯一ID
		connID := uuid.New().String()

		// 注册连接
		hub.Register(connID, conn)

		// 设置用户数据
		conn.SetUserValue("id", connID)
		conn.SetUserValue("joinTime", time.Now())

		// 通知其他用户有新连接
		notifyMsg := ChatMessage{
			Type:      "system",
			Content:   "新用户加入",
			Timestamp: time.Now().Unix(),
		}
		notifyData, _ := json.Marshal(notifyMsg)
		hub.BroadcastToAll(TextMessage, notifyData)

		// 持续接收消息
		for {
			msg, err := conn.Receive()
			if err != nil {
				// 连接已关闭
				break
			}

			// 只处理文本消息
			if msg.Type != TextMessage {
				continue
			}

			// 解析消息
			var chatMsg ChatMessage
			if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
				// 无效的消息格式
				continue
			}

			// 设置发送者ID和时间戳
			chatMsg.Sender = connID
			chatMsg.Timestamp = time.Now().Unix()

			// 重新编码消息
			data, _ := json.Marshal(chatMsg)

			// 根据消息类型处理
			switch chatMsg.Type {
			case "chat":
				// 普通聊天消息，广播给所有人
				hub.BroadcastToAll(TextMessage, data)

			case "join":
				// 加入房间
				if chatMsg.Room != "" {
					hub.JoinRoom(chatMsg.Room, connID)

					// 通知房间内其他人
					notification := ChatMessage{
						Type:      "system",
						Room:      chatMsg.Room,
						Content:   "用户 " + connID + " 加入房间",
						Timestamp: time.Now().Unix(),
					}
					notifyData, _ := json.Marshal(notification)
					hub.BroadcastToRoom(chatMsg.Room, TextMessage, notifyData)
				}

			case "leave":
				// 离开房间
				if chatMsg.Room != "" {
					hub.LeaveRoom(chatMsg.Room, connID)

					// 通知房间内其他人
					notification := ChatMessage{
						Type:      "system",
						Room:      chatMsg.Room,
						Content:   "用户 " + connID + " 离开房间",
						Timestamp: time.Now().Unix(),
					}
					notifyData, _ := json.Marshal(notification)
					hub.BroadcastToRoom(chatMsg.Room, TextMessage, notifyData)
				}

			case "room":
				// 房间消息，只广播给房间内的人
				if chatMsg.Room != "" {
					hub.BroadcastToRoom(chatMsg.Room, TextMessage, data)
				}
			}
		}

		// 连接关闭后，通知其他用户
		exitMsg := ChatMessage{
			Type:      "system",
			Content:   "用户 " + connID + " 离开",
			Timestamp: time.Now().Unix(),
		}
		exitData, _ := json.Marshal(exitMsg)
		hub.BroadcastToAll(TextMessage, exitData)

		// Hub会自动注销连接
	}))

	// 添加状态端点，查看WebSocket连接状态
	server.GET("/ws/status", func(ctx *mist.Context) {
		status := map[string]interface{}{
			"connections": hub.CountConnections(),
			"rooms":       hub.GetRooms(),
		}

		// 如果有room参数，显示房间内的连接
		if roomName := ctx.QueryValue("room").StringOrDefault(""); roomName != "" {
			status["room"] = roomName
			status["room_connections"] = hub.CountRoomConnections(roomName)
			status["connections_list"] = hub.GetRoomConnections(roomName)
		}

		// 返回JSON响应
		data, _ := json.Marshal(status)
		ctx.RespStatusCode = 200
		ctx.RespData = data
	})

	return server
}

// ExampleNotificationService 是一个实时通知服务示例
func ExampleNotificationService() *mist.HTTPServer {
	server := mist.InitHTTPServer()

	// 创建WebSocket Hub
	hub := NewHub()

	// 处理WebSocket连接
	server.GET("/ws/notifications/:userID", func(ctx *mist.Context) {
		// 获取用户ID
		userID := ctx.PathParams["userID"]

		// 验证用户身份（在实际应用中应该进行认证）
		if userID == "" {
			ctx.RespStatusCode = 400
			ctx.RespData = []byte("Missing userID")
			return
		}

		// 创建自定义的WebSocket配置
		config := DefaultConfig()
		config.PingInterval = 20 * time.Second // 更频繁的ping

		// 创建WebSocket连接处理函数
		wsHandler := WebSocket(config, func(conn *Connection) {
			// 注册连接
			hub.Register(userID, conn)

			// 将用户加入到个人通知通道
			hub.JoinRoom("user:"+userID, userID)

			// 发送欢迎消息
			welcomeMsg := map[string]interface{}{
				"type":    "welcome",
				"message": "Connected to notification service",
				"time":    time.Now().Format(time.RFC3339),
			}

			data, _ := json.Marshal(welcomeMsg)
			conn.SendText(string(data))

			// 循环处理消息
			for {
				_, err := conn.Receive()
				if err != nil {
					// 连接关闭
					break
				}
				// 对于通知服务，我们主要是服务器向客户端推送消息
				// 所以这里不需要处理客户端发送的消息
			}

			// 连接关闭后，自动注销
			log.Printf("User %s disconnected from notification service", userID)
		})

		// 执行处理函数
		wsHandler(ctx)
	})

	// 提供一个发送通知的API端点
	server.POST("/api/notify", func(ctx *mist.Context) {
		// 解析通知请求
		var req struct {
			UserID  string `json:"user_id"`
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if err := ctx.RespondSuccess(&req); err != nil {
			ctx.RespStatusCode = 400
			ctx.RespData = []byte("Invalid request format")
			return
		}

		// 构造通知消息
		notification := map[string]interface{}{
			"type":    req.Type,
			"message": req.Message,
			"time":    time.Now().Format(time.RFC3339),
		}

		data, _ := json.Marshal(notification)

		// 发送通知
		count := 0
		if req.UserID != "" {
			// 发送给特定用户
			count = hub.BroadcastTextToRoom("user:"+req.UserID, string(data))
		} else {
			// 发送给所有用户
			count = hub.BroadcastText(string(data))
		}

		// 返回结果
		result := map[string]interface{}{
			"success": true,
			"sent_to": count,
		}

		resultData, _ := json.Marshal(result)
		ctx.RespStatusCode = 200
		ctx.RespData = resultData
	})

	return server
}

// 示例HTML客户端代码:
/*
<!DOCTYPE html>
<html>
<head>
    <title>Mist WebSocket Chat</title>
    <style>
        body { margin: 0; padding-bottom: 3rem; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
        #messages { list-style-type: none; margin: 0; padding: 0; }
        #messages > li { padding: 0.5rem 1rem; }
        #messages > li:nth-child(odd) { background: #f8f8f8; }
        #form { background: rgba(0, 0, 0, 0.15); padding: 0.25rem; position: fixed; bottom: 0; left: 0; right: 0; display: flex; height: 3rem; box-sizing: border-box; backdrop-filter: blur(10px); }
        #input { border: none; padding: 0 1rem; flex-grow: 1; border-radius: 2rem; margin: 0.25rem; }
        #input:focus { outline: none; }
        #form > button { background: #333; border: none; padding: 0 1rem; margin: 0.25rem; border-radius: 3px; outline: none; color: #fff; }
        #room-form { padding: 1rem; display: flex; }
        #room-input { padding: 0.5rem; flex-grow: 1; }
    </style>
</head>
<body>
    <div id="room-form">
        <input id="room-input" placeholder="输入房间名..." />
        <button id="join-btn">加入房间</button>
        <button id="leave-btn">离开房间</button>
    </div>

    <div>
        <span>当前房间: </span>
        <span id="current-room">全局</span>
    </div>

    <ul id="messages"></ul>

    <form id="form" action="">
        <input id="input" autocomplete="off" placeholder="输入消息..." />
        <button>发送</button>
    </form>

    <script>
        // 连接WebSocket
        const socket = new WebSocket(`ws://${window.location.host}/ws/chat`);
        let currentRoom = '';

        // 显示消息
        function addMessage(message) {
            const item = document.createElement('li');

            // 格式化时间
            const date = new Date(message.timestamp * 1000);
            const time = date.toLocaleTimeString();

            // 根据消息类型格式化
            if (message.type === 'system') {
                item.textContent = `[${time}] 系统: ${message.content}`;
                item.style.color = '#888';
            } else {
                const roomInfo = message.room ? `[${message.room}] ` : '';
                item.textContent = `[${time}] ${roomInfo}${message.sender}: ${message.content}`;
            }

            document.getElementById('messages').appendChild(item);
            window.scrollTo(0, document.body.scrollHeight);
        }

        // 连接打开时
        socket.addEventListener('open', (event) => {
            addMessage({
                type: 'system',
                content: '已连接到服务器',
                timestamp: Math.floor(Date.now() / 1000)
            });
        });

        // 接收消息
        socket.addEventListener('message', (event) => {
            const message = JSON.parse(event.data);
            addMessage(message);
        });

        // 连接关闭时
        socket.addEventListener('close', (event) => {
            addMessage({
                type: 'system',
                content: '已断开连接',
                timestamp: Math.floor(Date.now() / 1000)
            });
        });

        // 发送消息
        document.getElementById('form').addEventListener('submit', (e) => {
            e.preventDefault();

            const input = document.getElementById('input');
            const message = input.value.trim();

            if (message) {
                // 创建消息对象
                const msgObj = {
                    type: currentRoom ? 'room' : 'chat',
                    content: message
                };

                // 如果在房间中，添加房间信息
                if (currentRoom) {
                    msgObj.room = currentRoom;
                }

                // 发送消息
                socket.send(JSON.stringify(msgObj));

                // 清空输入框
                input.value = '';
            }
        });

        // 加入房间
        document.getElementById('join-btn').addEventListener('click', () => {
            const roomInput = document.getElementById('room-input');
            const room = roomInput.value.trim();

            if (room) {
                // 发送加入房间消息
                socket.send(JSON.stringify({
                    type: 'join',
                    room: room
                }));

                currentRoom = room;
                document.getElementById('current-room').textContent = room;
            }
        });

        // 离开房间
        document.getElementById('leave-btn').addEventListener('click', () => {
            if (currentRoom) {
                // 发送离开房间消息
                socket.send(JSON.stringify({
                    type: 'leave',
                    room: currentRoom
                }));

                currentRoom = '';
                document.getElementById('current-room').textContent = '全局';
            }
        });
    </script>
</body>
</html>
*/
