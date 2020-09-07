package common

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// StdLogger 使用标准库封装的logger
type StdLogger struct {
	logEnable zap.AtomicLevel
}

// NewStdLogger new info level logger
func NewStdLogger() *StdLogger {
	logger := &StdLogger{
		logEnable: zap.NewAtomicLevelAt(zapcore.InfoLevel),
	}
	return logger
}

// Debugf debug
func (l *StdLogger) Debugf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.DebugLevel) {
		return
	}
	log.Printf(format, params...)
}

// DebugEnabled is debug enbale
func (l *StdLogger) DebugEnabled() bool {
	return l.logEnable.Enabled(zap.DebugLevel)
}

// Infof info
func (l *StdLogger) Infof(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.InfoLevel) {
		return
	}
	log.Printf(format, params...)
}

// InfoEnabled is info enable
func (l *StdLogger) InfoEnabled() bool {
	return l.logEnable.Enabled(zap.InfoLevel)
}

// Warnf warn
func (l *StdLogger) Warnf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.WarnLevel) {
		return
	}
	log.Printf(format, params...)
}

// WarnEnabled is  warn enabled
func (l *StdLogger) WarnEnabled() bool {
	return l.logEnable.Enabled(zap.WarnLevel)
}

// Errorf error
func (l *StdLogger) Errorf(format string, params ...interface{}) {
	if !l.logEnable.Enabled(zap.ErrorLevel) {
		return
	}
	log.Printf(format, params...)
}

// ErrorEnabled error
func (l *StdLogger) ErrorEnabled() bool {
	return l.logEnable.Enabled(zap.ErrorLevel)
}

// Sync sync
func (l *StdLogger) Sync() {

}

// SetLevel set the log level
func (l *StdLogger) SetLevel(level LogLevel) {
	zapl, ok := level.zapLevel()
	if ok {
		l.logEnable.SetLevel(zapl)
	}
}
