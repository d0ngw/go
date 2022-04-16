package common

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// LogLevel log level
type LogLevel int8

// log levels
const (
	Debug LogLevel = LogLevel(zapcore.DebugLevel)
	Info  LogLevel = LogLevel(zapcore.InfoLevel)
	Warn  LogLevel = LogLevel(zapcore.WarnLevel)
	Error LogLevel = LogLevel(zapcore.ErrorLevel)
)

// LogLevelName LogLevel
func LogLevelName(name string) LogLevel {
	switch name {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	}
	return Info
}

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

// Logf log
func Logf(level LogLevel, foramt string, params ...interface{}) {
	if level == Debug {
		logger.Debugf(foramt, params...)
	} else if level == Info {
		logger.Infof(foramt, params...)
	} else if level == Warn {
		logger.Warnf(foramt, params...)
	} else if level == Error {
		logger.Errorf(foramt, params...)
	} else {
		logger.Debugf(foramt, params...)
	}
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
