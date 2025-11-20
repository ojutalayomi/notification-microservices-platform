package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level, format string) (*zap.Logger, error) {
	var config zap.Config

	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Set log level
	logLevel := zap.InfoLevel
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	config.Level = zap.NewAtomicLevelAt(logLevel)

	// Customize encoder config for better structure
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// Global logger instance for easy access
var globalLogger *zap.Logger

func InitGlobal(level, format string) error {
	logger, err := New(level, format)
	if err != nil {
		return err
	}
	globalLogger = logger
	zap.ReplaceGlobals(logger)
	return nil
}

func L() *zap.Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		logger, _ := zap.NewProduction()
		return logger
	}
	return globalLogger
}
