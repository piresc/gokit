package middleware

import (
	"log/slog"
	"sync"

	"github.com/piresc/gokit/ratelimit"
	"github.com/piresc/gokit/tracing"
)

// RateLimitConfig aliases ratelimit.Config for convenience.
type RateLimitConfig = ratelimit.Config

// DefaultRateLimitConfig returns sane defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return ratelimit.DefaultConfig()
}

// Middleware holds shared dependencies for all Gin middleware handlers.
type Middleware struct {
	logger *slog.Logger
	tracer tracing.Tracer

	mu        sync.Mutex
	rateStore *ratelimit.Store
}

// NewMiddleware constructs a Middleware. Nil logger falls back to slog.Default();
// nil tracer falls back to a no-op tracer.
func NewMiddleware(logger *slog.Logger, tracer tracing.Tracer) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = tracing.NewNoOpTracer()
	}
	return &Middleware{
		logger: logger,
		tracer: tracer,
	}
}

// Shutdown releases resources held by the middleware (e.g. rate-limit store).
func (m *Middleware) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rateStore != nil {
		m.rateStore.Stop()
	}
}
