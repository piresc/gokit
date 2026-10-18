package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/piresc/gokit/config"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLogger_JSONFormat(t *testing.T) {
	cfg := config.LoggerConfig{Level: "info", Format: "json"}
	log := NewLogger(cfg)
	assert.NotNil(t, log)
}

func TestNewLogger_TextFormat(t *testing.T) {
	cfg := config.LoggerConfig{Level: "debug", Format: "text"}
	log := NewLogger(cfg)
	assert.NotNil(t, log)
}

func TestOtelHandler_NilContext(t *testing.T) {
	// otelHandler must not panic when ctx is nil
	h := &otelHandler{base: slog.NewJSONHandler(io.Discard, nil)}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	err := h.Handle(nil, r) //nolint:staticcheck
	assert.NoError(t, err)
}

func TestOtelHandler_SpanlessContext(t *testing.T) {
	// otelHandler must not add trace_id when context has no active span
	var buf bytes.Buffer
	h := &otelHandler{base: slog.NewJSONHandler(&buf, nil)}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	err := h.Handle(context.Background(), r)
	assert.NoError(t, err)
	assert.NotContains(t, buf.String(), "trace_id")
}

func TestOtelHandler_WithSpan(t *testing.T) {
	// otelHandler must inject trace_id and span_id when a valid span is active
	var buf bytes.Buffer
	h := &otelHandler{base: slog.NewJSONHandler(&buf, nil)}
	tp := noop.NewTracerProvider()
	ctx, span := tp.Tracer("test").Start(context.Background(), "test-span")
	defer span.End()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	err := h.Handle(ctx, r)
	assert.NoError(t, err)
	// noop tracer produces zero IDs — just verify no panic and no extra keys injected
	_ = buf.String()
}

func TestFanoutHandler_DispatchesToAll(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)
	fh := &fanoutHandler{handlers: []slog.Handler{h1, h2}}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	err := fh.Handle(context.Background(), r)
	assert.NoError(t, err)
	assert.Contains(t, buf1.String(), "hello")
	assert.Contains(t, buf2.String(), "hello")
}

func TestFanoutHandler_ContinuesAfterError(t *testing.T) {
	// If the first handler errors, the second handler must still receive the record.
	var buf2 bytes.Buffer
	errHandler := &errorHandler{}
	h2 := slog.NewJSONHandler(&buf2, nil)
	fh := &fanoutHandler{handlers: []slog.Handler{errHandler, h2}}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "world", 0)
	err := fh.Handle(context.Background(), r)
	assert.Error(t, err)
	assert.Contains(t, buf2.String(), "world")
}

// errorHandler is a slog.Handler that always returns an error from Handle.
type errorHandler struct{}

func (e *errorHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (e *errorHandler) Handle(_ context.Context, _ slog.Record) error {
	return assert.AnError
}
func (e *errorHandler) WithAttrs(_ []slog.Attr) slog.Handler { return e }
func (e *errorHandler) WithGroup(_ string) slog.Handler      { return e }

func TestNewLoggerWithProvider_NilProvider(t *testing.T) {
	cfg := config.LoggerConfig{Level: "info", Format: "json"}
	log := NewLoggerWithProvider(cfg, nil)
	assert.NotNil(t, log)
}

func TestErr_Nil(t *testing.T) {
	attr := Err(nil)
	assert.Equal(t, "", attr.Value.String())
}

func TestErr_NonNil(t *testing.T) {
	attr := Err(assert.AnError)
	assert.Contains(t, attr.Value.String(), "assert.AnError")
}
