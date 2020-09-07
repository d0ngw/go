package common

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// LogLevel log level
type LogLevel string

// log levels
const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func (p LogLevel) zapLevel() (level zapcore.Level, ok bool) {
	switch p {
	case Debug:
		level = zapcore.DebugLevel
		ok = true
	case Info:
		level = zapcore.InfoLevel
		ok = true
	case Warn:
		level = zapcore.WarnLevel
		ok = true
	case Error:
		level = zapcore.ErrorLevel
		ok = true
	}
	return
}

// Logger 日志记录接口
type Logger interface {
	Debugf(format string, params ...interface{})
	DebugEnabled() bool
	Infof(format string, params ...interface{})
	InfoEnabled() bool
	Warnf(format string, params ...interface{})
	WarnEnabled() bool
	Errorf(format string, params ...interface{})
	ErrorEnabled() bool
	SetLevel(level LogLevel)
	Sync()
}

// Debugf debug级别记录日志
func Debugf(format string, params ...interface{}) {
	logger.Debugf(format, params...)
}

// DebugEnabled debug
func DebugEnabled() bool {
	return logger.DebugEnabled()
}

// Infof info级别记录日志
func Infof(format string, params ...interface{}) {
	logger.Infof(format, params...)
}

// InfoEnabled info
func InfoEnabled() bool {
	return logger.InfoEnabled()
}

// Warnf warn级别记录日志
func Warnf(format string, params ...interface{}) {
	logger.Warnf(format, params...)
}

// WarnEnabled warn
func WarnEnabled() bool {
	return logger.WarnEnabled()
}

// Errorf error级别记录日志
func Errorf(format string, params ...interface{}) {
	logger.Errorf(format, params...)
}

// ErrorEnabled error
func ErrorEnabled() bool {
	return logger.ErrorEnabled()
}

// SetLogLevel set the log level
func SetLogLevel(level LogLevel) {
	logger.SetLevel(level)
}

// LoggerSync sync log
func LoggerSync() {
	logger.Sync()
}

var (
	logger Logger = NewStdLogger()
)

var m sync.Mutex
var loggerInitd bool

// initLogger 初始化logger
func initLogger(logConfig *LogConfig) (err error) {
	m.Lock()
	defer m.Unlock()

	if loggerInitd {
		Errorf("Logger has been already inited.")
		return
	}

	zapLogger := NewZapLogger(logConfig)
	logger = zapLogger
	return
}
