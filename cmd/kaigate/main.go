package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"kai/kaigate/pkg/bootstrap"
	"kai/kaigate/pkg/config"
	"kai/kaigate/pkg/log"
	"kai/kaigate/pkg/service/ai_agent"
	"kai/kaigate/pkg/service/mcp"
)

func main() {
	// 解析命令行参数
	configFile := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// 初始化配置
	if err := config.InitConfig(*configFile); err != nil {
		// 使用标准输出，因为日志系统可能尚未初始化
		println("Failed to initialize configuration:", err.Error())
		os.Exit(1)
	}

	// 初始化日志系统
	if err := log.InitLogger(
		config.GlobalConfig.Log.Level,
		config.GlobalConfig.Log.Format,
		config.GlobalConfig.Log.File,
		config.GlobalConfig.Log.Stdout,
	); err != nil {
		println("Failed to initialize logger:", err.Error())
		os.Exit(1)
	}

	logger := log.GlobalLogger
	logger.Info("Starting KaiGate service")
	logger.Info("Service version: " + config.ServiceVersion)

	// 创建AI Agent管理器
	agentManager := ai_agent.NewDefaultAIAgentManager()

	// 注册示例AI Agent工厂
	agentManager.RegisterFactory(&ai_agent.ExampleAIAgentFactory{})

	// 创建MCP服务管理器
	mcpManager := mcp.NewDefaultMCPServiceManager()

	// 注册示例MCP服务工厂
	mcpManager.RegisterFactory(&mcp.ExampleMCPServiceFactory{})

	// 创建服务器实例
	server := bootstrap.NewServer(
		bootstrap.WithLogger(logger),
		bootstrap.WithAIAgentManager(agentManager),
		bootstrap.WithMCPServiceManager(mcpManager),
	)

	// 启动服务器
	if err := server.Start(); err != nil {
		logger.Error("Failed to start server", zap.Error(err))
		os.Exit(1)
	}

	// 监听系统信号，优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh

	logger.Info("Shutting down server...")

	// 关闭服务器
	server.Stop()

	// 关闭AI Agent管理器
	if err := agentManager.Close(); err != nil {
		logger.Error("Failed to close AI Agent manager", zap.Error(err))
	}

	// 关闭MCP服务管理器
	if err := mcpManager.Close(); err != nil {
		logger.Error("Failed to close MCP service manager", zap.Error(err))
	}

	logger.Info("Server shutdown completed")
}
