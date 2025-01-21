package client

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH客户端结构
type SSHClient struct {
	Host     string
	Port     int
	Username string
	Password string
	client   *ssh.Client
}

// NewSSHClient 创建新的SSH客户端
func NewSSHClient(host string, port int, username, password string) *SSHClient {
	return &SSHClient{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

// Connect 连接到SSH服务器
func (c *SSHClient) Connect() error {
	config := &ssh.ClientConfig{
		User: c.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}

	c.client = client
	return nil
}

// CreateSession 创建新的SSH会话
func (c *SSHClient) CreateSession() (*ssh.Session, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	// 设置伪终端
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to request pty: %v", err)
	}

	return session, nil
}

// CreateInteractiveSession 创建交互式会话
func (c *SSHClient) CreateInteractiveSession() (*SSHSession, error) {
	session, err := c.CreateSession()
	if err != nil {
		return nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to start shell: %v", err)
	}

	return &SSHSession{
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
	}, nil
}

// Close 关闭SSH连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// SSHSession 交互式SSH会话
type SSHSession struct {
	Session *ssh.Session
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Stderr  io.Reader
}

// Close 关闭会话
func (s *SSHSession) Close() error {
	return s.Session.Close()
}

// ResizeTerminal 调整终端大小
func (s *SSHSession) ResizeTerminal(width, height int) error {
	return s.Session.WindowChange(height, width)
}

// Write 写入数据到终端
func (s *SSHSession) Write(data []byte) (int, error) {
	return s.Stdin.Write(data)
}
