package config

// 系统常量定义
const (
	// 服务名称
	ServiceName = "kaigate"
	// 服务版本
	ServiceVersion = "v1.0.0"

	// 默认监听地址
	DefaultHTTPAddr = ":8080"
	// 默认WebSocket监听地址
	DefaultWSAddr = ":8081"
	// 默认管理接口监听地址
	DefaultAdminAddr = ":8082"

	// 默认日志级别
	DefaultLogLevel = "info"
	// 默认日志格式
	DefaultLogFormat = "text"
	// 默认日志文件路径
	DefaultLogFile = "logs/kaigate.log"

	// 连接超时时间(秒)
	DefaultConnTimeout = 30
	// 读写超时时间(秒)
	DefaultRWTimeout = 60
	// WebSocket心跳间隔(秒)
	DefaultWSHeartbeatInterval = 30

	// 默认限流值(请求/秒)
	DefaultRateLimit = 100
	// 默认熔断阈值(错误率百分比)
	DefaultCircuitBreakThreshold = 50
)