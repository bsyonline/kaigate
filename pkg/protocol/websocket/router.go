package websocket

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"kai/kaigate/pkg/log"
	"kai/kaigate/pkg/service/ai_agent"
	"kai/kaigate/pkg/service/mcp"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有CORS请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Connection WebSocket连接实例
type Connection struct {
	Conn      *websocket.Conn
	ID        string
	SendChan  chan []byte
	RecvChan  chan []byte
	CloseChan chan struct{}
}

// ConnectionManager WebSocket连接管理器
type ConnectionManager struct {
	connections map[string]*Connection
	mutex       sync.RWMutex
	handlers    map[string]MessageHandler
	logger      log.Logger
}

// MessageHandler 消息处理器类型
type MessageHandler func(*Connection, []byte) error

// 全局连接管理器
var connManager = &ConnectionManager{
	connections: make(map[string]*Connection),
	handlers:    make(map[string]MessageHandler),
	logger:      log.GlobalLogger, // 默认使用全局日志记录器
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	manager := &ConnectionManager{
		connections: make(map[string]*Connection),
		handlers:    make(map[string]MessageHandler),
		logger:      log.GlobalLogger, // 默认使用全局日志记录器
	}

	return manager
}

// RegisterRoutes 注册WebSocket路由
func RegisterRoutes(router *gin.Engine, logger log.Logger, agentManager ai_agent.AIAgentManager, mcpManager mcp.MCPServiceManager) {
	// 更新全局连接管理器的logger
	connManager.logger = logger

	// 启动心跳检测
	go connManager.startHeartbeat()

	// WebSocket连接端点
	router.GET("/ws/connect", createHandleWSConnect(logger))
	router.GET("/ws/ai-agent", createHandleAIAgentWS(logger, agentManager))
	router.GET("/ws/mcp", createHandleMCPWS(logger, mcpManager))

	// 注册消息处理器
	connManager.RegisterHandler("ping", handlePing)
	connManager.RegisterHandler("echo", handleEcho)
}

// createHandleWSConnect 创建基础WebSocket连接处理函数
func createHandleWSConnect(logger log.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = log.GlobalLogger
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", zap.Error(err))
			c.String(http.StatusInternalServerError, "WebSocket upgrade failed")
			return
		}

		// 创建连接ID
		connID := generateConnID()

		// 创建连接实例
		connection := &Connection{
			Conn:      conn,
			ID:        connID,
			SendChan:  make(chan []byte, 100),
			RecvChan:  make(chan []byte, 100),
			CloseChan: make(chan struct{}),
		}

		// 添加连接到管理器
		connManager.AddConnection(connection)
		logger.Info("WebSocket connection established", zap.String("conn_id", connID))

		// 启动读写协程
		go connection.readMessages(logger)
		go connection.writeMessages(logger)

		// 处理连接关闭
		<-connection.CloseChan
		connection.Close()
		connManager.RemoveConnection(connID)
		logger.Info("WebSocket connection closed", zap.String("conn_id", connID))
	}
}

// createHandleAIAgentWS 创建AI Agent WebSocket处理函数
func createHandleAIAgentWS(logger log.Logger, agentManager ai_agent.AIAgentManager) gin.HandlerFunc {
	if logger == nil {
		logger = log.GlobalLogger
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", zap.Error(err))
			c.String(http.StatusInternalServerError, "WebSocket upgrade failed")
			return
		}

		// 获取Agent ID
		agentID := c.Query("agent_id")
		if agentID == "" {
			agentID = "default"
		}

		// 创建连接ID
		connID := generateConnID()

		// 创建连接实例
		connection := &Connection{
			Conn:      conn,
			ID:        connID,
			SendChan:  make(chan []byte, 100),
			RecvChan:  make(chan []byte, 100),
			CloseChan: make(chan struct{}),
		}

		// 添加连接到管理器
		connManager.AddConnection(connection)
		logger.Info("AI Agent WebSocket connection established", zap.String("conn_id", connID), zap.String("agent_id", agentID))

		// 启动读写协程
		go connection.readMessages(logger)
		go connection.writeMessages(logger)

		// 处理连接关闭
		<-connection.CloseChan
		connection.Close()
		connManager.RemoveConnection(connID)
		logger.Info("AI Agent WebSocket connection closed", zap.String("conn_id", connID), zap.String("agent_id", agentID))
	}
}

// readMessages 从WebSocket读取消息
func (c *Connection) readMessages(logger log.Logger) {
	for {
		select {
		case <-c.CloseChan:
			return
		default:
			// 设置读取超时
			c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			// 读取消息
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				// 记录错误但不中断循环
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error("WebSocket read error", zap.String("conn_id", c.ID), zap.Error(err))
				}
				close(c.CloseChan)
				return
			}

			// 解析消息
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				logger.Error("Failed to parse WebSocket message", zap.String("conn_id", c.ID), zap.Error(err))
				continue
			}

			// 检查消息类型
			msgType, ok := msg["type"].(string)
			if !ok {
				logger.Error("Missing message type", zap.String("conn_id", c.ID))
				continue
			}

			// 调用对应的处理器
			c.RecvChan <- message
			if handler, exists := connManager.handlers[msgType]; exists {
				if err := handler(c, message); err != nil {
					logger.Error("Failed to handle WebSocket message", zap.String("conn_id", c.ID), zap.String("msg_type", msgType), zap.Error(err))
				}
			} else {
				logger.Warn("No handler found for message type", zap.String("conn_id", c.ID), zap.String("msg_type", msgType))
			}
		}
	}
}

// writeMessages 向WebSocket写入消息
func (c *Connection) writeMessages(logger log.Logger) {
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.CloseChan:
			return
		case msg := <-c.SendChan:
			// 设置写入超时
			c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Error("WebSocket write error", zap.String("conn_id", c.ID), zap.Error(err))
				close(c.CloseChan)
				return
			}
		case <-pingTicker.C:
			// 发送心跳消息
			c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("WebSocket ping error", zap.String("conn_id", c.ID), zap.Error(err))
				close(c.CloseChan)
				return
			}
		}
	}
}

// Send 发送消息到连接
func (c *Connection) Send(message []byte) {
	select {
	case c.SendChan <- message:
		// 消息成功发送到通道
	case <-c.CloseChan:
		// 连接已关闭
	default:
		// 通道已满或其他错误
		connManager.logger.Error("Failed to send message, channel closed or full", zap.String("conn_id", c.ID))
	}
}

// Close 关闭连接
func (c *Connection) Close() {
	select {
	case <-c.CloseChan:
		// 已经关闭，不需要再次关闭
	default:
		// 标记连接为关闭状态
		close(c.CloseChan)
		// 关闭WebSocket连接
		c.Conn.Close()
		// 清理通道
		close(c.SendChan)
		close(c.RecvChan)
		// 从管理器中移除连接
		connManager.RemoveConnection(c.ID)
		// 记录连接关闭
		connManager.logger.Info("WebSocket connection closed", zap.String("conn_id", c.ID))
	}
}

// AddConnection 添加连接到管理器
func (cm *ConnectionManager) AddConnection(conn *Connection) {
	cm.mutex.Lock()
	cm.connections[conn.ID] = conn
	cm.mutex.Unlock()
}

// RemoveConnection 从管理器移除连接
func (cm *ConnectionManager) RemoveConnection(connID string) {
	cm.mutex.Lock()
	delete(cm.connections, connID)
	cm.mutex.Unlock()
}

// GetConnection 获取连接
func (cm *ConnectionManager) GetConnection(connID string) (*Connection, bool) {
	cm.mutex.RLock()
	conn, exists := cm.connections[connID]
	cm.mutex.RUnlock()
	return conn, exists
}

// Broadcast 广播消息到所有连接
func (cm *ConnectionManager) Broadcast(message []byte) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for _, conn := range cm.connections {
		conn.Send(message)
	}
}

// RegisterHandler 注册消息处理器
func (cm *ConnectionManager) RegisterHandler(msgType string, handler MessageHandler) {
	cm.mutex.Lock()
	cm.handlers[msgType] = handler
	cm.mutex.Unlock()
}

// startHeartbeat 启动心跳检测
func (cm *ConnectionManager) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查所有连接状态
			cm.mutex.RLock()
			for _, conn := range cm.connections {
				// 使用非阻塞方式发送心跳消息
				select {
				case conn.SendChan <- []byte(`{"type":"pong"}`):
					// 心跳消息发送成功
				default:
					// 通道已满，关闭连接
					cm.logger.Error("Connection heartbeat failed, closing", zap.String("conn_id", conn.ID))
					conn.Close()
				}
			}
			cm.mutex.RUnlock()
		}
	}
}

// handlePing 处理Ping消息
func handlePing(conn *Connection, message []byte) error {
	// 响应Pong消息
	response := `{"type":"pong"}`
	conn.Send([]byte(response))
	return nil
}

// handleEcho 处理Echo消息
func handleEcho(conn *Connection, message []byte) error {
	// 解析消息
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return err
	}

	// 获取echo数据
	data, ok := msg["data"]
	if !ok {
		return nil
	}

	// 构造响应消息
	response := map[string]interface{}{
		"type": "echo",
		"data": data,
	}

	// 发送响应
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	conn.Send(responseBytes)
	return nil
}

// createHandleMCPWS 创建MCP WebSocket处理函数
func createHandleMCPWS(logger log.Logger, mcpManager mcp.MCPServiceManager) gin.HandlerFunc {
	if logger == nil {
		logger = log.GlobalLogger
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", zap.Error(err))
			c.String(http.StatusInternalServerError, "WebSocket upgrade failed")
			return
		}

		// 获取Service ID
		serviceID := c.Query("service_id")
		if serviceID == "" {
			serviceID = "default"
		}

		// 创建连接ID
		connID := generateConnID()

		// 创建连接实例
		connection := &Connection{
			Conn:      conn,
			ID:        connID,
			SendChan:  make(chan []byte, 100),
			RecvChan:  make(chan []byte, 100),
			CloseChan: make(chan struct{}),
		}

		// 添加连接到管理器
		connManager.AddConnection(connection)
		logger.Info("MCP WebSocket connection established", zap.String("conn_id", connID), zap.String("service_id", serviceID))

		// 启动读写协程
		go connection.readMessages(logger)
		go connection.writeMessages(logger)

		// 处理连接关闭
		<-connection.CloseChan
		connection.Close()
		connManager.RemoveConnection(connID)
		logger.Info("MCP WebSocket connection closed", zap.String("conn_id", connID), zap.String("service_id", serviceID))
	}
}

// generateConnID 生成连接ID
func generateConnID() string {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[src.Intn(len(letters))]
	}
	return string(b)
}

// randString 生成随机字符串
func randString(n int) string {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[src.Intn(len(letters))]
	}
	return string(b)
}