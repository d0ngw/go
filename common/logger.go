package common

import (
	"fmt"
	"log"
	"os"
	"sync"

	seelog "github.com/cihub/seelog"
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

// SeeLogLogger 使用seelog封装的logger
type SeeLogLogger struct {
	seelogger seelog.LoggerInterface
}

// Tracef seelog trace级别记录日志
func (l *SeeLogLogger) Tracef(format string, params ...interface{}) {
	l.seelogger.Tracef(format, params...)
}

// Debugf seelog debug级别记录日志
func (l *SeeLogLogger) Debugf(format string, params ...interface{}) {
	l.seelogger.Debugf(format, params...)
}

// Infof seelog info级别记录日志
func (l *SeeLogLogger) Infof(format string, params ...interface{}) {
	l.seelogger.Infof(format, params...)
}

// Warnf seelog warn级别记录日志
func (l *SeeLogLogger) Warnf(format string, params ...interface{}) {
	l.seelogger.Warnf(format, params...)
}

// Errorf seelog error级别记录日志
func (l *SeeLogLogger) Errorf(format string, params ...interface{}) {
	l.seelogger.Errorf(format, params...)
}

// Criticalf seelog critical级别记录日志
func (l *SeeLogLogger) Criticalf(format string, params ...interface{}) {
	l.seelogger.Criticalf(format, params...)
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
	logger Logger = &SeeLogLogger{nil}
)

//默认的配置
const defaultConfig = `
    <seelog minlevel="info" type="sync">
        <outputs formatid="common">
			 <console/>
        </outputs>
        <formats>
		<format id="common" format="%Date/%Time [%LEV] %RelFile:%Line %Msg%n" />
        </formats>
    </seelog>
	`

// DefaultDebugLoggConfig 默认的debug
const DefaultDebugLoggConfig = `
    <seelog minlevel="debug" type="sync">
        <outputs formatid="common">
			 <console/>
        </outputs>
        <formats>
		<format id="common" format="%Date/%Time [%LEV] %RelFile:%Line %Msg%n" />
        </formats>
    </seelog>
    `

func init() {
	var seelogger seelog.LoggerInterface
	var err error

	seelogger, err = seelog.LoggerFromConfigAsBytes([]byte(defaultConfig))
	if err != nil {
		log.Panicf("Can't init Logger,error:%s", err)
		return
	}
	//确保取得正确的调用堆栈
	seelogger.SetAdditionalStackDepth(2)
	logger = &SeeLogLogger{seelogger}
}

var m sync.Mutex
var loggerInitd bool

// InitLogger 从配置文件configFile初始化logger
func InitLogger(configFile string) {
	fmt.Fprintln(os.Stderr, "Use "+configFile+" init Logger")
	initLogger(configFile, seelog.LoggerFromConfigAsFile)
}

// InitLoggerFromString 从config配置初始化logger
func InitLoggerFromString(config string) {
	initLogger(config, seelog.LoggerFromConfigAsString)
}

// initLogger 初始化logger
func initLogger(config string, loader func(conf string) (seelog.LoggerInterface, error)) {
	m.Lock()
	defer m.Unlock()

	if loggerInitd {
		Errorf("Logger has been already inited.")
		return
	}

	seelogger, err := loader(config)

	if err != nil {
		log.Panicf("Can't init Logger,error:%s", err)
		return
	}

	//确保取得正确的调用堆栈
	seelogger.SetAdditionalStackDepth(2)
	realLogger := logger.(*SeeLogLogger)
	if realLogger.seelogger != nil {
		realLogger.seelogger.Flush()
		realLogger.seelogger.Close()
	}
	realLogger.seelogger = seelogger
	loggerInitd = true
}
