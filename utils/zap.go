package utils

import (
	"os"
	"testing"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitZapLog() *zap.Logger {
	var coreArr []zapcore.Core
	// Configure the logger
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	if !testing.Testing() {
		// Log levels
		highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { // Error level
			return lev >= zap.ErrorLevel
		})
		lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { // Info and debug levels, debug level is the lowest
			return lev < zap.ErrorLevel && lev >= zap.DebugLevel
		})

		// Info file writeSyncer
		infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "./log/info.log", // Log file storage directory. If the folder does not exist, it will be created automatically.
			MaxSize:    1,                // File size limit, unit MB
			MaxBackups: 5,                // Maximum number of retained log files
			MaxAge:     30,               // Number of days to retain log files
			Compress:   false,            // Whether to compress
		})
		infoFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), lowPriority) // The third and subsequent parameters are the log levels for writing to the file. In ErrorLevel mode, only error - level logs are recorded.

		// Error file writeSyncer
		errorFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "./log/error.log", // Log file storage directory
			MaxSize:    1,                 // File size limit, unit MB
			MaxBackups: 5,                 // Maximum number of retained log files
			MaxAge:     30,                // Number of days to retain log files
			Compress:   false,             // Whether to compress
		})
		errorFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(errorFileWriteSyncer, zapcore.AddSync(os.Stdout)), highPriority) // The third and subsequent parameters are the log levels for writing to the file. In ErrorLevel mode, only error - level logs are recorded.
		coreArr = append(coreArr, infoFileCore)
		coreArr = append(coreArr, errorFileCore)
	} else {
		// In test mode, only log to stdout
		coreArr = append(coreArr, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.DebugLevel))
	}
	// Return the logger
	Logger = zap.New(zapcore.NewTee(coreArr...), zap.AddCaller())
	return Logger
}
