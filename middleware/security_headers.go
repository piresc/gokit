package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets standard HTTP security headers on every response.
// Headers applied:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - X-XSS-Protection: 0 (modern browsers ignore it; CSP covers this)
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains
//   - Content-Security-Policy: default-src 'self'
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: camera=(), microphone=(), geolocation=()
//   - Cache-Control: no-store
func (m *Middleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", "default-src 'self'")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}
