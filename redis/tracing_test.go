package redis

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/piresc/gokit/config"
)

func setupRedisTracing(t *testing.T) (*Client, *tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	})

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client, err := NewClient(config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return client, recorder, tp
}

func TestRedisHook_SpansWithParent(t *testing.T) {
	client, recorder, tp := setupRedisTracing(t)

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	require.NoError(t, client.Set(ctx, "k", "v", 0))
	_, err := client.Get(ctx, "k")
	require.NoError(t, err)
	parent.End()

	byName := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range recorder.Ended() {
		byName[s.Name()] = s
	}
	require.Contains(t, byName, "set")
	require.Contains(t, byName, "get")
	for _, name := range []string{"set", "get"} {
		s := byName[name]
		assert.Equal(t, parent.SpanContext().SpanID(), s.Parent().SpanID())
		assert.Equal(t, parent.SpanContext().TraceID(), s.SpanContext().TraceID())
		for _, a := range s.Attributes() {
			assert.False(t, strings.HasPrefix(string(a.Key), "db.statement"), string(a.Key))
		}
	}
}

func TestRedisHook_NoParentNoSpan(t *testing.T) {
	client, recorder, _ := setupRedisTracing(t)

	_, _ = client.Get(context.Background(), "k")

	assert.Empty(t, recorder.Ended())
}
