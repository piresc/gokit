package health

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts health endpoints on the given engine:
//
//	GET /health          — shallow check (service metadata)
//	GET /health/live     — liveness probe, always 200
//	GET /health/ready    — readiness probe (runs all checkers)
//	GET /health/detailed — same as /ready but always returns full detail
func RegisterRoutes(r *gin.Engine, serviceName, version string, svc *Service) {
	g := r.Group("/health")

	g.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			KeyStatus:      "ok",
			"service_name": serviceName,
			"version":      version,
			"go_version":   runtime.Version(),
		})
	})

	g.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{KeyStatus: "ok"})
	})

	g.GET("/ready", func(c *gin.Context) {
		result, err := svc.CheckAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}
		if s, _ := result[KeyStatus].(string); s != StatusHealthy {
			c.JSON(http.StatusServiceUnavailable, result)
			return
		}
		c.JSON(http.StatusOK, result)
	})

	g.GET("/detailed", func(c *gin.Context) {
		result, err := svc.CheckAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}
		if s, _ := result[KeyStatus].(string); s != StatusHealthy {
			c.JSON(http.StatusServiceUnavailable, result)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}
