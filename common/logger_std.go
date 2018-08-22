package common

import (
	"log"
)

// StdLogger 使用标准库封装的logger
type StdLogger struct {
}

// Tracef trace
func (l *StdLogger) Tracef(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Debugf debug
func (l *StdLogger) Debugf(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Infof info
func (l *StdLogger) Infof(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Warnf warn
func (l *StdLogger) Warnf(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Errorf error
func (l *StdLogger) Errorf(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Criticalf critical
func (l *StdLogger) Criticalf(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Sync sync
func (l *StdLogger) Sync() {

}
