package log

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 定义日志接口
type Logger interface {
	// Debug 记录调试日志
	Debug(msg string, fields ...zapcore.Field)
	// Info 记录信息日志
	Info(msg string, fields ...zapcore.Field)
	// Warn 记录警告日志
	Warn(msg string, fields ...zapcore.Field)
	// Error 记录错误日志
	Error(msg string, fields ...zapcore.Field)
	// Panic 记录恐慌日志并触发panic
	Panic(msg string, fields ...zapcore.Field)
	// Fatal 记录致命错误并退出程序
	Fatal(msg string, fields ...zapcore.Field)
	// Named 创建一个带名称的日志器
	Named(name string) Logger
	// With 添加固定字段
	With(fields ...zapcore.Field) Logger
	// Access 记录访问日志
	Access(reqPath string, method string, status int, latencyMs int64, remoteAddr string, fields ...zapcore.Field)
	// Audit 记录审计日志
	Audit(action string, operator string, resource string, success bool, fields ...zapcore.Field)
	// ErrorWithStack 记录带堆栈的错误日志
	ErrorWithStack(err error, msg string, fields ...zapcore.Field)
}

// GlobalLogger 全局日志器
var GlobalLogger Logger

// DefaultLogger 默认日志器实现
type DefaultLogger struct {
	logger *zap.Logger
}

// 全局访问日志器和审计日志器
var accessLogger *zap.Logger
var auditLogger *zap.Logger

// InitLogger 初始化日志器
func InitLogger(level, format, filePath string, enableStdout bool) error {
	// 日志级别映射
	levelMap := map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"warn":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"panic": zapcore.PanicLevel,
		"fatal": zapcore.FatalLevel,
	}

	// 获取日志级别
	logLevel, ok := levelMap[strings.ToLower(level)]
	if !ok {
		logLevel = zapcore.InfoLevel
	}

	// 创建编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.MessageKey = "message"

	var encoder zapcore.Encoder
	if format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建写入器
	writers := []zapcore.WriteSyncer{}

	// 如果配置了文件路径，添加文件写入器
	if filePath != "" {
		// 确保目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// 创建lumberjack写入器，支持日志轮转
		writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    100, // MB
			MaxAge:     7,   // days
			MaxBackups: 5,
			Compress:   true,
		}))
	}

	// 如果启用了标准输出，添加标准输出写入器
	if enableStdout {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 如果没有写入器，使用标准输出作为默认写入器
	if len(writers) == 0 {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 创建core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		zap.NewAtomicLevelAt(logLevel),
	)

	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// 设置全局日志器
	GlobalLogger = &DefaultLogger{logger}

	// 创建访问日志器和审计日志器
	accessLogger = logger.Named("access")
	auditLogger = logger.Named("audit")

	return nil
}

// 创建一个新的日志器实例
func createLogger(level, format, filePath string, enableStdout bool, loggerType string) (Logger, error) {
	// 日志级别映射
	levelMap := map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"warn":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"panic": zapcore.PanicLevel,
		"fatal": zapcore.FatalLevel,
	}

	// 获取日志级别
	logLevel, ok := levelMap[strings.ToLower(level)]
	if !ok {
		logLevel = zapcore.InfoLevel
	}

	// 创建编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.MessageKey = "message"

	var encoder zapcore.Encoder
	if format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建写入器
	writers := []zapcore.WriteSyncer{}

	// 如果配置了文件路径，添加文件写入器
	if filePath != "" {
		// 确保目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		// 创建lumberjack写入器，支持日志轮转
		writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    100, // MB
			MaxAge:     7,   // days
			MaxBackups: 5,
			Compress:   true,
		}))
	}

	// 如果启用了标准输出，添加标准输出写入器
	if enableStdout {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 如果没有写入器，使用标准输出作为默认写入器
	if len(writers) == 0 {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 创建core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		zap.NewAtomicLevelAt(logLevel),
	)

	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// 添加logger类型
	if loggerType != "" {
		logger = logger.Named(loggerType)
	}

	return &DefaultLogger{logger}, nil
}

// Access 记录访问日志
func (l *DefaultLogger) Access(reqPath string, method string, status int, latencyMs int64, remoteAddr string, fields ...zapcore.Field) {
	accessLogger.Info("HTTP Request",
		append(
			[]zapcore.Field{
				zap.String("path", reqPath),
				zap.String("method", method),
				zap.Int("status", status),
				zap.Int64("latency_ms", latencyMs),
				zap.String("remote_addr", remoteAddr),
			}, fields..., 
		)...,
	)
}

// ErrorWithStack 记录带堆栈的错误日志
func (l *DefaultLogger) ErrorWithStack(err error, msg string, fields ...zapcore.Field) {
	fields = append(fields, zap.Error(err))
	l.Error(msg, fields...)
}

// Audit 记录审计日志
func (l *DefaultLogger) Audit(action string, operator string, resource string, success bool, fields ...zapcore.Field) {
	auditLogger.Info("Audit Log",
		append(
			[]zapcore.Field{
				zap.String("action", action),
				zap.String("operator", operator),
				zap.String("resource", resource),
				zap.Bool("success", success),
			}, fields..., 
		)...,
	)
}

// Debug 记录调试日志
func (l *DefaultLogger) Debug(msg string, fields ...zapcore.Field) {
	l.logger.Debug(msg, fields...)
}

// Info 记录信息日志
func (l *DefaultLogger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Warn 记录警告日志
func (l *DefaultLogger) Warn(msg string, fields ...zapcore.Field) {
	l.logger.Warn(msg, fields...)
}

// Error 记录错误日志
func (l *DefaultLogger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}

// Panic 记录恐慌日志并触发panic
func (l *DefaultLogger) Panic(msg string, fields ...zapcore.Field) {
	l.logger.Panic(msg, fields...)
}

// Fatal 记录致命错误并退出程序
func (l *DefaultLogger) Fatal(msg string, fields ...zapcore.Field) {
	l.logger.Fatal(msg, fields...)
}

// Named 创建一个带名称的日志器
func (l *DefaultLogger) Named(name string) Logger {
	return &DefaultLogger{l.logger.Named(name)}
}

// With 添加固定字段
func (l *DefaultLogger) With(fields ...zapcore.Field) Logger {
	return &DefaultLogger{l.logger.With(fields...)}
}