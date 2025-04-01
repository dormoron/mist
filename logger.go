package mist

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel 定义日志级别类型
type LogLevel int

// 日志级别常量
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is an interface that specifies logging functionality.
// 扩展Logger接口，提供更多日志级别
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatalln(msg string, args ...any)
	WithField(key string, value any) Logger
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	SetOutput(w io.Writer)
}

// StdLogger 是Logger接口的标准实现
type StdLogger struct {
	mu     sync.Mutex
	out    io.Writer
	level  LogLevel
	fields map[string]any
}

// NewStdLogger 创建一个标准日志记录器
func NewStdLogger(level LogLevel, out io.Writer) *StdLogger {
	if out == nil {
		out = os.Stdout
	}
	return &StdLogger{
		out:    out,
		level:  level,
		fields: make(map[string]any),
	}
}

// formatLog 格式化日志消息
func (l *StdLogger) formatLog(level LogLevel, msg string, args ...any) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 获取调用者文件和行号
	_, file, line, ok := runtime.Caller(2)
	fileInfo := "???"
	if ok {
		fileInfo = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// 格式化字段
	fields := ""
	for k, v := range l.fields {
		fields += fmt.Sprintf(" %s=%v", k, v)
	}

	// 格式化参数
	logMsg := msg
	if len(args) > 0 {
		logMsg = fmt.Sprintf(msg, args...)
	}

	return fmt.Sprintf("[%s] [%s] [%s]%s %s\n", timestamp, level, fileInfo, fields, logMsg)
}

// log 执行实际的日志记录
func (l *StdLogger) log(level LogLevel, msg string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	logMsg := l.formatLog(level, msg, args...)
	_, _ = fmt.Fprint(l.out, logMsg)

	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug 记录调试级别日志
func (l *StdLogger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

// Info 记录信息级别日志
func (l *StdLogger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

// Warn 记录警告级别日志
func (l *StdLogger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

// Error 记录错误级别日志
func (l *StdLogger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

// Fatalln 记录致命级别日志，并终止程序
func (l *StdLogger) Fatalln(msg string, args ...any) {
	l.log(LevelFatal, msg, args...)
}

// WithField 返回带有附加字段的日志记录器
func (l *StdLogger) WithField(key string, value any) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 创建新的logger实例，复制当前配置
	newLogger := &StdLogger{
		out:    l.out,
		level:  l.level,
		fields: make(map[string]any, len(l.fields)+1),
	}

	// 复制现有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	newLogger.fields[key] = value

	return newLogger
}

// SetLevel 设置日志级别
func (l *StdLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *StdLogger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetOutput 设置日志输出目标
func (l *StdLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// 默认日志记录器
var defaultLogger Logger = NewStdLogger(LevelInfo, os.Stdout)

// SetDefaultLogger is a function that allows for the configuration of the
// application's default logging behavior by setting the provided logger
// as the new default logger.
//
// Parameters:
//
//	log Logger: This parameter is an implementation of the Logger interface.
//	            It represents the logger instance that the application should
//	            use as its new default logger. This logger will be used for
//	            all logging activities across the application, enabling a
//	            consistent logging approach.
//
// Purpose:
// The primary purpose of SetDefaultLogger is to provide a mechanism for
// changing the logging implementation used by an application at runtime.
// This is particularly useful in scenarios where the logging requirements
// change based on the environment the application is running in (e.g.,
// development, staging, production) or when integrating with different
// third-party logging services.
//
// Usage:
// To use SetDefaultLogger, an instance of a Logger implementation needs to
// be passed to it. This can be a custom logger tailored to the application's
// specific needs or an instance from a third-party logging library that
// adheres to the Logger interface. Once SetDefaultLogger is called with
// the new logger, all subsequent calls to the defaultLogger variable
// throughout the application will use this new logger instance,
// thereby affecting how logs are recorded and stored.
//
// Example:
// Suppose you have an application that uses a basic logging mechanism by
// default but requires integration with a more sophisticated logging
// system (like logrus or zap) for production environments. You can
// initialize the desired logger and pass it to SetDefaultLogger during
// the application's initialization phase. This ensures that all logging
// throughout the application uses the newly specified logger.
//
// Note:
// It is important to call SetDefaultLogger before any logging activity occurs
// to ensure that logs are consistently handled by the chosen logger. Failure
// to do so may result in some logs being handled by a different logger than
// intended, leading to inconsistency in log handling and potential loss of
// log data.
func SetDefaultLogger(log Logger) {
	defaultLogger = log
}

// GetDefaultLogger 返回当前默认的日志记录器
func GetDefaultLogger() Logger {
	return defaultLogger
}

// 全局日志方法
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	defaultLogger.Fatalln(msg, args...)
}

// WithField 向全局日志记录器添加字段
func WithField(key string, value any) Logger {
	return defaultLogger.WithField(key, value)
}

// 日志文件旋转配置
type LogRotateConfig struct {
	Filename   string
	MaxSize    int  // 单个文件的最大尺寸，MB
	MaxAge     int  // 保留日志的最大天数
	MaxBackups int  // 保留的最大旧日志文件数
	LocalTime  bool // 使用本地时间
	Compress   bool // 是否压缩旧日志
}

// 这里我们没有实现具体的日志轮转逻辑，在实际项目中可以使用第三方库如lumberjack
// 下面是一个使用接口的示例，用户可以根据需要实现自己的轮转逻辑

// LogRotator 日志文件轮转接口
type LogRotator interface {
	io.WriteCloser
}

// SetupLogRotation 设置日志轮转，这里只是示例，未实现具体逻辑
func SetupLogRotation(config LogRotateConfig) error {
	// 在实际实现中，可以使用如下代码：
	/*
		rotator := &lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,    // megabytes
			MaxAge:     config.MaxAge,     // days
			MaxBackups: config.MaxBackups,
			LocalTime:  config.LocalTime,
			Compress:   config.Compress,
		}

		// 设置默认logger使用rotator作为输出
		if l, ok := defaultLogger.(*StdLogger); ok {
			l.SetOutput(rotator)
		} else {
			defaultLogger.SetOutput(rotator)
		}
	*/

	// 由于依赖问题，这里不实现具体逻辑，返回nil
	return nil
}
