package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	config.Encoding = "console"
	config.EncoderConfig.TimeKey = ""
	config.EncoderConfig.LevelKey = ""
	config.EncoderConfig.CallerKey = ""
	config.EncoderConfig.NameKey = ""
	config.OutputPaths = []string{"stdout"}
	config.DisableStacktrace = true

	log, err := config.Build()
	if err != nil {
		panic(err)
	}

	Log = log.Sugar()
}

func UseDebugLogger() {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006.01.02 15:04:05.000000")
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	config.Encoding = "console"
	config.OutputPaths = []string{"stdout"}
	config.DisableStacktrace = true

	log, err := config.Build()
	if err != nil {
		panic(err)
	}

	Log = log.Sugar()
}
