package health_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piresc/gokit/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- stub checker ----

type stubChecker struct {
	name string
	err  error
}

func (s *stubChecker) Check(_ context.Context) error { return s.err }
func (s *stubChecker) Name() string                  { return s.name }

// ---- Service ----

func TestService_AllHealthy(t *testing.T) {
	svc := health.NewService(slog.Default())
	svc.AddChecker("db", &stubChecker{name: "db", err: nil})
	result, err := svc.CheckAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, health.StatusHealthy, result[health.KeyStatus])
}

func TestService_OneUnhealthy(t *testing.T) {
	svc := health.NewService(slog.Default())
	svc.AddChecker("db", &stubChecker{name: "db", err: errors.New("timeout")})
	result, err := svc.CheckAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, health.StatusUnhealthy, result[health.KeyStatus])
}

func TestService_Empty(t *testing.T) {
	svc := health.NewService(nil)
	result, err := svc.CheckAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, health.StatusHealthy, result[health.KeyStatus])
}

// ---- Routes ----

func newMux(svc *health.Service) *http.ServeMux {
	mux := http.NewServeMux()
	health.RegisterRoutes(mux, "test-svc", "v1.0.0", svc)
	return mux
}

func getPath(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestRoutes_Shallow(t *testing.T) {
	mux := newMux(health.NewService(nil))
	w := getPath(mux, "/health")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoutes_Live(t *testing.T) {
	mux := newMux(health.NewService(nil))
	w := getPath(mux, "/health/live")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoutes_Ready_Healthy(t *testing.T) {
	svc := health.NewService(nil)
	svc.AddChecker("ok", &stubChecker{name: "ok"})
	mux := newMux(svc)
	w := getPath(mux, "/health/ready")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoutes_Ready_Unhealthy(t *testing.T) {
	svc := health.NewService(nil)
	svc.AddChecker("broken", &stubChecker{name: "broken", err: errors.New("dead")})
	mux := newMux(svc)
	w := getPath(mux, "/health/ready")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRoutes_Detailed(t *testing.T) {
	svc := health.NewService(nil)
	mux := newMux(svc)
	w := getPath(mux, "/health/detailed")
	assert.Equal(t, http.StatusOK, w.Code)
}
