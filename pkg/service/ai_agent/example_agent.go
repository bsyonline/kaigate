package ai_agent

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// ExampleAIAgent 示例AI代理实现
// 用于演示如何使用AI代理接口和基础类
type ExampleAIAgent struct {
	*BaseAIAgent
}

// ExampleAIAgentFactory ExampleAIAgent的工厂实现
type ExampleAIAgentFactory struct {}

// NewExampleAIAgent 创建ExampleAIAgent实例
func NewExampleAIAgent() *ExampleAIAgent {
	return &ExampleAIAgent{
		BaseAIAgent: NewBaseAIAgent("example-ai-agent", "1.0.0"),
	}
}

// Init 初始化ExampleAIAgent
func (e *ExampleAIAgent) Init(config map[string]interface{}) error {
	// 调用基础类的Init方法
	if err := e.BaseAIAgent.Init(config); err != nil {
		return err
	}

	// 添加ExampleAIAgent特有的初始化逻辑
	e.GetLogger().Info("ExampleAIAgent initialized with custom logic")
	return nil
}

// Chat 实现聊天功能
func (e *ExampleAIAgent) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 记录聊天请求
	e.GetLogger().Info("Processing chat request",
		zap.String("model", req.Model),
		zap.Int("messages_count", len(req.Messages)),
	)

	// 模拟处理延迟
	time.Sleep(200 * time.Millisecond)

	// 构建响应
	resp := &ChatResponse{
		ID:      "chat-" + time.Now().Format("20060102-150405.000"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []struct {
			Index   int    `json:"index"`
			Message Message `json:"message"`
		}{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: "This is a response from ExampleAIAgent.",
			},
		}},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{},
	}

	return resp, nil
}

// ChatStream 实现流式聊天功能
func (e *ExampleAIAgent) ChatStream(ctx context.Context, req ChatRequest) (<-chan *ChatResponse, <-chan error) {
	respChan := make(chan *ChatResponse)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		// 模拟流式响应
		for i := 0; i < 3; i++ {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			// 构造部分响应
			response := &ChatResponse{
				ID:      "chat-" + time.Now().Format("20060102-150405.000"),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []struct {
					Index   int    `json:"index"`
					Message Message `json:"message"`
				}{{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "Chunk from ExampleAIAgent.",
					},
				}},
			}

			// 发送响应
			respChan <- response

			// 模拟延迟
			time.Sleep(100 * time.Millisecond)
		}
	}()

	return respChan, errChan
}

// Completion 实现文本生成功能
func (e *ExampleAIAgent) Completion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// 记录生成请求
	e.GetLogger().Info("Processing completion request",
		zap.String("model", req.Model),
	)

	// 模拟处理延迟
	time.Sleep(150 * time.Millisecond)

	// 构建响应
	resp := &CompletionResponse{
		ID:      "completion-" + time.Now().Format("20060102-150405.000"),
		Object:  "text.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []struct {
			Index int    `json:"index"`
			Text  string `json:"text"`
		}{{
			Index: 0,
			Text:  "This is a completion response from ExampleAIAgent.",
		}},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{},
	}

	return resp, nil
}

// CompletionStream 实现流式文本生成功能
func (e *ExampleAIAgent) CompletionStream(ctx context.Context, req CompletionRequest) (<-chan *CompletionResponse, <-chan error) {
	respChan := make(chan *CompletionResponse)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		// 模拟流式响应
		for i := 0; i < 3; i++ {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			// 构造部分响应
			response := &CompletionResponse{
				ID:      "completion-" + time.Now().Format("20060102-150405.000"),
				Object:  "text.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []struct {
					Index int    `json:"index"`
					Text  string `json:"text"`
				}{{
					Index: 0,
					Text:  "Chunk from ExampleAIAgent.",
				}},
			}

			// 发送响应
			respChan <- response

			// 模拟延迟
			time.Sleep(100 * time.Millisecond)
		}
	}()

	return respChan, errChan
}

// Embedding 实现嵌入向量生成功能
func (e *ExampleAIAgent) Embedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, errors.New("Embedding not implemented")
}

// BatchEmbedding 实现批量嵌入向量生成功能
func (e *ExampleAIAgent) BatchEmbedding(ctx context.Context, req []EmbeddingRequest) ([]*EmbeddingResponse, error) {
	return nil, errors.New("BatchEmbedding not implemented")
}

// ListModels 实现模型列表查询功能
func (e *ExampleAIAgent) ListModels(ctx context.Context) ([]string, error) {
	return []string{"example-model-1", "example-model-2"}, nil
}

// GetModel 实现模型详情查询功能
func (e *ExampleAIAgent) GetModel(ctx context.Context, modelName string) (map[string]interface{}, error) {
	modelInfo := map[string]interface{}{
		"name":         modelName,
		"description":  "Example model",
		"context_size": 4096,
		"version":      "1.0.0",
	}
	return modelInfo, nil
}

// HealthCheck 检查AI Agent健康状态
func (e *ExampleAIAgent) HealthCheck() error {
	return nil
}

// Create 实现AIAgentFactory接口的Create方法
func (f *ExampleAIAgentFactory) Create() (AIAgent, error) {
	return NewExampleAIAgent(), nil
}

// Name 实现AIAgentFactory接口的Name方法
func (f *ExampleAIAgentFactory) Name() string {
	return "example-ai-agent"
}