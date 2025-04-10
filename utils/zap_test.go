package utils

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestErrorLogging(t *testing.T) {
	// initialize core logger
	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
		os.Stderr,
		zapcore.DebugLevel,
	)

	Convey("can handle error level logs", t, func() {
		// test core
		observed, logs := observer.New(zapcore.DebugLevel)

		// new logger test
		logger := zap.New(zapcore.NewTee(zapCore, observed))
		logger.Error("zap logger test")

		entry := logs.All()[0]
		So(entry.Message, ShouldEqual, "zap logger test")
		So(entry.Level, ShouldEqual, zapcore.ErrorLevel)
	})

	Convey("must print the logged message", t, func() {
		// test core
		observed, logs := observer.New(zapcore.InfoLevel)

		// new logger test
		logger := zap.New(zapcore.NewTee(zapCore, observed))
		logger.Info("new logger test")

		entry := logs.All()[0]
		So(entry.Message, ShouldNotEqual, "zap logger test")
		So(entry.Message, ShouldEqual, "new logger test")
		So(entry.Level, ShouldNotEqual, zapcore.ErrorLevel)
		So(entry.Level, ShouldEqual, zapcore.InfoLevel)
	})
}
