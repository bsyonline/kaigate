package mcp

import (
	"errors"
	"sync"

	"go.uber.org/zap"
	"kai/kaigate/pkg/log"
)

// DefaultMCPServiceManager MCP服务管理器默认实现
// 负责MCP服务的注册、获取和生命周期管理
type DefaultMCPServiceManager struct {
	factories  map[string]MCPServiceFactory
	instances  map[string]MCPService
	configs    map[string]map[string]interface{}
	mutex      sync.RWMutex
	logger     log.Logger
}

// NewDefaultMCPServiceManager 创建DefaultMCPServiceManager实例
func NewDefaultMCPServiceManager() *DefaultMCPServiceManager {
	return &DefaultMCPServiceManager{
		factories:  make(map[string]MCPServiceFactory),
		instances:  make(map[string]MCPService),
		configs:    make(map[string]map[string]interface{}),
		logger:     log.GlobalLogger,
	}
}

// RegisterFactory 注册MCP服务工厂
func (m *DefaultMCPServiceManager) RegisterFactory(factory MCPServiceFactory) error {
	if factory == nil {
		return errors.New("factory cannot be nil")
	}

	name := factory.Name()
	if name == "" {
		return errors.New("factory name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.factories[name]; exists {
		m.logger.Warn("Factory already registered",
			zap.String("factory_name", name),
		)
		return errors.New("factory already registered")
	}

	m.factories[name] = factory
	m.logger.Info("MCP service factory registered",
		zap.String("factory_name", name),
	)
	return nil
}

// GetMCPService 获取MCP服务实例
func (m *DefaultMCPServiceManager) GetMCPService(name string, config map[string]interface{}) (MCPService, error) {
	if name == "" {
		return nil, errors.New("service name cannot be empty")
	}

	// 首先检查是否已经存在实例
	m.mutex.RLock()
	service, exists := m.instances[name]
	m.mutex.RUnlock()

	if exists {
		return service, nil
	}

	// 如果不存在，创建新实例
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 双重检查锁定模式
	service, exists = m.instances[name]
	if exists {
		return service, nil
	}

	// 获取工厂
	factory, factoryExists := m.factories[name]
	if !factoryExists {
		m.logger.Error("Factory not found",
			zap.String("factory_name", name),
		)
		return nil, errors.New("factory not found")
	}

	// 创建服务实例
	service, err := factory.Create()
	if err != nil {
		m.logger.Error("Failed to create service",
			zap.String("service_name", name),
			zap.Error(err),
		)
		return nil, err
	}

	// 保存配置
	if config == nil {
		config = make(map[string]interface{})
	}
	m.configs[name] = config

	// 初始化服务
	if err := service.Init(config); err != nil {
		m.logger.Error("Failed to initialize service",
			zap.String("service_name", name),
			zap.Error(err),
		)
		return nil, err
	}

	// 保存实例
	m.instances[name] = service
	m.logger.Info("MCP service instance created",
		zap.String("service_name", name),
	)
	return service, nil
}

// ReleaseMCPService 释放MCP服务实例
func (m *DefaultMCPServiceManager) ReleaseMCPService(name string) error {
	if name == "" {
		return errors.New("service name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	service, exists := m.instances[name]
	if !exists {
		return errors.New("service instance not found")
	}

	// 关闭服务
	if err := service.Close(); err != nil {
		m.logger.Warn("Failed to close service",
			zap.String("service_name", name),
			zap.Error(err),
		)
	}

	// 删除实例和配置
	delete(m.instances, name)
	delete(m.configs, name)
	m.logger.Info("MCP service instance released",
		zap.String("service_name", name),
	)
	return nil
}

// ListAvailableServices 列出所有可用的MCP服务
func (m *DefaultMCPServiceManager) ListAvailableServices() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := make([]string, 0, len(m.factories))
	for name := range m.factories {
		services = append(services, name)
	}
	return services
}

// Close 关闭MCP服务管理器
func (m *DefaultMCPServiceManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 关闭所有服务实例
	for name, service := range m.instances {
		if err := service.Close(); err != nil {
			m.logger.Warn("Failed to close service",
				zap.String("service_name", name),
				zap.Error(err),
			)
		}
	}

	// 清空映射
	m.instances = make(map[string]MCPService)
	m.configs = make(map[string]map[string]interface{})
	m.logger.Info("MCP service manager closed")
	return nil
}

// GetFactory 获取MCP服务工厂
func (m *DefaultMCPServiceManager) GetFactory(name string) (MCPServiceFactory, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	factory, exists := m.factories[name]
	return factory, exists
}

// GetConfig 获取MCP服务配置
func (m *DefaultMCPServiceManager) GetConfig(name string) (map[string]interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	config, exists := m.configs[name]
	return config, exists
}