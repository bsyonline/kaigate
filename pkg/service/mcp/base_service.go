package mcp

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"kai/kaigate/pkg/log"
)

// BaseMCPService MCP服务基础实现
// 提供通用功能和默认实现，可作为其他具体MCP服务实现的父类
type BaseMCPService struct {
	name    string
	version string
	config  map[string]interface{}
	logger  log.Logger
}

// NewBaseMCPService 创建BaseMCPService实例
func NewBaseMCPService(name, version string) *BaseMCPService {
	return &BaseMCPService{
		name:    name,
		version: version,
		config:  make(map[string]interface{}),
		logger:  log.GlobalLogger,
	}
}

// Init 初始化MCP服务
func (b *BaseMCPService) Init(config map[string]interface{}) error {
	if config == nil {
		config = make(map[string]interface{})
	}
	b.config = config
	b.logger.Info("MCP Service initialized",
		zap.String("name", b.name),
		zap.String("version", b.version),
	)
	return nil
}

// Close 清理资源
func (b *BaseMCPService) Close() error {
	b.logger.Info("MCP Service closed",
		zap.String("name", b.name),
	)
	return nil
}

// Name 获取MCP服务名称
func (b *BaseMCPService) Name() string {
	return b.name
}

// Version 获取MCP服务版本
func (b *BaseMCPService) Version() string {
	return b.version
}

// Call 调用MCP服务默认实现
func (b *BaseMCPService) Call(ctx context.Context, req MCPServiceRequest) (*MCPServiceResponse, error) {
	return nil, errors.New("Call not implemented")
}

// CallAsync 异步调用MCP服务默认实现
func (b *BaseMCPService) CallAsync(ctx context.Context, req MCPServiceRequest, callback func(*MCPServiceResponse, error)) error {
	go func() {
		resp, err := b.Call(ctx, req)
		callback(resp, err)
	}()
	return nil
}

// BatchCall 批量调用MCP服务默认实现
func (b *BaseMCPService) BatchCall(ctx context.Context, reqs []MCPServiceRequest) ([]*MCPServiceResponse, error) {
	return nil, errors.New("BatchCall not implemented")
}

// ListServices 列出可用的MCP服务默认实现
func (b *BaseMCPService) ListServices(ctx context.Context) ([]string, error) {
	return nil, errors.New("ListServices not implemented")
}

// GetService 获取MCP服务详情默认实现
func (b *BaseMCPService) GetService(ctx context.Context, serviceName string) (map[string]interface{}, error) {
	return nil, errors.New("GetService not implemented")
}

// HealthCheck 检查MCP服务健康状态默认实现
func (b *BaseMCPService) HealthCheck() error {
	return nil
}

// GetConfig 获取配置
func (b *BaseMCPService) GetConfig() map[string]interface{} {
	return b.config
}

// GetLogger 获取日志记录器
func (b *BaseMCPService) GetLogger() log.Logger {
	return b.logger
}

// SetConfig 设置配置
func (b *BaseMCPService) SetConfig(key string, value interface{}) {
	b.config[key] = value
}

// GetConfigValue 获取配置值
func (b *BaseMCPService) GetConfigValue(key string) (interface{}, bool) {
	value, ok := b.config[key]
	return value, ok
}

// CreateSuccessResponse 创建成功响应
func (b *BaseMCPService) CreateSuccessResponse(data interface{}) *MCPServiceResponse {
	return &MCPServiceResponse{
		Success: true,
		Data:    data,
	}
}

// CreateErrorResponse 创建错误响应
func (b *BaseMCPService) CreateErrorResponse(code string, message string) *MCPServiceResponse {
	return &MCPServiceResponse{
		Success: false,
		Error: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
}