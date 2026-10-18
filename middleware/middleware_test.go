package middleware_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/piresc/gokit/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func makeToken(userID, role string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	return tok
}

func newMiddleware() *middleware.Middleware {
	return middleware.NewMiddleware(nil, nil)
}

// chain wraps h with each middleware in order (first middleware is outermost).
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

func performRequest(h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// ---- JWTAuth ----

func TestJWTAuth_Valid(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, middleware.GetUserID(r))
		}),
		m.JWTAuth(testSecret),
	)

	token := makeToken("user-1", "admin")
	w := performRequest(h, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", w.Body.String())
}

func TestJWTAuth_Missing(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
	)

	w := performRequest(h, "GET", "/", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_Invalid(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
	)

	w := performRequest(h, "GET", "/", map[string]string{"Authorization": "Bearer bad.token"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_WrongFormat(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
	)

	w := performRequest(h, "GET", "/", map[string]string{"Authorization": "Basic something"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---- RequireRole ----

func TestRequireRole_Allowed(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
		m.RequireRole("admin"),
	)

	w := performRequest(h, "GET", "/", map[string]string{
		"Authorization": "Bearer " + makeToken("u1", "admin"),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
		m.RequireRole("admin"),
	)

	w := performRequest(h, "GET", "/", map[string]string{
		"Authorization": "Bearer " + makeToken("u2", "user"),
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---- SecurityHeaders ----

func TestSecurityHeaders(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.SecurityHeaders(),
	)

	w := performRequest(h, "GET", "/", nil)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

// ---- Recovery ----

func TestRecovery_Returns500(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("test panic") }),
		m.RecoveryHandler(),
	)

	w := performRequest(h, "GET", "/panic", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- RateLimiter ----

func TestRateLimiter_Allow(t *testing.T) {
	m := newMiddleware()
	cfg := middleware.RateLimitConfig{RequestsPerSecond: 100, BurstSize: 10}
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.RateLimiter(cfg),
	)

	w := performRequest(h, "GET", "/", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Reject(t *testing.T) {
	m := newMiddleware()
	cfg := middleware.RateLimitConfig{RequestsPerSecond: 1, BurstSize: 1}
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.RateLimiter(cfg),
	)

	performRequest(h, "GET", "/", nil) // consume burst
	w := performRequest(h, "GET", "/", nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimiter_SeparateBucketsBehindProxy(t *testing.T) {
	m := newMiddleware()
	cfg := middleware.RateLimitConfig{RequestsPerSecond: 1, BurstSize: 1}
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.RateLimiter(cfg),
	)
	do := func(xff string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "172.18.0.3:5000"
		req.Header.Set("X-Forwarded-For", xff)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	assert.Equal(t, http.StatusOK, do("198.51.100.7"))
	assert.Equal(t, http.StatusOK, do("198.51.100.8"))
	assert.Equal(t, http.StatusTooManyRequests, do("198.51.100.7"))
}

// ---- SetContextValues ----

func TestSetContextValues(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, middleware.GetTransactionID(r))
		}),
		m.SetContextValues(),
	)

	w := performRequest(h, "GET", "/", map[string]string{"transactionid": "txn-abc"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "txn-abc", w.Body.String())
}

// ---- CORS ----

func TestCorsHandler_NotNil(t *testing.T) {
	m := newMiddleware()
	assert.NotNil(t, m.CorsHandler([]string{"https://example.com"}))
	assert.NotNil(t, m.CorsHandler([]string{"*"}))
}

// ---- Shutdown ----

func TestShutdown_Safe(t *testing.T) {
	m := newMiddleware()
	assert.NotPanics(t, m.Shutdown)
}

// ---- error body shape ----

func TestJWTAuth_ErrorBody(t *testing.T) {
	m := newMiddleware()
	h := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		m.JWTAuth(testSecret),
	)

	w := performRequest(h, "GET", "/", nil)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "error", body["status"])
}
