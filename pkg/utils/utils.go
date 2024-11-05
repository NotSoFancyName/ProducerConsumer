package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GetLoggerDefaultConfig(logLevel string) (zap.Config, error) {
	if len(logLevel) == 0 {
		logLevel = "info"
	}

	parsedLevel, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return zap.Config{}, err
	}

	return zap.Config{
		Level:       parsedLevel,
		Development: true,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}, nil
}
