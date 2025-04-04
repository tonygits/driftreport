package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestZapLogger(t *testing.T) {
	Convey("TestZapLogger", t, func() {
		Convey("Zap logger ", func() {
			logger, logs := setupLogsCapture()

			logger.Warn("This is the warning")

			if logs.Len() != 1 {
				t.Errorf("No logs")
			} else {
				entry := logs.All()[0]
				if entry.Level != zap.WarnLevel || entry.Message != "This is the warning" {
					t.Errorf("Invalid log entry %v", entry)
				}
			}
		})
	})
}

func setupLogsCapture() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	return zap.New(core), logs
}