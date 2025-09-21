package bootstrap

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kai/kaigate/pkg/config"
	"kai/kaigate/pkg/log"
	http_protocol "kai/kaigate/pkg/protocol/http"
	"kai/kaigate/pkg/protocol/websocket"
	"kai/kaigate/pkg/service/ai_agent"
	"kai/kaigate/pkg/service/mcp"
)

// Server 服务器实例
type Server struct {
	httpServer    *http.Server
	wsServer      *http.Server
	adminServer   *http.Server
	httpRouter    *gin.Engine
	wsRouter      *gin.Engine
	adminRouter   *gin.Engine
	serverContext context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
	logger        log.Logger
	agentManager  ai_agent.AIAgentManager
	mcpManager    mcp.MCPServiceManager
	// 用于存储已注册的代理路由，便于更新
	registeredProxyRoutes map[string]bool
}

// ServerOption 服务器选项
type ServerOption func(*Server)

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithAIAgentManager 设置AI Agent管理器
func WithAIAgentManager(manager ai_agent.AIAgentManager) ServerOption {
	return func(s *Server) {
		s.agentManager = manager
	}
}

// WithMCPServiceManager 设置MCP服务管理器
func WithMCPServiceManager(manager mcp.MCPServiceManager) ServerOption {
	return func(s *Server) {
		s.mcpManager = manager
	}
}

// NewServer 创建新的服务器实例
func NewServer(options ...ServerOption) *Server {
	// 创建服务器上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建服务器实例
	server := &Server{
		serverContext:         ctx,
		cancelFunc:            cancel,
		logger:                log.GlobalLogger, // 使用默认日志记录器
		registeredProxyRoutes: make(map[string]bool),
	}

	// 应用选项
	for _, option := range options {
		option(server)
	}

	// 初始化HTTP路由
	server.httpRouter = gin.New()
	server.httpRouter.Use(gin.Recovery())

	// 初始化WebSocket路由
	server.wsRouter = gin.New()
	server.wsRouter.Use(gin.Recovery())

	// 初始化管理接口路由
	server.adminRouter = gin.New()
	server.adminRouter.Use(gin.Recovery())

	// 创建HTTP服务器
	server.httpServer = &http.Server{
		Addr:    config.GlobalConfig.Server.HTTPAddr,
		Handler: server.httpRouter,
	}

	// 创建WebSocket服务器
	server.wsServer = &http.Server{
		Addr:    config.GlobalConfig.Server.WSAddr,
		Handler: server.wsRouter,
	}

	// 创建管理接口服务器
	server.adminServer = &http.Server{
		Addr:    config.GlobalConfig.Server.AdminAddr,
		Handler: server.adminRouter,
	}

	// 定义一个回调函数来记录注册的路由
	onRouteRegistered := func(path string) {
		server.registeredProxyRoutes[path] = true
	}

	// 注册HTTP处理器，传入管理器和路由注册回调
	http_protocol.RegisterRoutes(server.httpRouter, server.logger, server.agentManager, server.mcpManager, onRouteRegistered)

	// 注册WebSocket处理器，传入管理器
	websocket.RegisterRoutes(server.wsRouter, server.logger, server.agentManager, server.mcpManager)

	// 注册管理接口处理器
	server.registerAdminRoutes(server.adminRouter)

	return server
}

// ReloadProxyRoutes 重新加载代理路由配置
func (s *Server) ReloadProxyRoutes() error {
	// 重新加载配置
	if err := config.ReloadConfig(); err != nil {
		s.logger.Error("Failed to reload config", zap.Error(err))
		return err
	}

	// 创建一个新的路由组来处理代理路由
	// 注意：Gin不支持直接删除路由，我们通过重新注册同名路由来覆盖旧的处理函数
	s.logger.Info("Reloading proxy routes...")

	// 清除已注册的代理路由记录
	clear(s.registeredProxyRoutes)

	// 定义一个回调函数来记录新注册的路由
	onRouteRegistered := func(path string) {
		s.registeredProxyRoutes[path] = true
	}

	// 重新注册代理路由
	http_protocol.RegisterProxyRoutesFromConfig(s.httpRouter, s.logger, onRouteRegistered)
	s.logger.Info("Proxy routes reloaded successfully")

	return nil
}

// handleReloadConfig 处理配置重载请求
func (s *Server) handleReloadConfig(c *gin.Context) {
	if err := config.ReloadConfig(); err != nil {
		s.logger.Error("Failed to reload config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload config: " + err.Error(),
		})
		return
	}

	s.logger.Info("Config reloaded successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Config reloaded successfully",
	})
}

// handleReloadProxyRoutes 处理代理路由重载请求
func (s *Server) handleReloadProxyRoutes(c *gin.Context) {
	if err := s.ReloadProxyRoutes(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload proxy routes: " + err.Error(),
		})
		return
	}

	// 获取当前已注册的代理路由信息
	proxyRoutesInfo := []string{}
	for route := range s.registeredProxyRoutes {
		proxyRoutesInfo = append(proxyRoutesInfo, route)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Proxy routes reloaded successfully",
		"proxy_routes": proxyRoutesInfo,
	})
}

// registerAdminRoutes 注册管理接口路由
func (s *Server) registerAdminRoutes(router *gin.Engine) {
	// 健康检查接口
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 状态信息接口
	router.GET("/status", func(c *gin.Context) {
		response := gin.H{
			"service":   config.ServiceName,
			"version":   config.ServiceVersion,
			"status":    "running",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// 添加已注册的AI Agent信息
		if s.agentManager != nil {
			response["ai_agents"] = s.agentManager.ListAvailableAgents()
		}

		// 添加已注册的MCP服务信息
		if s.mcpManager != nil {
			response["mcp_services"] = s.mcpManager.ListAvailableServices()
		}

		// 添加已注册的代理路由信息
		proxyRoutesInfo := []string{}
		for route := range s.registeredProxyRoutes {
			proxyRoutesInfo = append(proxyRoutesInfo, route)
		}
		response["proxy_routes"] = proxyRoutesInfo

		c.JSON(http.StatusOK, response)
	})

	// 配置信息接口（仅调试模式可见）
	if config.GlobalConfig.Server.Debug {
		router.GET("/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, config.GlobalConfig)
		})
	}

	// 配置重载接口
	router.POST("/reload-config", s.handleReloadConfig)

	// 代理路由重载接口
	router.POST("/reload-proxy-routes", s.handleReloadProxyRoutes)
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动HTTP服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("Starting HTTP server", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// 启动WebSocket服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("Starting WebSocket server", zap.String("addr", s.wsServer.Addr))
		if err := s.wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server error", zap.Error(err))
		}
	}()

	// 启动管理接口服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("Starting admin server", zap.String("addr", s.adminServer.Addr))
		if err := s.adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Admin server error", zap.Error(err))
		}
	}()

	// 监听系统信号
	s.handleSignals()

	return nil
}

// handleSignals 处理系统信号
func (s *Server) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 等待终止信号
	go func() {
		sig := <-sigCh
		s.logger.Info("Received signal, shutting down...", zap.String("signal", sig.String()))
		s.Stop()
	}()
}

// Stop 停止服务器
func (s *Server) Stop() {
	// 取消服务器上下文
	s.cancelFunc()

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.GlobalConfig.Server.ConnTimeout)*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("HTTP server shutdown error", zap.Error(err))
		} else {
			s.logger.Info("HTTP server stopped gracefully")
		}
	}

	// 关闭WebSocket服务器
	if s.wsServer != nil {
		if err := s.wsServer.Shutdown(ctx); err != nil {
			s.logger.Error("WebSocket server shutdown error", zap.Error(err))
		} else {
			s.logger.Info("WebSocket server stopped gracefully")
		}
	}

	// 关闭管理接口服务器
	if s.adminServer != nil {
		if err := s.adminServer.Shutdown(ctx); err != nil {
			s.logger.Error("Admin server shutdown error", zap.Error(err))
		} else {
			s.logger.Info("Admin server stopped gracefully")
		}
	}

	// 等待所有goroutine完成
	s.wg.Wait()

	// 记录关闭信息
	s.logger.Info("Kaigate server exited gracefully")
}