package router

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"kai/kaigate/pkg/log"
)

// 熔断器状态
const (
	StateClosed   = "closed"   // 正常状态
	StateOpen     = "open"     // 熔断状态
	StateHalfOpen = "half-open" // 半开状态（尝试恢复）
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	mutex               sync.Mutex
	state               string            // 当前状态
	errorThreshold      int               // 错误阈值
	timeout             time.Duration     // 熔断超时时间
	successThreshold    int               // 半开状态下的成功阈值
	errorCount          map[string]int    // 各服务的错误计数
	successCount        map[string]int    // 各服务的成功计数
	lastStateChange     map[string]time.Time // 各服务的上次状态变化时间
	serviceStates       map[string]string    // 各服务的当前状态
	disableFallback     bool              // 是否禁用熔断
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		errorThreshold:   5,
		timeout:          10 * time.Second,
		successThreshold: 2,
		errorCount:       make(map[string]int),
		successCount:     make(map[string]int),
		lastStateChange:  make(map[string]time.Time),
		serviceStates:    make(map[string]string),
		disableFallback:  false,
	}
}

// AllowRequest 检查是否允许请求通过
func (cb *CircuitBreaker) AllowRequest(serviceName string) bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 获取服务的当前状态
	state := cb.getServiceState(serviceName)

	// 如果禁用熔断，直接返回允许
	if cb.disableFallback {
		return true
	}

	switch state {
	case StateOpen:
		// 熔断状态下检查是否可以尝试恢复
		if cb.canTryAgain(serviceName) {
			// 进入半开状态
			cb.setServiceState(serviceName, StateHalfOpen)
			return true
		}
		// 熔断状态，拒绝请求
		log.GlobalLogger.Info("Circuit breaker is open, request rejected", zap.String("service", serviceName))
		return false

	case StateHalfOpen:
		// 半开状态，允许有限请求通过
		return true

	case StateClosed:
		// 正常状态，允许请求通过
		return true

	default:
		// 未知状态，默认允许
		return true
	}
}

// RecordSuccess 记录成功请求
func (cb *CircuitBreaker) RecordSuccess(serviceName string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 获取服务的当前状态
	state := cb.getServiceState(serviceName)

	// 只有在半开状态下需要处理成功计数
	if state == StateHalfOpen {
		// 增加成功计数
		cb.successCount[serviceName]++

		// 检查是否达到成功阈值
		if cb.successCount[serviceName] >= cb.successThreshold {
			// 重置状态，恢复到正常状态
			cb.resetServiceState(serviceName)
			cb.setServiceState(serviceName, StateClosed)
			log.GlobalLogger.Info("Circuit breaker closed, service recovered", zap.String("service", serviceName))
		}
	} else if state == StateClosed {
		// 正常状态下，重置错误计数
		cb.errorCount[serviceName] = 0
	}
}

// RecordFailure 记录失败请求
func (cb *CircuitBreaker) RecordFailure(serviceName string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 获取服务的当前状态
	state := cb.getServiceState(serviceName)

	switch state {
	case StateClosed:
		// 正常状态下，增加错误计数
		cb.errorCount[serviceName]++

		// 检查是否达到错误阈值
		if cb.errorCount[serviceName] >= cb.errorThreshold {
			// 触发熔断，进入开路状态
			cb.setServiceState(serviceName, StateOpen)
			log.GlobalLogger.Info("Circuit breaker opened due to too many errors",
				zap.String("service", serviceName),
				zap.Int("error_count", cb.errorCount[serviceName]),
			)
		}

	case StateHalfOpen:
		// 半开状态下，如果请求失败，立即回到开路状态
		cb.setServiceState(serviceName, StateOpen)
		log.GlobalLogger.Info("Circuit breaker re-opened during recovery", zap.String("service", serviceName))

	case StateOpen:
		// 开路状态下，不做处理
		// 错误计数可能需要额外处理
	}
}

// getServiceState 获取服务的当前状态
func (cb *CircuitBreaker) getServiceState(serviceName string) string {
	state, exists := cb.serviceStates[serviceName]
	if !exists {
		// 如果服务状态不存在，默认设置为正常状态
		cb.serviceStates[serviceName] = StateClosed
		return StateClosed
	}
	return state
}

// setServiceState 设置服务的状态
func (cb *CircuitBreaker) setServiceState(serviceName, state string) {
	cb.serviceStates[serviceName] = state
	cb.lastStateChange[serviceName] = time.Now()
}

// resetServiceState 重置服务的状态计数
func (cb *CircuitBreaker) resetServiceState(serviceName string) {
	cb.errorCount[serviceName] = 0
	cb.successCount[serviceName] = 0
}

// canTryAgain 检查是否可以尝试恢复
func (cb *CircuitBreaker) canTryAgain(serviceName string) bool {
	lastChange, exists := cb.lastStateChange[serviceName]
	if !exists {
		return false
	}

	// 检查是否超过了熔断超时时间
	timeSinceLastChange := time.Since(lastChange)
	return timeSinceLastChange >= cb.timeout
}

// GetState 获取熔断器状态
func (cb *CircuitBreaker) GetState() map[string]interface{} {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return map[string]interface{}{
		"global_state":        cb.state,
		"error_threshold":     cb.errorThreshold,
		"timeout":             cb.timeout,
		"success_threshold":   cb.successThreshold,
		"disable_fallback":    cb.disableFallback,
		"service_states":      cb.serviceStates,
		"error_counts":        cb.errorCount,
		"success_counts":      cb.successCount,
		"last_state_changes":  cb.lastStateChange,
	}
}

// SetErrorThreshold 设置错误阈值
func (cb *CircuitBreaker) SetErrorThreshold(threshold int) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.errorThreshold = threshold
}

// SetTimeout 设置熔断超时时间
func (cb *CircuitBreaker) SetTimeout(timeout time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.timeout = timeout
}

// SetSuccessThreshold 设置成功阈值
func (cb *CircuitBreaker) SetSuccessThreshold(threshold int) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.successThreshold = threshold
}

// EnableFallback 启用熔断
func (cb *CircuitBreaker) EnableFallback() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.disableFallback = false
}

// DisableFallback 禁用熔断
func (cb *CircuitBreaker) DisableFallback() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.disableFallback = true
}

// ResetService 重置指定服务的熔断器状态
func (cb *CircuitBreaker) ResetService(serviceName string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.resetServiceState(serviceName)
	cb.setServiceState(serviceName, StateClosed)
	log.GlobalLogger.Info("Circuit breaker reset for service", zap.String("service", serviceName))
}

// ResetAll 重置所有服务的熔断器状态
func (cb *CircuitBreaker) ResetAll() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	for serviceName := range cb.serviceStates {
		cb.resetServiceState(serviceName)
		cb.setServiceState(serviceName, StateClosed)
	}

	log.GlobalLogger.Info("All circuit breakers reset")
}