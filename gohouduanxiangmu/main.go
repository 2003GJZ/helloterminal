package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"webterminal/config"
	"webterminal/ssh/session"
	"webterminal/ws/conn"
)

var (
	wsManager      *conn.Manager
	sessionManager *session.Manager
	upgrader       = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 开发环境允许所有来源
		},
	}
)

func init() {
	// 初始化配置
	config.LoadConfig()

	// 初始化管理器
	wsManager = conn.NewManager()
	sessionManager = session.NewManager()

	// 设置WebSocket事件处理器
	wsManager.SetHandlers(
		func(c *conn.Connection) {
			log.Printf("New connection: %s", c.ID)
		},
		func(c *conn.Connection) {
			log.Printf("Connection closed: %s", c.ID)
			if session := sessionManager.GetSession(c.ID); session != nil {
				sessionManager.CloseSession(c.ID)
			}
		},
		func(c *conn.Connection, message []byte) {
			if err := sessionManager.HandleWSMessage(c.ID, message); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		},
	)
}

func main() {
	r := gin.Default()

	// 允许跨域
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API 路由组
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		// 获取可用的服务器列表
		api.GET("/servers", func(c *gin.Context) {
			// TODO: 从数据库或配置文件获取服务器列表
			servers := []gin.H{
				{
					"id":       "1",
					"name":     "本地主机",
					"host":     "localhost",
					"port":     22,
					"username": "root",
				},
			}
			c.JSON(200, servers)
		})

		// 创建新的终端会话
		api.POST("/terminal", func(c *gin.Context) {
			var params struct {
				Host     string `json:"host" binding:"required"`
				Port     int    `json:"port" binding:"required"`
				Username string `json:"username" binding:"required"`
				Password string `json:"password" binding:"required"`
			}

			if err := c.ShouldBindJSON(&params); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			sessionID := uuid.New().String()
			c.JSON(200, gin.H{
				"session_id": sessionID,
			})
		})
	}

	// WebSocket 终端处理
	r.GET("/ws/terminal/:id", func(c *gin.Context) {
		sessionID := c.Param("id")
		wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}

		// 创建WebSocket连接
		conn := &conn.Connection{
			ID:     sessionID,
			Socket: wsConn,
			Send:   make(chan []byte, 256),
		}

		// 添加到连接管理器
		wsManager.AddConnection(conn)

		// 获取会话参数
		host := c.Query("host")
		port := 22 // 默认SSH端口
		username := c.Query("username")
		password := c.Query("password")

		// 创建SSH会话
		if err := sessionManager.CreateSession(sessionID, host, port, username, password, conn); err != nil {
			conn.SendMessage("error", fmt.Sprintf("Failed to create session: %v", err))
			wsManager.RemoveConnection(conn)
			return
		}
	})

	// 启动服务器
	cfg := config.GetConfig()
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s...", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
