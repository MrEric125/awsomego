package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel 日志级别
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level         LogLevel `json:"level"`
	Format        string   `json:"format"`      // json, console
	OutputPath    string   `json:"output_path"` // stdout, stderr, file path
	MaxSize       int      `json:"max_size"`    // MB
	MaxBackups    int      `json:"max_backups"`
	MaxAge        int      `json:"max_age"` // days
	Compress      bool     `json:"compress"`
	EnableCaller  bool     `json:"enable_caller"`
	EnableStack   bool     `json:"enable_stack"`
	ComponentName string   `json:"component_name"`
}

// DefaultLoggerConfig 默认日志配置
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:         InfoLevel,
		Format:        "console",
		OutputPath:    "stdout",
		MaxSize:       100,
		MaxBackups:    10,
		MaxAge:        30,
		Compress:      true,
		EnableCaller:  true,
		EnableStack:   true,
		ComponentName: "evaluation",
	}
}

// Logger 日志记录器
type Logger struct {
	zapLogger *zap.Logger
	sugar     *zap.SugaredLogger
	config    *LoggerConfig
	mu        sync.RWMutex
	fields    []zap.Field
}

var (
	globalLogger *Logger
	once         sync.Once
)

// InitLogger 初始化全局日志记录器
func InitLogger(cfg *LoggerConfig) error {
	var err error
	once.Do(func() {
		globalLogger, err = NewLogger(cfg)
	})
	return err
}

// NewLogger 创建新的日志记录器
func NewLogger(cfg *LoggerConfig) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultLoggerConfig()
	}

	// 解析日志级别
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 配置编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 选择编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 配置输出
	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else if cfg.OutputPath == "stderr" {
		writeSyncer = zapcore.AddSync(os.Stderr)
	} else {
		// 文件输出使用 lumberjack 进行轮转
		// 这里简化实现，实际应使用 lumberjack
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建logger选项
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	if cfg.EnableStack {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// 创建zap logger
	zapLogger := zap.New(core, options...)

	// 添加组件名称
	if cfg.ComponentName != "" {
		zapLogger = zapLogger.Named(cfg.ComponentName)
	}

	return &Logger{
		zapLogger: zapLogger,
		sugar:     zapLogger.Sugar(),
		config:    cfg,
		fields:    make([]zap.Field, 0),
	}, nil
}

// GetLogger 获取全局日志记录器
func GetLogger() *Logger {
	if globalLogger == nil {
		_ = InitLogger(DefaultLoggerConfig())
	}
	return globalLogger
}

// WithFields 添加字段
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 转换 zap.Field 为 interface{} 切片
	sugarFields := make([]interface{}, len(fields))
	for i, f := range fields {
		sugarFields[i] = f
	}

	newLogger := &Logger{
		zapLogger: l.zapLogger.With(fields...),
		sugar:     l.sugar.With(sugarFields...),
		config:    l.config,
		fields:    append(l.fields, fields...),
	}
	return newLogger
}

// WithContext 添加上下文
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// 从context中提取traceID等信息
	traceID := ctx.Value("trace_id")
	if traceID == nil {
		return l
	}
	return l.WithFields(zap.String("trace_id", traceID.(string)))
}

// WithComponent 添加组件名称
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithFields(zap.String("component", component))
}

// WithTask 添加任务ID
func (l *Logger) WithTask(taskID string) *Logger {
	return l.WithFields(zap.String("task_id", taskID))
}

// Debug 调试日志
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zapLogger.Debug(msg, fields...)
}

// Debugf 格式化调试日志
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Info 信息日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
}

// Infof 格式化信息日志
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Warn 警告日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
}

// Warnf 格式化警告日志
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Error 错误日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, fields...)
}

// Errorf 格式化错误日志
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zapLogger.Fatal(msg, fields...)
}

// Fatalf 格式化致命错误日志
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// Panic panic日志
func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zapLogger.Panic(msg, fields...)
}

// Panicf 格式化panic日志
func (l *Logger) Panicf(template string, args ...interface{}) {
	l.sugar.Panicf(template, args...)
}

// LogError 记录错误
func (l *Logger) LogError(err error, msg string, fields ...zap.Field) {
	allFields := append(fields, zap.Error(err))
	l.Error(msg, allFields...)
}

// LogOperation 记录操作
func (l *Logger) LogOperation(operation, component string, success bool, duration interface{}, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("operation", operation),
		zap.String("component", component),
		zap.Bool("success", success),
		zap.Any("duration", duration),
	)

	if success {
		l.Info("Operation completed", allFields...)
	} else {
		l.Error("Operation failed", allFields...)
	}
}

// Sync 同步日志
func (l *Logger) Sync() error {
	return l.zapLogger.Sync()
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	return l.Sync()
}

// 全局日志函数

// Debug 全局调试日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Debugf 全局格式化调试日志
func Debugf(template string, args ...interface{}) {
	GetLogger().Debugf(template, args...)
}

// Info 全局信息日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Infof 全局格式化信息日志
func Infof(template string, args ...interface{}) {
	GetLogger().Infof(template, args...)
}

// Warn 全局警告日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Warnf 全局格式化警告日志
func Warnf(template string, args ...interface{}) {
	GetLogger().Warnf(template, args...)
}

// Error 全局错误日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Errorf 全局格式化错误日志
func Errorf(template string, args ...interface{}) {
	GetLogger().Errorf(template, args...)
}

// Fatal 全局致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Fatalf 全局格式化致命错误日志
func Fatalf(template string, args ...interface{}) {
	GetLogger().Fatalf(template, args...)
}
