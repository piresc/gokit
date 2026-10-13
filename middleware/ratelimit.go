package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/piresc/gokit/ratelimit"
	"github.com/piresc/gokit/response"
)

// clientIP extracts the client IP from X-Forwarded-For or RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma-separated list; take the first entry.
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimiter returns a net/http middleware that enforces per-client-IP rate
// limiting. The store is created lazily on first call and reused across requests.
func (m *Middleware) RateLimiter(cfg RateLimitConfig) func(http.Handler) http.Handler {
	m.mu.Lock()
	if m.rateStore == nil {
		m.rateStore = ratelimit.NewStore(cfg)
	}
	m.mu.Unlock()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.rateStore.Allow(clientIP(r)) {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
