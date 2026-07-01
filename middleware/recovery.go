package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/piresc/gokit/response"

	"github.com/gin-gonic/gin"
	traceapi "go.opentelemetry.io/otel/trace"
)

// RecoveryHandler catches panics, logs them with a stack trace, records the
// error in the active trace span, and returns 500 to the caller.
func (m *Middleware) RecoveryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err := fmt.Errorf("panic recovered: %v", r)

				m.logger.Error("panic recovered",
					slog.String("error", fmt.Sprintf("%v", r)),
					slog.String("stack", string(stack)),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)

				if m.tracer.ShouldTrace(c.Request.URL.Path) {
					ctx, end := m.tracer.StartSegment(c.Request.Context(), "panic.recovery")
					if span := traceapi.SpanFromContext(ctx); span.IsRecording() {
						span.RecordError(err)
					}
					end()
				}

				requestID := c.GetHeader("transactionid")
				if requestID == "" {
					requestID = c.GetHeader("X-Request-ID")
				}

				c.Abort()
				response.ErrorWithRequestID(c, http.StatusInternalServerError, "internal server error", requestID)
			}
		}()
		c.Next()
	}
}
