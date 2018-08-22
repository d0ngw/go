package common

import (
	"sync"
)

// Logger 日志记录接口
type Logger interface {
	// Tracef trace级别记录日志
	Tracef(format string, params ...interface{})

	// Debugf debug级别记录日志
	Debugf(format string, params ...interface{})

	// Infof info级别记录日志
	Infof(format string, params ...interface{})

	// Warnf warn级别记录日志
	Warnf(format string, params ...interface{})

	// Errorf error级别记录日志
	Errorf(format string, params ...interface{})

	// Errorf critical级别记录日志
	Criticalf(format string, params ...interface{})
}

// Tracef trace级别记录日志
func Tracef(format string, params ...interface{}) {
	logger.Tracef(format, params...)
}

// Debugf debug级别记录日志
func Debugf(format string, params ...interface{}) {
	logger.Debugf(format, params...)
}

// Infof info级别记录日志
func Infof(format string, params ...interface{}) {
	logger.Infof(format, params...)
}

// Warnf warn级别记录日志
func Warnf(format string, params ...interface{}) {
	logger.Warnf(format, params...)
}

// Errorf error级别记录日志
func Errorf(format string, params ...interface{}) {
	logger.Errorf(format, params...)
}

// Criticalf critical级别记录日志
func Criticalf(format string, params ...interface{}) {
	logger.Criticalf(format, params...)
}

var (
	logger Logger = &StdLogger{}
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
