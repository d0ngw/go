package common

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ZapLogger 使用zap封装的logger
type ZapLogger struct {
	logEnable zap.AtomicLevel
	logger    *zap.SugaredLogger
}

// Debugf debug
func (l *ZapLogger) Debugf(format string, params ...interface{}) {
	l.logger.Debugf(format, params...)
}

// DebugEnabled is debug enbale
func (l *ZapLogger) DebugEnabled() bool {
	return l.logEnable.Enabled(zap.DebugLevel)
}

// Infof info
func (l *ZapLogger) Infof(format string, params ...interface{}) {
	l.logger.Infof(format, params...)
}

// InfoEnabled is info enbale
func (l *ZapLogger) InfoEnabled() bool {
	return l.logEnable.Enabled(zap.InfoLevel)
}

// Warnf warn
func (l *ZapLogger) Warnf(format string, params ...interface{}) {
	l.logger.Warnf(format, params...)
}

// WarnEnabled is info enbale
func (l *ZapLogger) WarnEnabled() bool {
	return l.logEnable.Enabled(zap.WarnLevel)
}

// Errorf error
func (l *ZapLogger) Errorf(format string, params ...interface{}) {
	l.logger.Errorf(format, params...)
}

// ErrorEnabled is info enbale
func (l *ZapLogger) ErrorEnabled() bool {
	return l.logEnable.Enabled(zap.ErrorLevel)
}

// Sync impls Logger.Sync
func (l *ZapLogger) Sync() {
	l.logger.Sync()
}

// SetLevel set the log level
func (l *ZapLogger) SetLevel(level LogLevel) {
	zapl, ok := level.zapLevel()
	if ok {
		l.logEnable.SetLevel(zapl)
	}
}

// NewZapLogger new zap logger
func NewZapLogger(logConfig *LogConfig) *ZapLogger {
	var encoder zapcore.Encoder
	var writerSync zapcore.WriteSyncer
	var logEnable zap.AtomicLevel

	if logConfig.Env == EnvProduction {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(config)
		logEnable = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else {
		config := zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(config)
		logEnable = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logConfig.Level != "" {
		zapl, ok := LogLevel(logConfig.Level).zapLevel()
		if ok {
			logEnable = zap.NewAtomicLevelAt(zapl)
		}
	}

	if logConfig.FileName != "" {
		writerSync = zapcore.AddSync(&lumberjack.Logger{
			Filename:   logConfig.FileName,
			MaxSize:    logConfig.MaxSize,
			MaxBackups: logConfig.MaxBackups,
			MaxAge:     logConfig.MaxAge,
			LocalTime:  true,
		})
	} else {
		writerSync = zapcore.AddSync(os.Stderr)
	}

	core := zapcore.NewCore(encoder, writerSync, logEnable)
	logger := zap.New(core)
	if !logConfig.NoCaller {
		logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))
	}
	sugarLogger := logger.Sugar()
	return &ZapLogger{logger: sugarLogger, logEnable: logEnable}
}
