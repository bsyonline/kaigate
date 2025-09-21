package router

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"kai/kaigate/pkg/log"
)

// RateLimiter 限流控制器
type RateLimiter struct {
	rate       int           // 每秒允许的请求数
	burst      int           // 最大突发请求数
	mutex      sync.Mutex    // 互斥锁
	tokens     float64       // 当前可用令牌数
	lastRefill time.Time     // 上次填充令牌的时间
	enabled    bool          // 是否启用
}

// NewRateLimiter 创建限流控制器
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
		enabled:    true,
	}
}

// Allow 检查是否允许请求通过
func (rl *RateLimiter) Allow() bool {
	if !rl.enabled {
		return true
	}

	// 计算应该添加的令牌数
	now := time.Now()
	ratePerSecond := float64(rl.rate)

	// 加锁保护共享资源
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// 计算时间差并填充令牌
	duration := now.Sub(rl.lastRefill).Seconds()
	newTokens := rl.tokens + duration*ratePerSecond

	// 限制最大令牌数为突发数
	if newTokens > float64(rl.burst) {
		newTokens = float64(rl.burst)
	}

	// 更新令牌数和最后填充时间
	rl.tokens = newTokens
	rl.lastRefill = now

	// 检查是否有足够的令牌
	if rl.tokens >= 1.0 {
		// 消耗一个令牌
		rl.tokens--
		return true
	}

	// 令牌不足，拒绝请求
	return false
}

// SetRate 设置限流速率
func (rl *RateLimiter) SetRate(rate int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.rate = rate
}

// SetBurst 设置最大突发请求数
func (rl *RateLimiter) SetBurst(burst int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.burst = burst
	// 如果当前令牌数超过新的突发数，调整令牌数
	if rl.tokens > float64(burst) {
		rl.tokens = float64(burst)
	}
}

// Enable 启用限流
func (rl *RateLimiter) Enable() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.enabled = true
}

// Disable 禁用限流
func (rl *RateLimiter) Disable() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.enabled = false
}

// GetState 获取限流控制器状态
func (rl *RateLimiter) GetState() map[string]interface{} {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	return map[string]interface{}{
		"rate":       rl.rate,
		"burst":      rl.burst,
		"tokens":     rl.tokens,
		"enabled":    rl.enabled,
		"last_refill": rl.lastRefill,
	}
}

// RateLimitManager 限流管理器
type RateLimitManager struct {
	rateLimiters map[string]*RateLimiter
	mutex        sync.RWMutex
	defaultRate  int
	defaultBurst int
}

// NewRateLimitManager 创建限流管理器
func NewRateLimitManager(defaultRate, defaultBurst int) *RateLimitManager {
	return &RateLimitManager{
		rateLimiters: make(map[string]*RateLimiter),
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
	}
}

// GetRateLimiter 获取或创建限流控制器
func (rlm *RateLimitManager) GetRateLimiter(key string) *RateLimiter {
	// 先尝试不加锁获取
	rlm.mutex.RLock()
	rl, ok := rlm.rateLimiters[key]
	rlm.mutex.RUnlock()

	if ok {
		return rl
	}

	// 如果不存在，创建新的限流控制器
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	// 再次检查，防止竞态条件
	rl, ok = rlm.rateLimiters[key]
	if ok {
		return rl
	}

	// 创建新的限流控制器
	rl = NewRateLimiter(rlm.defaultRate, rlm.defaultBurst)
	rlm.rateLimiters[key] = rl

	log.GlobalLogger.Info("Rate limiter created",
		zap.String("key", key),
		zap.Int("rate", rlm.defaultRate),
		zap.Int("burst", rlm.defaultBurst),
	)

	return rl
}

// RemoveRateLimiter 移除限流控制器
func (rlm *RateLimitManager) RemoveRateLimiter(key string) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	delete(rlm.rateLimiters, key)

	log.GlobalLogger.Info("Rate limiter removed", zap.String("key", key))
}

// UpdateRateLimiter 更新限流控制器配置
func (rlm *RateLimitManager) UpdateRateLimiter(key string, rate, burst int) {
	// 获取限流控制器
	rl := rlm.GetRateLimiter(key)
	// 更新配置
	rl.SetRate(rate)
	rl.SetBurst(burst)

	log.GlobalLogger.Info("Rate limiter updated",
		zap.String("key", key),
		zap.Int("rate", rate),
		zap.Int("burst", burst),
	)
}

// EnableRateLimiter 启用指定限流控制器
func (rlm *RateLimitManager) EnableRateLimiter(key string) {
	// 获取限流控制器
	rl := rlm.GetRateLimiter(key)
	// 启用限流
	rl.Enable()

	log.GlobalLogger.Info("Rate limiter enabled", zap.String("key", key))
}

// DisableRateLimiter 禁用指定限流控制器
func (rlm *RateLimitManager) DisableRateLimiter(key string) {
	// 获取限流控制器
	rl := rlm.GetRateLimiter(key)
	// 禁用限流
	rl.Disable()

	log.GlobalLogger.Info("Rate limiter disabled", zap.String("key", key))
}

// GetAllRateLimiters 获取所有限流控制器
func (rlm *RateLimitManager) GetAllRateLimiters() map[string]map[string]interface{} {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()

	result := make(map[string]map[string]interface{})
	for key, rl := range rlm.rateLimiters {
		result[key] = rl.GetState()
	}

	return result
}