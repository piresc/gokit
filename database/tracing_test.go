package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/database"
	"github.com/piresc/gokit/testkit"
)

func setupPostgresTracing(t *testing.T) (*database.PostgresClient, *tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	t.Helper()

	ctx := context.Background()
	pc, err := testkit.NewPostgresContainer(ctx, nil, "")
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close(ctx) })

	host, err := pc.Container.Host(ctx)
	require.NoError(t, err)
	port, err := pc.Container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

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

	client, err := database.NewPostgresClient(config.DatabaseConfig{
		Host:     host,
		Port:     int(port.Num()),
		Username: "postgres",
		Password: "postgres",
		Database: "testdb",
		SSLMode:  "disable",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	recorder.Reset()
	return client, recorder, tp
}

func TestPostgres_SpanWithParent(t *testing.T) {
	client, recorder, tp := setupPostgresTracing(t)

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	var one int
	require.NoError(t, client.GetDB().QueryRowxContext(ctx, "SELECT 1").Scan(&one))
	parent.End()

	found := false
	for _, s := range recorder.Ended() {
		if s.SpanContext().TraceID() == parent.SpanContext().TraceID() && s.Name() != "parent" {
			found = true
		}
	}
	assert.True(t, found, "expected a child span of the parent trace")
}

func TestPostgres_NoParentNoSpan(t *testing.T) {
	client, recorder, _ := setupPostgresTracing(t)

	require.NoError(t, client.GetDB().PingContext(context.Background()))
	var one int
	require.NoError(t, client.GetDB().QueryRowxContext(context.Background(), "SELECT 1").Scan(&one))

	assert.Empty(t, recorder.Ended())
}
