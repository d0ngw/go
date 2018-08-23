package common

import (
	"log"

	"go.uber.org/zap"

	"go.uber.org/zap/zapcore"
)

// StdLogger 使用标准库封装的logger
type StdLogger struct {
	logEnable zapcore.LevelEnabler
}

// NewStdLogger new info level logger
func NewStdLogger() *StdLogger {
	logger := &StdLogger{
		logEnable: zap.InfoLevel,
	}
	return logger
}

// Tracef trace
func (l *StdLogger) Tracef(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.DebugLevel) {
		return
	}
	log.Printf(format, params...)
}

// Debugf debug
func (l *StdLogger) Debugf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.DebugLevel) {
		return
	}
	log.Printf(format, params...)
}

// Infof info
func (l *StdLogger) Infof(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.InfoLevel) {
		return
	}
	log.Printf(format, params...)
}

// Warnf warn
func (l *StdLogger) Warnf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.WarnLevel) {
		return
	}
	log.Printf(format, params...)
}

// Errorf error
func (l *StdLogger) Errorf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.ErrorLevel) {
		return
	}
	log.Printf(format, params...)
}

// Criticalf critical
func (l *StdLogger) Criticalf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.ErrorLevel) {
		return
	}
	log.Printf(format, params...)
}

// Sync sync
func (l *StdLogger) Sync() {

}
