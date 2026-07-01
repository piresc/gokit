# gokit

Shared infrastructure primitives for Go services. Consumed by app-template via a `replace` directive in local dev and as a versioned module dependency in CI.

## Package reference

| Package | What it does | Key API |
|---------|-------------|---------|
| `apperror` | Typed application errors with HTTP status codes | `New`, `BadRequest`, `NotFound`, `Forbidden`, `Conflict`, `Internal`, `ServiceUnavailable` |
| `auth` | JWT sign / validate | `GenerateToken(userID, role, secret, ttl)`, `ValidateToken(token, secret) (*Claims, error)` |
| `bootstrap` | One-call service wiring — config, DB, Redis, NATS, Asynq, tracing, metrics, graceful shutdown | `NewGRPCService[T](name)`, `NewHTTPService[T](name)`, `app.GRPCConn(name)`, `app.Run()`, `app.Shutdown(ctx)` |
| `circuitbreaker` | Sony gobreaker wrapper | `New(Config)`, `cb.Execute(fn)` |
| `config` | Generic Viper-based config loader; merges YAML + env vars | `Load[T](serviceName)`, `Config[T]{Base, Service T}` |
| `database` | pgx/sqlx Postgres client with tracing | `NewPostgresClient(cfg)`, `client.GetDB()`, `client.Ping(ctx)`, `client.Close()` |
| `grpcclient` | Dial a gRPC backend with OTel tracing | `Dial(ctx, ClientConfig{Target, Tracer, Logger})` |
| `grpcerror` | Map `apperror` → gRPC status codes | `Map(err) error` |
| `grpcmiddleware` | Unary + stream interceptors: auth, logging, tracing, recovery, rate-limit | `UnaryServerInterceptors(...)`, `StreamServerInterceptors(...)` |
| `grpcserver` | gRPC server with graceful shutdown | `NewServer(cfg, log, interceptors...)`, `server.Start()`, `server.Stop(ctx)` |
| `health` | Liveness / readiness HTTP endpoints + pluggable checkers | `NewService(log)`, `svc.AddChecker(name, checker)`, `RegisterRoutes(engine, name, version, svc)` |
| `httperror` | Write `apperror` as a JSON HTTP response | `Write(c, err)`, `WriteWithRequestID(c, err, reqID)` |
| `idempotency` | Redis-backed idempotency key enforcement | `NewStore(redis, ttl)`, `store.Do(ctx, key, fn)` |
| `logger` | `slog.Logger` factory (JSON or text, configurable level) | `NewLogger(cfg)`, `Err(err)` |
| `mail` | SMTP mail sender with provider abstraction | `NewSender(cfg) Sender`, `sender.Send(ctx, msg)` |
| `metrics` | OTel metrics — HTTP middleware + gRPC interceptor + Prometheus server | `NewMetrics(service, otlpEndpoint)`, `m.HTTPMiddleware()`, `m.GRPCUnaryInterceptor()` |
| `middleware` | Gin HTTP middleware: JWT auth, CORS, rate-limit, recovery, security headers, tracing | `NewMiddleware(log, tracer)`, `mw.JWTAuth(secret)`, `mw.RequireRole(roles...)`, `mw.CorsHandler(origins)` |
| `nats` | NATS JetStream client + typed publisher | `NewClient(url)`, `NewPublisher(client)`, `publisher.Publish(ctx, subject, data)` |
| `ratelimit` | Token-bucket rate-limit store (used by middleware) | `NewStore(cfg)`, `store.Allow(key)` |
| `redis` | go-redis v9 client with OTel tracing | `NewClient(cfg)`, `client.GetClient()`, `client.Close()` |
| `redislock` | Distributed lock via Redis | `NewLocker(client, cfg)`, `locker.Acquire(ctx, key) (Lock, error)`, `lock.Release(ctx)` |
| `response` | Standard JSON response envelope | `Success(c, status, data)`, `Error(c, status, msg)` |
| `retry` | Exponential-backoff retry helper | `Do(ctx, cfg, fn)`, `DefaultConfig()` |
| `server` | Graceful HTTP server + shutdown manager | `NewGracefulServer(engine, log, port, timeout)`, `NewShutdownManager(log)` |
| `telemetry` | Bootstrap OTel SDK (traces + metrics providers) | `Init(ctx, Config) (*Providers, error)`, `providers.Shutdown(ctx)` |
| `testkit` | Test helpers: containers (Postgres, Redis), JWT factory, HTTP client, assertions | `NewPostgresContainer(t)`, `NewRedisContainer(t)`, `IssueToken(t, userID, role, secret)` |
| `tracing` | OTel tracer factory + no-op fallback | `NewTracer(cfg) (Tracer, error)`, `NewNoOpTracer()`, `tracer.Start(ctx, name)` |

## Generic config pattern

`config.Config[T]` embeds all base infra config and squash-decodes a `service:` YAML block into `T`:

```go
type MyServiceConfig struct {
    MaxItems int `mapstructure:"max_items"`
}

cfg, err := config.Load[MyServiceConfig]("my-service")
// cfg.Base.Database, cfg.Base.JWT, …
// cfg.Service.MaxItems
```

## Bootstrap one-liner

```go
// gRPC service
app, err := bootstrap.NewGRPCService[SvcConfig]("users")

// HTTP gateway
app, err := bootstrap.NewHTTPService[GatewayConfig]("gateway")
conn := app.GRPCConn("users") // cached, traced, registered for graceful close
```

Both constructors wire config → logger → tracing → metrics → DB → Redis → NATS → Asynq and register all components with `ShutdownManager`.
