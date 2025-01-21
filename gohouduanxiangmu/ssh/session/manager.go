package session

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"webterminal/ssh/client"
	"webterminal/ws/conn"
)

// TerminalSession 终端会话
type TerminalSession struct {
	ID        string
	SSHClient *client.SSHClient
	Session   *client.SSHSession
	WSConn    *conn.Connection
	done      chan struct{}
}

// Manager 会话管理器
type Manager struct {
	sessions map[string]*TerminalSession
	mu       sync.RWMutex
}

// NewManager 创建新的会话管理器
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*TerminalSession),
	}
}

// CreateSession 创建新的终端会话
func (m *Manager) CreateSession(id string, host string, port int, username, password string, wsConn *conn.Connection) error {
	sshClient := client.NewSSHClient(host, port, username, password)
	if err := sshClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	session, err := sshClient.CreateInteractiveSession()
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("failed to create SSH session: %v", err)
	}

	terminalSession := &TerminalSession{
		ID:        id,
		SSHClient: sshClient,
		Session:   session,
		WSConn:    wsConn,
		done:      make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[id] = terminalSession
	m.mu.Unlock()

	// 启动数据传输
	go m.handleSSHOutput(terminalSession)

	return nil
}

// GetSession 获取会话
func (m *Manager) GetSession(id string) *TerminalSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// CloseSession 关闭会话
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found")
	}
	delete(m.sessions, id)
	m.mu.Unlock()

	close(session.done)
	session.Session.Close()
	session.SSHClient.Close()

	return nil
}

// handleSSHOutput 处理SSH输出
func (m *Manager) handleSSHOutput(session *TerminalSession) {
	defer func() {
		m.CloseSession(session.ID)
	}()

	// 创建输出缓冲区
	buf := make([]byte, 8192)
	for {
		select {
		case <-session.done:
			return
		default:
			n, err := session.Session.Stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from SSH session: %v", err)
				}
				return
			}

			if n > 0 {
				data := buf[:n]
				err = session.WSConn.SendMessage("data", string(data))
				if err != nil {
					log.Printf("Error sending message to WebSocket: %v", err)
					// 如果是连接关闭错误，直接返回
					if err.Error() == "connection closed" {
						return
					}
				}
			}
		}
	}
}

// HandleWSMessage 处理WebSocket消息
func (m *Manager) HandleWSMessage(sessionID string, message []byte) error {
	session := m.GetSession(sessionID)
	if session == nil {
		return fmt.Errorf("session not found")
	}

	var msg conn.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %v", err)
	}

	switch msg.Type {
	case "data":
		// 处理终端输入
		if data, ok := msg.Payload.(string); ok {
			_, err := session.Session.Write([]byte(data))
			if err != nil {
				return fmt.Errorf("failed to write to SSH session: %v", err)
			}
		}

	case "resize":
		// 处理终端大小调整
		if size, ok := msg.Payload.(map[string]interface{}); ok {
			width := int(size["width"].(float64))
			height := int(size["height"].(float64))
			err := session.Session.ResizeTerminal(width, height)
			if err != nil {
				return fmt.Errorf("failed to resize terminal: %v", err)
			}
		}
	}

	return nil
}
