package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"

	pkgasynq "github.com/piresc/gokit/asynq"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/grpcmiddleware"
	"github.com/piresc/gokit/grpcserver"
	"github.com/piresc/gokit/logger"
	"github.com/piresc/gokit/mail"
	pkgnats "github.com/piresc/gokit/nats"
	pkgredis "github.com/piresc/gokit/redis"
	"github.com/piresc/gokit/tracing"
)

// GRPCApp is a fully-wired gRPC service. Public fields are ready for injection
// into repositories and use-cases.
type GRPCApp[T any] struct {
	// Public wiring fields
	DB          *sqlx.DB
	Redis       *pkgredis.Client
	NATSClient  *pkgnats.Client
	AsynqClient *pkgasynq.Client
	MailSender  mail.Sender
	Logger      *slog.Logger
	Tracer      tracing.Tracer
	Config      *config.Config[T]

	base       *baseApp
	grpcServer *grpcserver.GRPCServer
}

// NewGRPCService loads config, wires all configured infrastructure, and
// builds a gRPC server with the standard interceptor chain.
//
// Required config: app + logger. DB/Redis/NATS/Asynq wired when host/enabled.
func NewGRPCService[T any](serviceName string) (*GRPCApp[T], error) {
	base, cfg, err := initBase[T](serviceName)
	if err != nil {
		return nil, err
	}

	app := &GRPCApp[T]{
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

	// Build interceptor chain.
	interceptors := grpcmiddleware.NewInterceptors(
		cfg.JWT.Secret,
		base.log,
		base.tracer,
		base.redis,
	)

	var chain []grpc.UnaryServerInterceptor
	if base.appMetrics != nil {
		chain = append(chain, base.appMetrics.GRPCUnaryInterceptor())
	}
	chain = append(chain,
		interceptors.RecoveryUnaryInterceptor(),
		interceptors.AuthUnaryInterceptor(nil),
		interceptors.LoggingUnaryInterceptor(),
		interceptors.TracingUnaryInterceptor(),
	)

	shutdownTimeout := cfg.GRPC.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	app.grpcServer = grpcserver.New(
		base.log,
		cfg.GRPC.Server.Port,
		shutdownTimeout,
		grpc.ChainUnaryInterceptor(chain...),
	)

	return app, nil
}

// RegisterService registers gRPC service handlers with the underlying server.
func (a *GRPCApp[T]) RegisterService(reg func(*grpc.Server)) {
	reg(a.grpcServer.Server())
}

// Run starts the gRPC server and blocks until SIGINT/SIGTERM, then shuts down.
func (a *GRPCApp[T]) Run() error {
	a.Logger.Info("gRPC server starting", slog.Int("port", a.Config.GRPC.Server.Port))
	// Start blocks until signal received and gRPC server is drained.
	if err := a.grpcServer.Start(); err != nil {
		return fmt.Errorf("grpc server: %w", err)
	}
	shutdownTimeout := a.Config.GRPC.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return a.Shutdown(shutdownCtx)
}

// Shutdown gracefully stops the gRPC server and all registered infrastructure.
func (a *GRPCApp[T]) Shutdown(ctx context.Context) error {
	if err := a.grpcServer.GracefulStop(); err != nil {
		a.Logger.Warn("grpc graceful stop error", logger.Err(err))
	}
	return a.base.shutdownMgr.Shutdown(ctx)
}
