package middleware

import (
	"net/http"

	"github.com/piresc/gokit/ratelimit"
	"github.com/piresc/gokit/response"

	"github.com/gin-gonic/gin"
)

// RateLimiter returns a Gin middleware that enforces per-client-IP rate limiting.
// The store is created lazily on first call and reused across requests.
func (m *Middleware) RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	m.mu.Lock()
	if m.rateStore == nil {
		m.rateStore = ratelimit.NewStore(cfg)
	}
	m.mu.Unlock()

	return func(c *gin.Context) {
		if !m.rateStore.Allow(c.ClientIP()) {
			c.Abort()
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		c.Next()
	}
}
