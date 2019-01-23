package common

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ZapLogger 使用zap封装的logger
type ZapLogger struct {
	logger *zap.SugaredLogger
}

// Tracef trace
func (l *ZapLogger) Tracef(format string, params ...interface{}) {
	l.logger.Debugf(format, params...)
}

// Debugf debug
func (l *ZapLogger) Debugf(format string, params ...interface{}) {
	l.logger.Debugf(format, params...)
}

// Infof info
func (l *ZapLogger) Infof(format string, params ...interface{}) {
	l.logger.Infof(format, params...)
}

// Warnf warn
func (l *ZapLogger) Warnf(format string, params ...interface{}) {
	l.logger.Warnf(format, params...)
}

// Errorf error
func (l *ZapLogger) Errorf(format string, params ...interface{}) {
	l.logger.Errorf(format, params...)
}

// Criticalf critical
func (l *ZapLogger) Criticalf(format string, params ...interface{}) {
	l.logger.Errorf(format, params...)
}

// Sync impls Logger.Sync
func (l *ZapLogger) Sync() {
	l.logger.Sync()
}

// NewZapLogger new zap logger
func NewZapLogger(logConfig *LogConfig) *ZapLogger {
	var encoder zapcore.Encoder
	var writerSync zapcore.WriteSyncer
	var logEnable zapcore.LevelEnabler
	if logConfig.Env == EnvProduction {
		config := zap.NewProductionEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(config)
		logEnable = zap.InfoLevel
	} else {
		config := zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(config)
		logEnable = zap.DebugLevel
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
		writerSync = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writerSync, logEnable)
	logger := zap.New(core)
	logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))
	sugarLogger := logger.Sugar()
	return &ZapLogger{logger: sugarLogger}
}
