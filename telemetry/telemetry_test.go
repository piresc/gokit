package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_NoEndpoint_ReturnsProviders(t *testing.T) {
	cfg := Config{
		ServiceName:     "test-service",
		OTLPEndpoint:    "",
		TraceSampleRate: 1.0,
	}

	p, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.NotNil(t, p.TracerProvider)
	assert.NotNil(t, p.MeterProvider)
	assert.NotNil(t, p.LoggerProvider)

	err = p.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInit_WithEndpoint_ConnectsLazily(t *testing.T) {
	// OTLP gRPC connects lazily — constructor should succeed even with unreachable endpoint
	cfg := Config{
		ServiceName:     "test-service",
		OTLPEndpoint:    "localhost:0",
		TraceSampleRate: 0.5,
		MetricInterval:  1 * time.Second,
	}

	p, err := Init(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.Shutdown(ctx)
}

func TestProviders_Shutdown_NilProviders_IsNoop(t *testing.T) {
	p := &Providers{}
	err := p.Shutdown(context.Background())
	assert.NoError(t, err)
}
