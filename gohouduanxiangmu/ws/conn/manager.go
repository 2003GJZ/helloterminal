package conn

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Message WebSocket消息结构
type Message struct {
	Type    string      `json:"type"`    // 消息类型：data, resize, error
	Payload interface{} `json:"payload"` // 消息内容
}

// Connection WebSocket连接结构
type Connection struct {
	ID     string
	Socket *websocket.Conn
	Send   chan []byte
	mu     sync.Mutex
}

// Manager WebSocket连接管理器
type Manager struct {
	connections  map[string]*Connection
	mu           sync.RWMutex
	onConnect    func(*Connection)
	onDisconnect func(*Connection)
	onMessage    func(*Connection, []byte)
}

// NewManager 创建新的连接管理器
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
	}
}

// SetHandlers 设置事件处理器
func (m *Manager) SetHandlers(onConnect, onDisconnect func(*Connection), onMessage func(*Connection, []byte)) {
	m.onConnect = onConnect
	m.onDisconnect = onDisconnect
	m.onMessage = onMessage
}

// AddConnection 添加新连接
func (m *Manager) AddConnection(conn *Connection) {
	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.mu.Unlock()

	if m.onConnect != nil {
		m.onConnect(conn)
	}

	// 启动消息处理
	go m.handleMessages(conn)
	go m.handleWrites(conn)
}

// RemoveConnection 移除连接
func (m *Manager) RemoveConnection(conn *Connection) {
	m.mu.Lock()
	if _, ok := m.connections[conn.ID]; ok {
		delete(m.connections, conn.ID)
		conn.mu.Lock()
		if conn.Send != nil {
			close(conn.Send)
			conn.Send = nil
		}
		conn.mu.Unlock()
	}
	m.mu.Unlock()

	if m.onDisconnect != nil {
		m.onDisconnect(conn)
	}
}

// GetConnection 获取连接
func (m *Manager) GetConnection(id string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[id]
}

// handleMessages 处理接收到的消息
func (m *Manager) handleMessages(conn *Connection) {
	defer func() {
		m.RemoveConnection(conn)
		conn.Socket.Close()
	}()

	for {
		_, message, err := conn.Socket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		if m.onMessage != nil {
			m.onMessage(conn, message)
		}
	}
}

// handleWrites 处理发送消息
func (m *Manager) handleWrites(conn *Connection) {
	defer func() {
		conn.Socket.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			if !ok {
				conn.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.mu.Lock()
			err := conn.Socket.WriteMessage(websocket.TextMessage, message)
			conn.mu.Unlock()

			if err != nil {
				log.Printf("error writing message: %v", err)
				return
			}
		}
	}
}

// SendMessage 发送消息到指定连接
func (conn *Connection) SendMessage(msgType string, payload interface{}) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.Send == nil {
		return fmt.Errorf("connection closed")
	}

	msg := Message{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case conn.Send <- data:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Broadcast 广播消息给所有连接
func (m *Manager) Broadcast(msgType string, payload interface{}) {
	msg := Message{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling broadcast message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, conn := range m.connections {
		select {
		case conn.Send <- data:
		default:
			continue
		}
	}
}
