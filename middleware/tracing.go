package middleware

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

// statusWriter wraps http.ResponseWriter to capture the written status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func newStatusWriter(w http.ResponseWriter) *statusWriter {
	return &statusWriter{ResponseWriter: w, status: http.StatusOK}
}

func traceFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	traceID := span.SpanContext().TraceID()
	if !traceID.IsValid() {
		return ""
	}
	return traceID.String()
}

// TracingHandler starts an OTel segment for each request, injects the trace ID
// into the response header, and records the HTTP status code.
func (m *Middleware) TracingHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if !m.tracer.ShouldTrace(path) {
				next.ServeHTTP(w, r)
				return
			}

			ctx, txn := m.tracer.StartHTTPRequest(r)
			r = r.WithContext(ctx)

			if spanID := traceFromContext(ctx); spanID != "" {
				w.Header().Set("X-Trace-Id", spanID)
			}

			txn.SetName(fmt.Sprintf("%s %s", r.Method, path))
			txn.AddAttribute("http.method", r.Method)
			txn.AddAttribute("http.url", r.RequestURI)
			txn.AddAttribute("http.route", path)
			txn.AddAttribute("http.user_agent", r.UserAgent())

			sw := newStatusWriter(w)
			defer func() {
				txn.AddAttribute("http.status_code", fmt.Sprintf("%d", sw.status))
				if sw.status >= http.StatusInternalServerError {
					txn.AddError(fmt.Errorf("http error: status %d", sw.status))
				}
				txn.End()
			}()

			next.ServeHTTP(sw, r)
		})
	}
}
