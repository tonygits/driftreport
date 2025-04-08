package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitZapLog() *zap.Logger {
	// Configure the logger
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// Create a core
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),                // Use Console encoder
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), // Output to console
		zap.InfoLevel,                                           // Set log level
	)

	// Return the logger
	Logger = zap.New(core, zap.AddCaller())
	return Logger
}
