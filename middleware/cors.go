package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CorsHandler returns a Gin CORS middleware configured with the given allowed
// origins. Standard headers used by this toolkit (Authorization, X-Request-ID,
// X-API-Key, transactionid, x-msisdn) are whitelisted.
func (m *Middleware) CorsHandler(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"X-API-Key",
			"X-Request-ID",
			"transactionid",
			"x-msisdn",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
