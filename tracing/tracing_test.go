package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpTracer_ShouldReturnNoOpTracer(t *testing.T) {
	tracer := NewNoOpTracer()
	assert.NotNil(t, tracer)
	_, ok := tracer.(*NoOpTracer)
	assert.True(t, ok)
}

func TestNoOpTracer_IsEnabled_ShouldReturnFalse(t *testing.T) {
	assert.False(t, NewNoOpTracer().IsEnabled())
}

func TestNoOpTracer_ShouldTrace_ShouldReturnFalse(t *testing.T) {
	tracer := NewNoOpTracer()
	assert.False(t, tracer.ShouldTrace("/api/v1/users"))
	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace(""))
}

func TestNoOpTracer_Shutdown_ShouldReturnNil(t *testing.T) {
	assert.NoError(t, NewNoOpTracer().Shutdown(context.Background()))
}

func TestNoOpTracer_StartHTTPRequest(t *testing.T) {
	tracer := NewNoOpTracer()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx, txn := tracer.StartHTTPRequest(req)
	assert.NotNil(t, ctx)
	assert.NotNil(t, txn)
	_, ok := txn.(*NoOpTransaction)
	assert.True(t, ok)
}

func TestNoOpTracer_StartExternalCall(t *testing.T) {
	tracer := NewNoOpTracer()
	newCtx, endFn := tracer.StartExternalCall(context.Background(), "example.com", "GET")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartMessage(t *testing.T) {
	tracer := NewNoOpTracer()
	newCtx, endFn := tracer.StartMessage(context.Background(), "orders", "publish")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartDatabase(t *testing.T) {
	tracer := NewNoOpTracer()
	newCtx, endFn := tracer.StartDatabase(context.Background(), "SELECT", "users")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_StartSegment(t *testing.T) {
	tracer := NewNoOpTracer()
	newCtx, endFn := tracer.StartSegment(context.Background(), "seg")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, endFn)
	endFn()
}

func TestNoOpTracer_InjectContext_ShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		NewNoOpTracer().InjectContext(context.Background(), nil)
	})
}

func TestNoOpTracer_ExtractContext_ShouldReturnSameContext(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, ctx, NewNoOpTracer().ExtractContext(ctx, nil))
}

func TestNoOpTransaction_Methods_ShouldNotPanic(t *testing.T) {
	txn := &NoOpTransaction{ctx: context.Background()}
	assert.NotPanics(t, func() {
		txn.SetName("test")
		txn.AddAttribute("k", "v")
		txn.AddError(assert.AnError)
		txn.End()
	})
}

func TestNewTracer_NilConfig_ReturnsNoOp(t *testing.T) {
	tracer, err := NewTracer(nil)
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_Disabled_ReturnsNoOp(t *testing.T) {
	tracer, err := NewTracer(&Config{Enabled: false, Exporter: "stdout"})
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_NoOpForTesting_ReturnsNoOp(t *testing.T) {
	tracer, err := NewTracer(&Config{Enabled: true, NoOpForTesting: true})
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_NoopExporter_ReturnsNoOp(t *testing.T) {
	tracer, err := NewTracer(&Config{Enabled: true, Exporter: "noop"})
	require.NoError(t, err)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_StdoutExporter_ReturnsOtelTracer(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-service",
		SampleRate:  1.0,
		Exporter:    "stdout",
	}
	tracer, err := NewTracer(cfg)
	require.NoError(t, err)
	assert.IsType(t, &otelTracer{}, tracer)
	assert.True(t, tracer.IsEnabled())
	_ = tracer.Shutdown(context.Background())
}

func TestOtelTracer_ShouldTrace_ExcludedPaths(t *testing.T) {
	cfg := &Config{
		Enabled:      true,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Exporter:     "stdout",
		ExcludePaths: []string{"/health", "/metrics"},
	}
	tracer, err := NewTracer(cfg)
	require.NoError(t, err)
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	assert.True(t, tracer.ShouldTrace("/api/v1/users"))
	assert.False(t, tracer.ShouldTrace("/health"))
	assert.False(t, tracer.ShouldTrace("/health/ready"))
	assert.False(t, tracer.ShouldTrace("/metrics"))
}
