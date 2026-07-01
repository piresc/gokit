package grpcmiddleware

import (
	"log/slog"

	"github.com/piresc/gokit/ratelimit"
	"github.com/piresc/gokit/redis"
	"github.com/piresc/gokit/tracing"
)

// Interceptors holds shared dependencies for all gRPC server interceptors.
type Interceptors struct {
	jwtSecret      string
	logger         *slog.Logger
	tracer         tracing.Tracer
	redisClient    *redis.Client
	rateLimitStore *ratelimit.Store
}

// NewInterceptors constructs an Interceptors instance. Nil logger falls back to
// slog.Default(); nil tracer falls back to a no-op tracer.
func NewInterceptors(jwtSecret string, logger *slog.Logger, tracer tracing.Tracer, redisClient *redis.Client) *Interceptors {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = tracing.NewNoOpTracer()
	}
	return &Interceptors{
		jwtSecret:   jwtSecret,
		logger:      logger,
		tracer:      tracer,
		redisClient: redisClient,
	}
}

// Shutdown releases resources held by the interceptors (e.g. rate-limit store).
func (i *Interceptors) Shutdown() {
	if i.rateLimitStore != nil {
		i.rateLimitStore.Stop()
	}
}
