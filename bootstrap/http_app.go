package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"

	pkgasynq "github.com/piresc/gokit/asynq"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/grpcclient"
	"github.com/piresc/gokit/health"
	"github.com/piresc/gokit/mail"
	"github.com/piresc/gokit/middleware"
	pkgnats "github.com/piresc/gokit/nats"
	"github.com/piresc/gokit/ratelimit"
	pkgredis "github.com/piresc/gokit/redis"
	"github.com/piresc/gokit/server"
	"github.com/piresc/gokit/tracing"
)

// HTTPApp is a fully-wired HTTP gateway service. Public fields are ready for
// injection into handlers and use-cases.
type HTTPApp[T any] struct {
	// Public wiring fields
	DB          *sqlx.DB
	Redis       *pkgredis.Client
	NATSClient  *pkgnats.Client
	AsynqClient *pkgasynq.Client
	MailSender  mail.Sender
	Logger      *slog.Logger
	Tracer      tracing.Tracer
	Config      *config.Config[T]
	// Mux is the underlying router; register application routes here.
	Mux *http.ServeMux

	base *baseApp
	srv  *server.GracefulServer
}

// NewHTTPService loads config, wires all configured infrastructure, and builds
// an http.ServeMux with the standard middleware chain.
//
// Health routes (/health/live, /health/ready) are auto-registered.
func NewHTTPService[T any](serviceName string) (*HTTPApp[T], error) {
	base, cfg, err := initBase[T](serviceName)
	if err != nil {
		return nil, err
	}

	app := &HTTPApp[T]{
		DB:          base.db,
		Redis:       base.redis,
		NATSClient:  base.natsClient,
		AsynqClient: base.asynqClient,
		MailSender:  base.mailSender,
		Logger:      base.log,
		Tracer:      base.tracer,
		Config:      cfg,
		base:        base,
	}

	mux := http.NewServeMux()

	// Auto-register health checks for connected dependencies.
	healthSvc := health.NewService(base.log)
	if base.pgClient != nil {
		healthSvc.AddChecker("postgres", health.NewPostgresChecker(base.pgClient))
	}
	if base.redis != nil {
		healthSvc.AddChecker("redis", health.NewRedisChecker(base.redis))
	}
	health.RegisterRoutes(mux, cfg.App.Name, cfg.App.Version, healthSvc)

	// Build middleware chain (outermost first).
	mw := middleware.NewMiddleware(base.log, base.tracer)

	allowedOrigins := cfg.Server.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	rlCfg := middleware.DefaultRateLimitConfig()
	if len(cfg.Server.TrustedProxies) > 0 {
		// Entries were already validated by config loading.
		trusted, err := ratelimit.ParseTrustedProxies(cfg.Server.TrustedProxies)
		if err != nil {
			return nil, fmt.Errorf("server.trusted_proxies: %w", err)
		}
		rlCfg.TrustedProxies = trusted
	}

	var handler http.Handler = mux
	handler = mw.TracingHandler()(handler)
	handler = mw.RateLimiter(rlCfg)(handler)
	handler = mw.CorsHandler(allowedOrigins)(handler)
	if base.appMetrics != nil {
		handler = base.appMetrics.HTTPMiddleware()(handler)
	}
	handler = mw.SecurityHeaders()(handler)
	handler = mw.RecoveryHandler()(handler)

	shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeout) * time.Second
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	app.Mux = mux
	app.srv = server.NewGracefulServer(handler, base.log, cfg.Server.Port, shutdownTimeout)

	return app, nil
}

// GRPCConn returns a cached gRPC client connection for the named backend service.
// The name must match a key in grpc_clients config. Panics on missing config or
// dial failure (fail-fast at startup).
func (a *HTTPApp[T]) GRPCConn(name string) *grpc.ClientConn {
	if conn, ok := a.base.grpcConns[name]; ok {
		return conn
	}

	clients := a.Config.GRPCClients
	entry, ok := clients[name]
	if !ok || entry.Target == "" {
		panic(fmt.Sprintf("bootstrap: grpc_clients.%s not configured", name))
	}

	conn, err := grpcclient.Dial(context.Background(), grpcclient.ClientConfig{
		Target: entry.Target,
		Tracer: a.base.tracer,
		Logger: a.base.log,
	})
	if err != nil {
		panic(fmt.Sprintf("bootstrap: dial %s (%s): %v", name, entry.Target, err))
	}

	a.base.grpcConns[name] = conn
	// Register close in shutdown manager.
	c := conn
	a.base.shutdownMgr.Register(func(_ context.Context) error { return c.Close() })

	return conn
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM, then shuts down
// all registered infrastructure.
func (a *HTTPApp[T]) Run() error {
	a.Logger.Info("HTTP server starting", slog.Int("port", a.Config.Server.Port))
	// Start blocks until signal received and the HTTP server is drained.
	if err := a.srv.Start(); err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	timeout := 30 * time.Second
	if a.Config.Server.ShutdownTimeout > 0 {
		timeout = time.Duration(a.Config.Server.ShutdownTimeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.Shutdown(ctx)
}

// Shutdown closes all registered infrastructure components.
func (a *HTTPApp[T]) Shutdown(ctx context.Context) error {
	return a.base.shutdownMgr.Shutdown(ctx)
}
