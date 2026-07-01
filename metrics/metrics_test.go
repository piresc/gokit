package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics_NoEndpoint(t *testing.T) {
	m, err := NewMetrics("test-service", "")
	require.NoError(t, err)
	require.NotNil(t, m)

	assert.NotNil(t, m.HTTPRequestsTotal)
	assert.NotNil(t, m.HTTPRequestDuration)
	assert.NotNil(t, m.HTTPResponseSize)
	assert.NotNil(t, m.GRPCRequestsTotal)
	assert.NotNil(t, m.GRPCRequestDuration)
	assert.NotNil(t, m.DBQueryDuration)
	assert.NotNil(t, m.Meter())

	err = m.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNewMetrics_WithEndpoint_ConnectsLazily(t *testing.T) {
	m, err := NewMetrics("test-service", "localhost:0")
	require.NoError(t, err)
	require.NotNil(t, m)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = m.Shutdown(ctx)
}

func TestMetrics_Shutdown_NilProvider(t *testing.T) {
	m := &Metrics{provider: nil}
	err := m.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestMetrics_RecordDBQuery_NoPanic(t *testing.T) {
	m, err := NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = m.Shutdown(context.Background()) }()

	assert.NotPanics(t, func() {
		m.RecordDBQuery(context.Background(), "SELECT", "users", 0.042)
	})
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000/profile", "/api/v1/users/:id/profile"},
		{"/api/v1/parking/123/spots", "/api/v1/parking/:id/spots"},
		{"/api/v1/health", "/api/v1/health"},
		{"/api/v1/floors/3/spots/42", "/api/v1/floors/:id/spots/:id"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, normalizePath(tt.input), tt.input)
	}
}

func TestGRPCUnaryInterceptor_NotNil(t *testing.T) {
	m, err := NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = m.Shutdown(context.Background()) }()

	assert.NotNil(t, m.GRPCUnaryInterceptor())
}

func TestHTTPMiddleware_NotNil(t *testing.T) {
	m, err := NewMetrics("test-service", "")
	require.NoError(t, err)
	defer func() { _ = m.Shutdown(context.Background()) }()

	assert.NotNil(t, m.HTTPMiddleware())
}
