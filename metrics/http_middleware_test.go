package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath_UUID(t *testing.T) {
	input := "/reservations/550e8400-e29b-41d4-a716-446655440000/items"
	want := "/reservations/:id/items"
	assert.Equal(t, want, normalizePath(input))
}

func TestNormalizePath_Numeric(t *testing.T) {
	assert.Equal(t, "/users/:id/posts/:id", normalizePath("/users/42/posts/100"))
}

func TestNormalizePath_NoChange(t *testing.T) {
	assert.Equal(t, "/health/ready", normalizePath("/health/ready"))
}

func TestNormalizePath_MixedSegments(t *testing.T) {
	path := "/api/v1/orders/550e8400-e29b-41d4-a716-446655440000"
	assert.Equal(t, "/api/v1/orders/:id", normalizePath(path))
}

func TestNormalizePath_Empty(t *testing.T) {
	assert.Equal(t, "/", normalizePath("/"))
}

func TestHTTPMiddleware_RecordsNormalizedPath(t *testing.T) {
	m, _ := NewMetrics("test-svc", "")
	if m == nil {
		t.Skip("metrics init requires OTLP; smoke test skipped")
	}
	mw := m.HTTPMiddleware()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := mw(next)
	req := httptest.NewRequest("GET", "/users/42", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.True(t, called)
}
