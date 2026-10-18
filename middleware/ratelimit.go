package middleware

import (
	"net/http"

	"github.com/piresc/gokit/ratelimit"
	"github.com/piresc/gokit/response"
)

// RateLimiter returns a net/http middleware that enforces per-client-IP rate
// limiting. The store is created lazily on first call and reused across requests.
// The client IP is resolved using cfg.TrustedProxies (nil means the defaults, a
// non-nil empty slice trusts no proxy).
func (m *Middleware) RateLimiter(cfg RateLimitConfig) func(http.Handler) http.Handler {
	m.mu.Lock()
	if m.rateStore == nil {
		m.rateStore = ratelimit.NewStore(cfg)
	}
	store := m.rateStore
	m.mu.Unlock()

	trusted := cfg.TrustedProxies
	if trusted == nil {
		trusted = ratelimit.DefaultTrustedProxies()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !store.Allow(ClientIP(r, trusted)) {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
