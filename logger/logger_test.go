package logger_test

import (
	"testing"

	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/logger"
)

func TestNewLogger(t *testing.T) {
	log := logger.NewLogger(config.LoggerConfig{Level: "info", Format: "json"})
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	// Should not panic
	log.Info("test message")
}

func TestErr(t *testing.T) {
	attr := logger.Err(nil)
	if attr.Value.String() != "" {
		t.Errorf("want empty, got %s", attr.Value.String())
	}
}
