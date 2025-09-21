package ai_agent

import (
	"errors"
	"sync"

	"go.uber.org/zap"
	"kai/kaigate/pkg/log"
)

// DefaultAIAgentManager 默认的AI Agent管理器实现
// 用于管理多个AI Agent实例
type DefaultAIAgentManager struct {
	factories    map[string]AIAgentFactory
	instances    map[string]AIAgent
	configs      map[string]map[string]interface{}
	mutex        sync.RWMutex
	logger       log.Logger
}

// NewDefaultAIAgentManager 创建DefaultAIAgentManager实例
func NewDefaultAIAgentManager() *DefaultAIAgentManager {
	return &DefaultAIAgentManager{
		factories: make(map[string]AIAgentFactory),
		instances: make(map[string]AIAgent),
		configs:   make(map[string]map[string]interface{}),
		logger:    log.GlobalLogger,
	}
}

// RegisterFactory 注册AI Agent工厂
func (m *DefaultAIAgentManager) RegisterFactory(factory AIAgentFactory) error {
	if factory == nil {
		return errors.New("factory cannot be nil")
	}

	name := factory.Name()
	if name == "" {
		return errors.New("factory name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.factories[name] = factory
	m.logger.Info("Registered AI Agent factory",
		zap.String("name", name),
	)
	return nil
}

// GetAIAgent 创建并获取AI Agent实例
func (m *DefaultAIAgentManager) GetAIAgent(name string, config map[string]interface{}) (AIAgent, error) {
	// 先检查是否已有实例
	m.mutex.RLock()
	agent, exists := m.instances[name]
	m.mutex.RUnlock()

	if exists {
		return agent, nil
	}

	// 如果没有实例，创建一个新的
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 双重检查
	agent, exists = m.instances[name]
	if exists {
		return agent, nil
	}

	// 获取工厂
	factory, exists := m.factories[name]
	if !exists {
		return nil, errors.New("AI Agent factory not found: " + name)
	}

	// 创建实例
	instance, err := factory.Create()
	if err != nil {
		m.logger.Error("Failed to create AI Agent instance",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}

	// 初始化实例
	if config == nil {
		config = make(map[string]interface{})
	}

	if err := instance.Init(config); err != nil {
		m.logger.Error("Failed to initialize AI Agent instance",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}

	// 保存实例和配置
	m.instances[name] = instance
	m.configs[name] = config

	m.logger.Info("Created and initialized AI Agent instance",
		zap.String("name", name),
	)
	return instance, nil
}

// ReleaseAIAgent 释放AI Agent实例
func (m *DefaultAIAgentManager) ReleaseAIAgent(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.instances[name]
	if !exists {
		return nil // 实例不存在，视为成功
	}

	// 关闭实例
	if err := instance.Close(); err != nil {
		m.logger.Error("Failed to close AI Agent instance",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// 移除实例和配置
	delete(m.instances, name)
	delete(m.configs, name)

	m.logger.Info("Released AI Agent instance",
		zap.String("name", name),
	)
	return nil
}

// ListAvailableAgents 列出所有可用的AI Agent名称
func (m *DefaultAIAgentManager) ListAvailableAgents() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agents := make([]string, 0, len(m.factories))
	for name := range m.factories {
		agents = append(agents, name)
	}

	return agents
}

// Close 清理所有资源
func (m *DefaultAIAgentManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 关闭所有实例
	for name, instance := range m.instances {
		if err := instance.Close(); err != nil {
			m.logger.Error("Failed to close AI Agent instance",
				zap.String("name", name),
				zap.Error(err),
			)
		}
	}

	// 清空映射
	m.instances = make(map[string]AIAgent)
	m.configs = make(map[string]map[string]interface{})

	m.logger.Info("AI Agent manager closed")
	return nil
}

// GetFactory 获取工厂
func (m *DefaultAIAgentManager) GetFactory(name string) (AIAgentFactory, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	factory, exists := m.factories[name]
	return factory, exists
}

// GetConfig 获取实例配置
func (m *DefaultAIAgentManager) GetConfig(name string) (map[string]interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	config, exists := m.configs[name]
	return config, exists
}