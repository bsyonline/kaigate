package mcp

import (
	"context"
)

// MCPServiceRequest MCP服务请求
// 定义了向MCP服务发送请求的数据结构
type MCPServiceRequest struct {
	ServiceName string                 `json:"service_name"` // MCP服务名称
	ToolName    string                 `json:"tool_name"`    // MCP工具名称
	Params      map[string]interface{} `json:"params"`       // 请求参数
}

// MCPServiceResponse MCP服务响应
// 定义了MCP服务返回的响应数据结构
type MCPServiceResponse struct {
	Success bool                   `json:"success"` // 操作是否成功
	Data    interface{}            `json:"data,omitempty"` // 响应数据
	Error   map[string]interface{} `json:"error,omitempty"` // 错误信息
}

// MCPService MCP服务接口
// 所有MCP服务实现都需要实现此接口
// 用于标准化MCP服务的调用方式
// 支持同步和异步调用模式
type MCPService interface {
	// 初始化MCP服务
	Init(config map[string]interface{}) error
	
	// 清理资源
	Close() error
	
	// 获取MCP服务名称
	Name() string
	
	// 获取MCP服务版本
	Version() string
	
	// 调用MCP服务
	Call(ctx context.Context, req MCPServiceRequest) (*MCPServiceResponse, error)
	
	// 异步调用MCP服务
	CallAsync(ctx context.Context, req MCPServiceRequest, callback func(*MCPServiceResponse, error)) error
	
	// 批量调用MCP服务
	BatchCall(ctx context.Context, reqs []MCPServiceRequest) ([]*MCPServiceResponse, error)
	
	// 列出可用的MCP服务
	ListServices(ctx context.Context) ([]string, error)
	
	// 获取MCP服务详情
	GetService(ctx context.Context, serviceName string) (map[string]interface{}, error)
	
	// 检查MCP服务健康状态
	HealthCheck() error
}

// MCPServiceFactory MCP服务工厂接口
// 用于创建MCPService实例
type MCPServiceFactory interface {
	// 创建MCPService实例
	Create() (MCPService, error)
	
	// 获取工厂名称
	Name() string
}

// MCPServiceManager MCP服务管理器
// 用于管理多个MCPService实例
type MCPServiceManager interface {
	// 注册MCP服务工厂
	RegisterFactory(factory MCPServiceFactory) error
	
	// 创建并获取MCP服务实例
	GetMCPService(name string, config map[string]interface{}) (MCPService, error)
	
	// 释放MCP服务实例
	ReleaseMCPService(name string) error
	
	// 列出所有可用的MCP服务名称
	ListAvailableServices() []string
	
	// 清理所有资源
	Close() error
}

// MCPServiceMiddleware MCP服务中间件接口
// 用于在MCP服务调用前后添加额外的处理逻辑
type MCPServiceMiddleware interface {
	// 执行中间件逻辑
	Process(ctx context.Context, req interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error)
	
	// 获取中间件名称
	Name() string
}