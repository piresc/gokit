// Package bootstrap wires all gokit infra packages into two constructors:
// NewGRPCService and NewHTTPService.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"

	pkgasynq "github.com/piresc/gokit/asynq"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/database"
	"github.com/piresc/gokit/logger"
	"github.com/piresc/gokit/mail"
	"github.com/piresc/gokit/metrics"
	pkgnats "github.com/piresc/gokit/nats"
	pkgredis "github.com/piresc/gokit/redis"
	"github.com/piresc/gokit/server"
	"github.com/piresc/gokit/telemetry"
	"github.com/piresc/gokit/tracing"
)

// baseApp holds infrastructure components shared by GRPCApp and HTTPApp.
type baseApp struct {
	log         *slog.Logger
	tracer      tracing.Tracer
	providers   *telemetry.Providers
	appMetrics  *metrics.Metrics
	db          *sqlx.DB
	pgClient    *database.PostgresClient
	redis       *pkgredis.Client
	natsClient  *pkgnats.Client
	asynqClient *pkgasynq.Client
	mailSender  mail.Sender
	shutdownMgr *server.ShutdownManager
	grpcConns   map[string]*grpc.ClientConn

	hookMu sync.Mutex
	hooks  []func(context.Context) error
}

// onShutdown registers a hook that runs before infrastructure cleanup.
func (a *baseApp) onShutdown(fn func(context.Context) error) {
	a.hookMu.Lock()
	defer a.hookMu.Unlock()
	a.hooks = append(a.hooks, fn)
}

// shutdown runs the hooks newest-first, then the infrastructure cleanup.
// Every step runs even if an earlier one fails; all errors are joined.
func (a *baseApp) shutdown(ctx context.Context) error {
	a.hookMu.Lock()
	hooks := make([]func(context.Context) error, len(a.hooks))
	copy(hooks, a.hooks)
	a.hookMu.Unlock()

	var errs []error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i](ctx); err != nil {
			a.log.Error("shutdown hook failed", logger.Err(err))
			errs = append(errs, err)
		}
	}
	if err := a.shutdownMgr.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// initBase loads config[T], creates a logger, and wires observability + infra.
func initBase[T any](serviceName string) (*baseApp, *config.Config[T], error) {
	cfg, err := config.Load[T](serviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("config load: %w", err)
	}

	log := logger.NewLogger(cfg.Logger)
	app := &baseApp{
		log:         log,
		shutdownMgr: server.NewShutdownManager(log),
		grpcConns:   make(map[string]*grpc.ClientConn),
	}

	app.initObservability(cfg.Base, serviceName)

	if err := app.initInfra(cfg.Base); err != nil {
		return nil, nil, err
	}

	return app, cfg, nil
}

// initObservability sets up telemetry, tracing, and metrics.
// Failures are non-fatal: each falls back to a noop implementation.
func (a *baseApp) initObservability(cfg config.Base, serviceName string) {
	otlp := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint)

	providers, err := telemetry.Init(context.Background(), telemetry.Config{
		ServiceName:     serviceName,
		OTLPEndpoint:    otlp,
		TraceSampleRate: cfg.Tracing.SampleRate,
	})
	if err != nil {
		a.log.Warn("telemetry init failed, continuing with noop", logger.Err(err))
	} else {
		a.providers = providers
	}

	tracer, err := tracing.NewTracer(&tracing.Config{
		Enabled:      cfg.Tracing.Enabled,
		ServiceName:  serviceName,
		SampleRate:   cfg.Tracing.SampleRate,
		Exporter:     cfg.Tracing.Exporter,
		OTLPEndpoint: cfg.Tracing.OTLPEndpoint,
		ExcludePaths: tracingExcludePaths(cfg.Tracing.ExcludePaths),
	})
	if err != nil {
		a.log.Warn("tracer init failed, noop fallback", logger.Err(err))
		tracer = tracing.NewNoOpTracer()
	}
	a.tracer = tracer

	m, err := metrics.NewMetrics(serviceName, otlp)
	if err != nil {
		a.log.Warn("metrics init failed, continuing", logger.Err(err))
	} else {
		a.appMetrics = m
	}
}

// initInfra connects DB, Redis, NATS, Asynq, and mail based on config presence.
func (a *baseApp) initInfra(cfg config.Base) error {
	if cfg.Database.Host != "" {
		pg, err := database.NewPostgresClient(cfg.Database)
		if err != nil {
			return fmt.Errorf("database connect: %w", err)
		}
		a.pgClient = pg
		a.db = pg.GetDB()
		a.shutdownMgr.Register(func(_ context.Context) error { return pg.Close() })
	}

	if cfg.Redis.Host != "" {
		rdb, err := pkgredis.NewClient(cfg.Redis)
		if err != nil {
			return fmt.Errorf("redis connect: %w", err)
		}
		a.redis = rdb
		a.shutdownMgr.Register(func(_ context.Context) error { return rdb.Close() })
	}

	if cfg.NATS.Enabled && cfg.NATS.URL != "" {
		nc, err := pkgnats.NewClient(cfg.NATS.URL)
		if err != nil {
			return fmt.Errorf("nats connect: %w", err)
		}
		a.natsClient = nc
		a.shutdownMgr.Register(func(ctx context.Context) error {
			nc.Close(ctx)
			return nil
		})
	}

	if cfg.Asynq.Enabled && cfg.Redis.Host != "" {
		addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
		ac := pkgasynq.NewClient(addr, cfg.Redis.Password)
		a.asynqClient = ac
		a.shutdownMgr.Register(func(_ context.Context) error {
			return ac.Close()
		})
	}

	a.mailSender = mail.NewSender(cfg.Mail)
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
