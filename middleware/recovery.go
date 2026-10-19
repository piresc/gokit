package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/piresc/gokit/response"

	traceapi "go.opentelemetry.io/otel/trace"
)

// RecoveryHandler catches panics, logs them with a stack trace, records the
// error in the active trace span, and returns 500 to the caller.
func (m *Middleware) RecoveryHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					err := fmt.Errorf("panic recovered: %v", rec)

					m.logger.Error("panic recovered",
						slog.String("error", fmt.Sprintf("%v", rec)),
						slog.String("stack", string(stack)),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					if m.tracer.ShouldTrace(r.URL.Path) {
						ctx, end := m.tracer.StartSegment(r.Context(), "panic.recovery")
						if span := traceapi.SpanFromContext(ctx); span.IsRecording() {
							span.RecordError(err)
						}
						end()
					}

					requestID := r.Header.Get("transactionid")
					if requestID == "" {
						requestID = r.Header.Get("X-Request-ID")
					}

					response.ErrorWithRequestID(w, http.StatusInternalServerError, response.CodeInternal, "Internal server error", requestID)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
