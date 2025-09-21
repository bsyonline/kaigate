package ai_agent

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"kai/kaigate/pkg/log"
)

// BaseAIAgent AI代理的基础实现
// 提供通用功能和默认实现，可作为其他具体AI代理实现的父类
type BaseAIAgent struct {
	name    string
	version string
	config  map[string]interface{}
	logger  log.Logger
}

// NewBaseAIAgent 创建BaseAIAgent实例
func NewBaseAIAgent(name, version string) *BaseAIAgent {
	return &BaseAIAgent{
		name:    name,
		version: version,
		config:  make(map[string]interface{}),
		logger:  log.GlobalLogger,
	}
}

// Init 初始化AI代理
func (b *BaseAIAgent) Init(config map[string]interface{}) error {
	if config == nil {
		config = make(map[string]interface{})
	}
	b.config = config
	b.logger.Info("AI Agent initialized",
		zap.String("name", b.name),
		zap.String("version", b.version),
	)
	return nil
}

// Close 清理资源
func (b *BaseAIAgent) Close() error {
	b.logger.Info("AI Agent closed",
		zap.String("name", b.name),
	)
	return nil
}

// Name 获取AI代理名称
func (b *BaseAIAgent) Name() string {
	return b.name
}

// Version 获取AI代理版本
func (b *BaseAIAgent) Version() string {
	return b.version
}

// Chat 实现AIAgent接口的Chat方法
// 提供默认实现，抛出未实现错误
func (b *BaseAIAgent) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return nil, errors.New("Chat not implemented")
}

// Completion 实现AIAgent接口的Completion方法
// 提供默认实现，抛出未实现错误
func (b *BaseAIAgent) Completion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return nil, errors.New("Completion not implemented")
}

// Embedding 实现AIAgent接口的Embedding方法
// 提供默认实现，抛出未实现错误
func (b *BaseAIAgent) Embedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, errors.New("Embedding not implemented")
}

// GetLogger 获取日志记录器
func (b *BaseAIAgent) GetLogger() log.Logger {
	return b.logger
}