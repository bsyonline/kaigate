package mcp

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ExampleMCPService 示例MCP服务实现
// 用于演示如何使用MCP服务接口和基础类
type ExampleMCPService struct {
	*BaseMCPService
}

// ExampleMCPServiceFactory ExampleMCPService的工厂实现
type ExampleMCPServiceFactory struct {}

// NewExampleMCPService 创建ExampleMCPService实例
func NewExampleMCPService() *ExampleMCPService {
	return &ExampleMCPService{
		BaseMCPService: NewBaseMCPService("example-mcp-service", "1.0.0"),
	}
}

// Init 初始化ExampleMCPService
func (e *ExampleMCPService) Init(config map[string]interface{}) error {
	// 调用基础类的Init方法
	if err := e.BaseMCPService.Init(config); err != nil {
		return err
	}

	// 添加ExampleMCPService特有的初始化逻辑
	e.GetLogger().Info("ExampleMCPService initialized with custom logic")
	return nil
}

// Call 实现调用MCP服务功能
func (e *ExampleMCPService) Call(ctx context.Context, req MCPServiceRequest) (*MCPServiceResponse, error) {
	// 模拟处理MCP服务请求
	e.GetLogger().Info("Processing MCP service request",
		zap.String("service_name", req.ServiceName),
		zap.String("tool_name", req.ToolName),
	)

	// 模拟延迟
	time.Sleep(100 * time.Millisecond)

	// 根据工具名称处理不同的请求
	switch req.ToolName {
	case "echo":
		// 模拟回显功能
		return e.CreateSuccessResponse(map[string]interface{}{
			"echo": req.Params,
		}), nil
	case "get_time":
		// 模拟获取当前时间功能
		return e.CreateSuccessResponse(map[string]interface{}{
			"current_time": time.Now().Format(time.RFC3339),
		}), nil
	case "calculate":
		// 模拟简单计算功能
		return e.handleCalculateRequest(req)
	default:
		// 未知工具
		return e.CreateErrorResponse("UNKNOWN_TOOL", fmt.Sprintf("Unknown tool: %s", req.ToolName)), nil
	}
}

// 处理计算请求
func (e *ExampleMCPService) handleCalculateRequest(req MCPServiceRequest) (*MCPServiceResponse, error) {
	// 检查参数
	if req.Params == nil {
		return e.CreateErrorResponse("INVALID_PARAMS", "Params cannot be nil"), nil
	}

	a, aOk := req.Params["a"].(float64)
	b, bOk := req.Params["b"].(float64)
	op, opOk := req.Params["operation"].(string)

	if !aOk || !bOk || !opOk {
		return e.CreateErrorResponse("INVALID_PARAMS", "Missing or invalid parameters"), nil
	}

	// 执行计算
	var result float64
	switch op {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return e.CreateErrorResponse("DIVISION_BY_ZERO", "Cannot divide by zero"), nil
		}
		result = a / b
	default:
		return e.CreateErrorResponse("INVALID_OPERATION", fmt.Sprintf("Unknown operation: %s", op)), nil
	}

	// 返回结果
	return e.CreateSuccessResponse(map[string]interface{}{
		"result":    result,
		"operation": op,
		"a":         a,
		"b":         b,
	}), nil
}

// ListServices 实现列出可用的MCP服务功能
func (e *ExampleMCPService) ListServices(ctx context.Context) ([]string, error) {
	// 返回模拟的服务列表
	return []string{"echo", "get_time", "calculate"}, nil
}

// Create 实现MCPServiceFactory接口的Create方法
func (f *ExampleMCPServiceFactory) Create() (MCPService, error) {
	return NewExampleMCPService(), nil
}

// Name 实现MCPServiceFactory接口的Name方法
func (f *ExampleMCPServiceFactory) Name() string {
	return "example-mcp-service"
}