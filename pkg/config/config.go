package config

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 定义系统配置结构
type Config struct {
	// 服务配置
	Server struct {
		HTTPAddr      string `yaml:"http_addr"`
		WSAddr        string `yaml:"ws_addr"`
		AdminAddr     string `yaml:"admin_addr"`
		Debug         bool   `yaml:"debug"`
		ConnTimeout   int    `yaml:"conn_timeout"`
		RWTimeout     int    `yaml:"rw_timeout"`
	} `yaml:"server"`

	// 日志配置
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		File   string `yaml:"file"`
		Stdout bool   `yaml:"stdout"`
	} `yaml:"log"`

	// WebSocket配置
	WebSocket struct {
		HeartbeatInterval int `yaml:"heartbeat_interval"`
		MaxConnections    int `yaml:"max_connections"`
	} `yaml:"websocket"`

	// 路由配置
	Router struct {
		EnableRateLimit       bool `yaml:"enable_rate_limit"`
		DefaultRateLimit      int  `yaml:"default_rate_limit"`
		CircuitBreak          bool `yaml:"circuit_break"`
		CircuitBreakThreshold int  `yaml:"circuit_break_threshold"`
	} `yaml:"router"`

	// 代理路由配置
	ProxyRoutes []struct {
		Path       string `yaml:"path"`       // 代理路径
		TargetURL  string `yaml:"target_url"` // 目标URL
		Enable     bool   `yaml:"enable"`     // 是否启用
	} `yaml:"proxy_routes"`
}

// GlobalConfig 全局配置实例
var GlobalConfig Config

// configMutex 保护全局配置的互斥锁
var configMutex sync.RWMutex

// configFile 保存当前使用的配置文件路径
var configFile string

// InitConfig 初始化配置
func InitConfig(file string) error {
	// 初始化默认值
	initDefaultConfig()

	// 从配置文件加载配置
	if file != "" {
		if err := loadFromFile(file); err != nil {
			return fmt.Errorf("load config file failed: %w", err)
		}
	}

	// 从命令行参数覆盖配置
	loadFromCmdLine()

	// 保存配置文件路径
	configFile = file

	return nil
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	// 检查是否有配置文件
	if configFile == "" {
		return fmt.Errorf("no config file specified")
	}

	// 创建新的配置实例，避免直接修改正在使用的配置
	newConfig := Config{}

	// 初始化默认值
	initDefaultConfigFor(&newConfig)

	// 从配置文件加载配置
	if err := loadFromFileFor(configFile, &newConfig); err != nil {
		return fmt.Errorf("reload config file failed: %w", err)
	}

	// 从命令行参数覆盖配置（保持与初始化时一致）
	loadFromCmdLineFor(&newConfig)

	// 使用互斥锁保护配置更新
	configMutex.Lock()
	GlobalConfig = newConfig
	configMutex.Unlock()

	return nil
}

// GetConfig 获取当前配置（线程安全）
func GetConfig() Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return GlobalConfig
}

// initDefaultConfig 初始化默认配置值
func initDefaultConfig() {
	initDefaultConfigFor(&GlobalConfig)
}

// initDefaultConfigFor 为指定配置实例初始化默认值
func initDefaultConfigFor(config *Config) {
	// 服务配置
	config.Server.HTTPAddr = DefaultHTTPAddr
	config.Server.WSAddr = DefaultWSAddr
	config.Server.AdminAddr = DefaultAdminAddr
	config.Server.Debug = false
	config.Server.ConnTimeout = DefaultConnTimeout
	config.Server.RWTimeout = DefaultRWTimeout

	// 日志配置
	config.Log.Level = DefaultLogLevel
	config.Log.Format = DefaultLogFormat
	config.Log.File = DefaultLogFile
	config.Log.Stdout = true

	// WebSocket配置
	config.WebSocket.HeartbeatInterval = DefaultWSHeartbeatInterval
	config.WebSocket.MaxConnections = 1000

	// 路由配置
	config.Router.EnableRateLimit = true
	config.Router.DefaultRateLimit = DefaultRateLimit
	config.Router.CircuitBreak = true
	config.Router.CircuitBreakThreshold = DefaultCircuitBreakThreshold
}

// loadFromFile 从配置文件加载配置
func loadFromFile(configFile string) error {
	return loadFromFileFor(configFile, &GlobalConfig)
}

// loadFromFileFor 从配置文件加载配置到指定实例
func loadFromFileFor(configFile string, config *Config) error {
	// 检查文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not exists: %s", configFile)
	}

	// 读取配置文件内容
	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file failed: %w", err)
	}

	// 解析YAML配置
	if err := yaml.Unmarshal(content, config); err != nil {
		return fmt.Errorf("parse config file failed: %w", err)
	}

	return nil
}

// loadFromCmdLine 从命令行参数加载配置
func loadFromCmdLine() {
	loadFromCmdLineFor(&GlobalConfig)
}

// loadFromCmdLineFor 从命令行参数加载配置到指定实例
func loadFromCmdLineFor(config *Config) {
	// 定义命令行参数
	debug := flag.Bool("debug", config.Server.Debug, "Enable debug mode")
	httpAddr := flag.String("http-addr", config.Server.HTTPAddr, "HTTP server listen address")
	wsAddr := flag.String("ws-addr", config.Server.WSAddr, "WebSocket server listen address")
	adminAddr := flag.String("admin-addr", config.Server.AdminAddr, "Admin interface listen address")
	logLevel := flag.String("log-level", config.Log.Level, "Log level")
	logFormat := flag.String("log-format", config.Log.Format, "Log format")
	logFile := flag.String("log-file", config.Log.File, "Log file path")

	// 解析命令行参数
	flag.Parse()

	// 更新配置
	config.Server.Debug = *debug
	config.Server.HTTPAddr = *httpAddr
	config.Server.WSAddr = *wsAddr
	config.Server.AdminAddr = *adminAddr
	config.Log.Level = *logLevel
	config.Log.Format = *logFormat
	config.Log.File = *logFile
}