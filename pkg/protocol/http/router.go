package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kai/kaigate/pkg/config"
	"kai/kaigate/pkg/log"
	"kai/kaigate/pkg/service/ai_agent"
	"kai/kaigate/pkg/service/mcp"
)

// RegisterRoutes 注册HTTP路由
func RegisterRoutes(router *gin.Engine, logger log.Logger, agentManager ai_agent.AIAgentManager, mcpManager mcp.MCPServiceManager, onRouteRegistered func(string)) {
	// 添加全局中间件
	router.Use(loggerMiddleware(logger))
	router.Use(recoveryMiddleware())
	router.Use(corsMiddleware())

	// 从配置中动态注册代理路由
	registerProxyRoutesFromConfig(router, logger, onRouteRegistered)

	// API路由组
	api := router.Group("/api/v1")
	{
		// 通用接口示例
		api.GET("/health", handleHealthCheck)
		api.GET("/version", handleVersion)

		// AI Agent接口
		aia := api.Group("/ai-agent")
		{
			aia.POST("/chat", createHandleAIChat(agentManager))
			aia.POST("/completion", createHandleAICompletion(agentManager))
			aia.POST("/embedding", createHandleAIEmbedding(agentManager))
			aia.GET("/models", createHandleListModels(agentManager))
		}

		// MCP服务接口
		mcp := api.Group("/mcp")
		{
			mcp.POST("/command", createHandleMCPCommand(mcpManager))
			mcp.GET("/services", createHandleListMCPServices(mcpManager))
		}
	}

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})
}

// loggerMiddleware 日志中间件
func loggerMiddleware(logger log.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = log.GlobalLogger
	}
	return func(c *gin.Context) {
		// 记录请求开始时间
		tstart := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		remoteAddr := c.ClientIP()

		// 处理请求
		c.Next()

		// 计算请求耗时
		latency := time.Since(tstart).Milliseconds()
		statusCode := c.Writer.Status()

		// 记录访问日志
		logger.Access(path, method, statusCode, latency, remoteAddr,
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("content_length", c.Writer.Size()),
		)

		// 记录错误日志
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error("HTTP request error",
					zap.String("path", path),
					zap.String("method", method),
					zap.Int("status", statusCode),
					zap.Error(err.Err),
				)
			}
		}
	}
}

// recoveryMiddleware 恢复中间件
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取请求信息
				path := c.Request.URL.Path
				method := c.Request.Method

				// 记录错误信息
				log.GlobalLogger.Error("HTTP panic",
					zap.String("path", path),
					zap.String("method", method),
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
				)

				// 返回错误响应
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置CORS头
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")

		// 处理OPTIONS请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// handleHealthCheck 处理健康检查请求
func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

// handleVersion 处理版本信息请求
func handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": "1.0.0", "build_time": time.Now().Format(time.RFC3339), "go_version": "unknown"})
}

// createHandleAIChat 创建AI聊天处理函数
func createHandleAIChat(agentManager ai_agent.AIAgentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received AI chat request", zap.String("path", c.Request.URL.Path))

		// 解析请求体
		var request struct {
			AgentID    string                 `json:"agent_id" binding:"required"`
			Messages   []interface{}          `json:"messages" binding:"required"`
			Parameters map[string]interface{} `json:"parameters"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			logLogger.Error("Invalid chat request format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// 获取AI Agent
		agent, err := agentManager.GetAIAgent(request.AgentID, nil)
		if err != nil {
			logLogger.Error("Failed to get AI agent", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "AI agent not found"})
			return
		}

		// 创建上下文
		ctx := c.Request.Context()

		// 构建聊天请求
		chatReq := ai_agent.ChatRequest{
			Messages: []ai_agent.Message{},
		}
		// 这里简化处理，实际项目中需要进行类型转换
		if len(request.Messages) > 0 {
			for _, msg := range request.Messages {
				msgMap, ok := msg.(map[string]interface{})
				if ok {
					role, _ := msgMap["role"].(string)
					content, _ := msgMap["content"].(string)
					chatReq.Messages = append(chatReq.Messages, ai_agent.Message{
						Role:    role,
						Content: content,
					})
				}
			}
		}

		// 调用AI Agent进行聊天
		response, err := agent.Chat(ctx, chatReq)
		if err != nil {
			logLogger.Error("AI chat failed", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat failed: " + err.Error()})
			return
		}

		// 返回响应
		c.JSON(http.StatusOK, response)
	}
}

// createHandleAICompletion 创建AI补全处理函数
func createHandleAICompletion(agentManager ai_agent.AIAgentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received AI completion request", zap.String("path", c.Request.URL.Path))

		// 解析请求体
		var request struct {
			AgentID    string                 `json:"agent_id" binding:"required"`
			Prompt     string                 `json:"prompt" binding:"required"`
			Parameters map[string]interface{} `json:"parameters"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			logLogger.Error("Invalid completion request format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// 获取AI Agent
		agent, err := agentManager.GetAIAgent(request.AgentID, nil)
		if err != nil {
			logLogger.Error("Failed to get AI agent", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "AI agent not found"})
			return
		}

		// 创建上下文
		ctx := c.Request.Context()

		// 构建补全请求
		completionReq := ai_agent.CompletionRequest{
			Prompt: request.Prompt,
		}

		// 调用AI Agent进行补全
		response, err := agent.Completion(ctx, completionReq)
		if err != nil {
			logLogger.Error("AI completion failed", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Completion failed: " + err.Error()})
			return
		}

		// 返回响应
		c.JSON(http.StatusOK, response)
	}
}

// createHandleAIEmbedding 创建AI嵌入处理函数
func createHandleAIEmbedding(agentManager ai_agent.AIAgentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received AI embedding request", zap.String("path", c.Request.URL.Path))

		// 解析请求体
		var request struct {
			AgentID    string                 `json:"agent_id" binding:"required"`
			Input      interface{}            `json:"input" binding:"required"`
			Parameters map[string]interface{} `json:"parameters"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			logLogger.Error("Invalid embedding request format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// 获取AI Agent
		agent, err := agentManager.GetAIAgent(request.AgentID, nil)
		if err != nil {
			logLogger.Error("Failed to get AI agent", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "AI agent not found"})
			return
		}

		// 创建上下文
		ctx := c.Request.Context()

		// 构建嵌入请求
		embeddingReq := ai_agent.EmbeddingRequest{
			Input: []string{},
		}
		// 处理输入数据
		if strSlice, ok := request.Input.([]interface{}); ok {
			for _, item := range strSlice {
				if str, ok := item.(string); ok {
					embeddingReq.Input = append(embeddingReq.Input, str)
				}
			}
		} else if str, ok := request.Input.(string); ok {
			embeddingReq.Input = append(embeddingReq.Input, str)
		}

		// 调用AI Agent获取嵌入
		response, err := agent.Embedding(ctx, embeddingReq)
		if err != nil {
			logLogger.Error("AI embedding failed", zap.String("agent_id", request.AgentID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Embedding failed: " + err.Error()})
			return
		}

		// 返回响应
		c.JSON(http.StatusOK, response)
	}
}

// createHandleListModels 创建获取模型列表处理函数
func createHandleListModels(agentManager ai_agent.AIAgentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received list models request", zap.String("path", c.Request.URL.Path))

		// 获取可用的AI Agent列表
		agents := agentManager.ListAvailableAgents()

		// 返回响应
		c.JSON(http.StatusOK, gin.H{"models": agents})
	}
}

// createHandleMCPCommand 创建MCP命令处理函数
func createHandleMCPCommand(mcpManager mcp.MCPServiceManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received MCP command request", zap.String("path", c.Request.URL.Path))

		// 解析请求体
		var request struct {
			ServiceID  string                 `json:"service_id" binding:"required"`
			Command    string                 `json:"command" binding:"required"`
			Parameters map[string]interface{} `json:"parameters"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			logLogger.Error("Invalid MCP command request format", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// 获取MCP服务
		service, err := mcpManager.GetMCPService(request.ServiceID, nil)
		if err != nil {
			logLogger.Error("Failed to get MCP service", zap.String("service_id", request.ServiceID), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP service not found"})
			return
		}

		// 创建上下文
		ctx := c.Request.Context()

		// 调用MCP服务执行命令
		// 注意：这里使用Call方法而不是Execute方法
		req := mcp.MCPServiceRequest{
			ServiceName: request.ServiceID,
			ToolName:    request.Command,
			Params:      request.Parameters,
		}
		response, err := service.Call(ctx, req)
		if err != nil {
			logLogger.Error("MCP command execution failed", zap.String("service_id", request.ServiceID), zap.String("command", request.Command), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Command execution failed: " + err.Error()})
			return
		}

		// 返回响应
		c.JSON(http.StatusOK, response)
	}
}

// createHandleListMCPServices 创建获取MCP服务列表处理函数
func createHandleListMCPServices(mcpManager mcp.MCPServiceManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取日志器
		logger, exists := c.Get("logger")
		var logLogger log.Logger
		if exists {
			logLogger = logger.(log.Logger)
		} else {
			logLogger = log.GlobalLogger
		}

		// 记录请求信息
		logLogger.Info("Received list MCP services request", zap.String("path", c.Request.URL.Path))

		// 获取服务列表
		services := mcpManager.ListAvailableServices()

		// 返回响应
		c.JSON(http.StatusOK, gin.H{"services": services})
	}
}

// createReverseProxyHandler 创建反向代理处理函数
func createReverseProxyHandler(logger log.Logger, targetURL string) gin.HandlerFunc {
	if logger == nil {
		logger = log.GlobalLogger
	}

	// 解析目标URL
	target, err := url.Parse(targetURL)
	if err != nil {
		logger.Error("Failed to parse target URL", zap.String("target_url", targetURL), zap.Error(err))
		return func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Proxy configuration error"})
		}
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义Director函数，保留原始请求路径
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// 记录代理请求信息
		logger.Info("Proxy request", zap.String("path", req.URL.Path), zap.String("target", targetURL))
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Proxy request failed", zap.String("path", r.URL.Path), zap.String("target", targetURL), zap.Error(err))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "Proxy request failed"}`))
	}

	return func(c *gin.Context) {
		// 执行代理请求
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// RegisterProxyRoutesFromConfig 从配置中注册代理路由（公开函数）
func RegisterProxyRoutesFromConfig(router *gin.Engine, logger log.Logger, onRouteRegistered func(string)) {
	// 从全局配置中获取代理路由配置
	proxyRoutes := config.GlobalConfig.ProxyRoutes

	if len(proxyRoutes) == 0 {
		logger.Info("No proxy routes configured")
		return
	}

	// 遍历并注册每个代理路由
	for _, route := range proxyRoutes {
		// 检查路由是否启用
		if !route.Enable {
			logger.Info("Skipping disabled proxy route", zap.String("path", route.Path))
			continue
		}

		// 检查路径和目标URL是否有效
		if route.Path == "" || route.TargetURL == "" {
			logger.Error("Invalid proxy route configuration", zap.String("path", route.Path), zap.String("target_url", route.TargetURL))
			continue
		}

		// 尝试注册代理路由，如果已存在则跳过
		// 由于Gin不提供直接检查路由是否存在的API，我们需要使用defer/recover来处理
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 捕获"handlers are already registered"错误
					if errStr, ok := r.(string); ok && strings.Contains(errStr, "handlers are already registered") {
						logger.Info("Route already registered, skipping", zap.String("path", route.Path))
					} else {
						// 其他错误重新抛出
						panic(r)
					}
				}
			}()

			// 注册代理路由
			router.Any(route.Path, createReverseProxyHandler(logger, route.TargetURL))
			logger.Info("Registered proxy route", zap.String("path", route.Path), zap.String("target_url", route.TargetURL))

			// 如果提供了回调函数，则调用它记录已注册的路由
			if onRouteRegistered != nil {
				onRouteRegistered(route.Path)
			}
		}()
	}
}

// registerProxyRoutesFromConfig 从配置中注册代理路由（内部调用公开函数）
func registerProxyRoutesFromConfig(router *gin.Engine, logger log.Logger, onRouteRegistered func(string)) {
	// 直接调用公开的函数
	RegisterProxyRoutesFromConfig(router, logger, onRouteRegistered)
}
