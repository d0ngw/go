package common

import (
	"fmt"
	seelog "github.com/cihub/seelog"
	"log"
	"os"
	"sync"
)

//日志,目前提供对seelog的封装

type ILogger interface {
	Tracef(format string, params ...interface{})

	Debugf(format string, params ...interface{})

	Infof(format string, params ...interface{})

	Warnf(format string, params ...interface{})

	Errorf(format string, params ...interface{})

	Criticalf(format string, params ...interface{})
}

//使用seelog封装的logger
type SeeLogLogger struct {
	seelogger seelog.LoggerInterface
}

func (l *SeeLogLogger) Tracef(format string, params ...interface{}) {
	l.seelogger.Tracef(format, params...)
}

func (l *SeeLogLogger) Debugf(format string, params ...interface{}) {
	l.seelogger.Debugf(format, params...)
}

func (l *SeeLogLogger) Infof(format string, params ...interface{}) {
	l.seelogger.Infof(format, params...)
}

func (l *SeeLogLogger) Warnf(format string, params ...interface{}) {
	l.seelogger.Warnf(format, params...)
}

func (l *SeeLogLogger) Errorf(format string, params ...interface{}) {
	l.seelogger.Errorf(format, params...)
}

func (l *SeeLogLogger) Criticalf(format string, params ...interface{}) {
	l.seelogger.Criticalf(format, params...)
}

//默认的记录日志的函数
func Tracef(format string, params ...interface{}) {
	logger.Tracef(format, params...)
}

func Debugf(format string, params ...interface{}) {
	logger.Debugf(format, params...)
}

func Infof(format string, params ...interface{}) {
	logger.Infof(format, params...)
}

func Warnf(format string, params ...interface{}) {
	logger.Warnf(format, params...)
}

func Errorf(format string, params ...interface{}) {
	logger.Errorf(format, params...)
}

func Criticalf(format string, params ...interface{}) {
	logger.Criticalf(format, params...)
}

//全局Logger
var logger ILogger = &SeeLogLogger{nil}
var configFile string

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

func init() {
	var seelogger seelog.LoggerInterface
	var err error

	seelogger, err = seelog.LoggerFromConfigAsBytes([]byte(defaultConfig))

	if err != nil {
		log.Panicf("Can't init Logger,error:%s", err)
		return
	} else {
		//确保取得正确的调用堆栈
		seelogger.SetAdditionalStackDepth(2)
	}
	logger = &SeeLogLogger{seelogger}
}

var m sync.Mutex
var loggerInitd bool

func InitLogger(configFile string) {
	fmt.Fprintln(os.Stderr, "Use "+configFile+" init Logger")
	m.Lock()
	defer m.Unlock()

	if loggerInitd {
		Errorf("Logger has been already inited.")
		return
	}

	seelogger, err := seelog.LoggerFromConfigAsFile(configFile)

	if err != nil {
		log.Panicf("Can't init Logger,error:%s", err)
		return
	} else {
		//确保取得正确的调用堆栈
		seelogger.SetAdditionalStackDepth(2)
	}

	realLogger := logger.(*SeeLogLogger)
	if realLogger.seelogger != nil {
		realLogger.seelogger.Flush()
		realLogger.seelogger.Close()
	}
	realLogger.seelogger = seelogger
	loggerInitd = true
}
