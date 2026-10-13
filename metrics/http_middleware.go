package metrics

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
var numericPattern = regexp.MustCompile(`^\d+$`)

func normalizePath(path string) string {
	normalized := uuidPattern.ReplaceAllString(path, ":id")

	parts := splitPath(normalized)
	for i, part := range parts {
		if numericPattern.MatchString(part) {
			parts[i] = ":id"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			parts = append(parts, "/")
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func joinPath(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}

// HTTPMiddleware returns a middleware that records HTTP request metrics.
func (m *Metrics) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()
			method := r.Method
			path := normalizePath(r.URL.Path)
			statusCode := fmt.Sprintf("%d", rw.statusCode)

			attrs := otelmetric.WithAttributes(
				attribute.String("method", method),
				attribute.String("path", path),
				attribute.String("status_code", statusCode),
			)

			m.HTTPRequestsTotal.Add(r.Context(), 1, attrs)
			m.HTTPRequestDuration.Record(r.Context(), duration, attrs)

			sizeAttrs := otelmetric.WithAttributes(
				attribute.String("method", method),
				attribute.String("path", path),
			)
			m.HTTPResponseSize.Record(r.Context(), int64(rw.size), sizeAttrs)
		})
	}
}
