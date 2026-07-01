package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/piresc/gokit/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

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

func performRequest(r http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- JWTAuth ----

func TestJWTAuth_Valid(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), func(c *gin.Context) {
		uid, _ := c.Get(middleware.KeyUserID)
		c.String(http.StatusOK, uid.(string))
	})

	token := makeToken("user-1", "admin")
	w := performRequest(r, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", w.Body.String())
}

func TestJWTAuth_Missing(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_Invalid(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", map[string]string{"Authorization": "Bearer bad.token"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_WrongFormat(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", map[string]string{"Authorization": "Basic something"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---- RequireRole ----

func TestRequireRole_Allowed(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), m.RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := performRequest(r, "GET", "/", map[string]string{
		"Authorization": "Bearer " + makeToken("u1", "admin"),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), m.RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := performRequest(r, "GET", "/", map[string]string{
		"Authorization": "Bearer " + makeToken("u2", "user"),
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---- SecurityHeaders ----

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.SecurityHeaders(), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", nil)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

// ---- Recovery ----

func TestRecovery_Returns500(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/panic", m.RecoveryHandler(), func(c *gin.Context) { panic("test panic") })

	w := performRequest(r, "GET", "/panic", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- RateLimiter ----

func TestRateLimiter_Allow(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	cfg := middleware.RateLimitConfig{RequestsPerSecond: 100, BurstSize: 10}
	r.GET("/", m.RateLimiter(cfg), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Reject(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	cfg := middleware.RateLimitConfig{RequestsPerSecond: 1, BurstSize: 1}
	r.GET("/", m.RateLimiter(cfg), func(c *gin.Context) { c.Status(200) })

	performRequest(r, "GET", "/", nil) // consume burst
	w := performRequest(r, "GET", "/", nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// ---- SetContextValues ----

func TestSetContextValues(t *testing.T) {
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.SetContextValues(), func(c *gin.Context) {
		tid, _ := c.Get(middleware.KeyTransactionID)
		c.String(http.StatusOK, tid.(string))
	})

	w := performRequest(r, "GET", "/", map[string]string{"transactionid": "txn-abc"})
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
	r := gin.New()
	m := newMiddleware()
	r.GET("/", m.JWTAuth(testSecret), func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/", nil)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "error", body["status"])
}
