# gokit

Shared infrastructure primitives for Go services. Install with `go get github.com/piresc/gokit@v0.4.0`.

## Package reference

| Package | What it does | Key API |
|---------|-------------|---------|
| `apperror` | Typed application errors with HTTP status codes | `New`, `BadRequest`, `NotFound`, `Forbidden`, `Conflict`, `Internal`, `ServiceUnavailable` |
| `auth` | HS256 JWT and RS256 sign / verify, Redis token revocation | `GenerateToken(userID, role, cfg)`, `ValidateToken(token, secret)`, `LoadRSAPrivateKey(pem, path)`, `LoadRSAPublicKey(pem, path)`, `NewRSASigner(key, issuer, ttl).Sign(userID, email, role)`, `NewRSAVerifier(key).Verify(token)`, `NewRedisRevocationStore(client)` |
| `bootstrap` | One-call service wiring — config, DB, Redis, NATS, Asynq, tracing, metrics, graceful shutdown | `NewGRPCService[T](name)`, `NewHTTPService[T](name)`, `app.GRPCConn(name)`, `app.Run()`, `app.Shutdown(ctx)` |
| `circuitbreaker` | Sony gobreaker wrapper | `New(Config)`, `cb.Execute(fn)` |
| `config` | Generic Viper-based config loader; merges YAML + env vars | `Load[T](serviceName)`, `Config[T]{Base, Service T}` |
| `database` | pgx/sqlx Postgres client with tracing | `NewPostgresClient(cfg)`, `client.GetDB()`, `client.Ping(ctx)`, `client.Close()` |
| `geo` | Great-circle distance between coordinates | `Point{Lat, Lon}`, `DistanceKm(a, b)`, `DistanceM(a, b)` |
| `grpcclient` | Dial a gRPC backend with OTel tracing | `Dial(ctx, ClientConfig{Target, Tracer, Logger})` |
| `grpcerror` | Map `apperror` → gRPC status codes | `MapError(err) error` |
| `grpcmiddleware` | Unary + stream interceptors: auth, logging, tracing, recovery, rate-limit | `NewInterceptors(jwtSecret, log, tracer, redisClient)` |
| `grpcserver` | gRPC server with graceful shutdown | `New(log, port, timeout, opts...)`, `server.Start()`, `server.GracefulStop()` |
| `health` | Liveness / readiness HTTP endpoints + pluggable checkers | `NewService(log)`, `svc.AddChecker(name, checker)`, `RegisterRoutes(mux, name, version, svc)` |
| `httperror` | Write `apperror` as a JSON HTTP response | `Write(w, err)`, `WriteWithRequestID(w, err, reqID)` |
| `httpclient` | HTTP client with timeout, optional tracing and JSON helpers | `NewClient(Config{BaseURL, APIKey, Timeout, Tracer})`, `c.Do(ctx, method, endpoint, body)`, `c.GetJSON(ctx, endpoint, &out)`, `c.PostJSON(ctx, endpoint, body, &out)` |
| `idempotency` | Lookup-first idempotency check | `Check[T](ctx, key, lookupFn, errNotFound, operation)` |
| `logger` | `slog.Logger` factory (JSON or text, configurable level) | `NewLogger(cfg)`, `Err(err)` |
| `mail` | SMTP mail sender with provider abstraction | `NewSender(cfg) Sender`, `sender.Send(ctx, to, subject, body)` |
| `metrics` | OTel metrics — HTTP middleware + gRPC interceptor + Prometheus server | `NewMetrics(service, otlpEndpoint)`, `m.HTTPMiddleware()`, `m.GRPCUnaryInterceptor()` |
| `middleware` | `net/http` middleware: JWT / RS256 auth, CORS, rate-limit, recovery, security headers, tracing | `NewMiddleware(log, tracer)`, `mw.JWTAuth(secret)`, `mw.RS256Auth(verifier, RS256Options{Revocation, FailOpen})`, `GetClaims(r)`, `mw.RequireRole(roles...)`, `mw.CorsHandler(origins)` |
| `nats` | NATS JetStream client + typed publisher | `NewClient(url)`, `NewPublisher(client)`, `publisher.Publish(ctx, subject, data, msgID)` |
| `ratelimit` | Token-bucket rate-limit store (used by middleware) | `NewStore(cfg)`, `store.Allow(key)` |
| `redis` | go-redis v9 client with OTel tracing | `NewClient(cfg)`, `client.GetClient()`, `client.Close()` |
| `redislock` | Distributed lock via Redis | `NewLocker(client, cfg)`, `locker.Acquire(ctx, key) (Lock, error)`, `lock.Release(ctx)` |
| `response` | Standard JSON response envelope | `Success(w, status, data)`, `Error(w, status, msg)`, `ErrorWithRequestID(w, status, msg, reqID)` |
| `retry` | Exponential-backoff retry helper | `Do(ctx, cfg, fn)`, `DefaultConfig()` |
| `semver` | Semantic version parse and compare | `Parse(v)`, `MustParse(v)`, `Compare(v1, v2)`, `LessThan(v1, v2)`, `GreaterThanOrEqual(v1, v2)` |
| `server` | Graceful HTTP server + shutdown manager | `NewGracefulServer(handler, log, port, timeout)`, `NewShutdownManager(log)` |
| `telemetry` | Bootstrap OTel SDK (traces + metrics providers) | `Init(ctx, Config) (*Providers, error)`, `providers.Shutdown(ctx)` |
| `testkit` | Test helpers: containers (Postgres, Redis), JWT / RSA key factories, HTTP client, assertions | `NewPostgresContainer(ctx, migrationFS, dir)`, `NewRedisContainer(ctx)`, `GenerateToken(userID, role, secret)`, `NewRSAKeyPair(t)` |
| `tracing` | OTel tracer factory + no-op fallback | `NewTracer(cfg) (Tracer, error)`, `NewNoOpTracer()` |

## RS256 auth

The auth service signs tokens with a private key. Other services verify with the public key and can check a Redis revocation list.

Env vars: `JWT_PRIVATE_KEY_PEM` / `JWT_PRIVATE_KEY_PATH`, `JWT_PUBLIC_KEY_PEM` / `JWT_PUBLIC_KEY_PATH`, `JWT_ISSUER`, `ACCESS_TOKEN_TTL` (15m), `REFRESH_TOKEN_TTL` (720h).

```go
// Auth service: sign
priv, err := auth.LoadRSAPrivateKey(cfg.Auth.PrivateKeyPEM, cfg.Auth.PrivateKeyPath)
signer := auth.NewRSASigner(priv, cfg.Auth.Issuer, cfg.Auth.AccessTTL)
token, jti, expiresAt, err := signer.Sign(userID, email, role)

// Any service: verify + revocation
pub, err := auth.LoadRSAPublicKey(cfg.Auth.PublicKeyPEM, cfg.Auth.PublicKeyPath)
verifier := auth.NewRSAVerifier(pub)
store := auth.NewRedisRevocationStore(redisClient) // redisClient is *redis.Client from gokit/redis

mw := middleware.NewMiddleware(log, tracer)
protected := mw.RS256Auth(verifier, middleware.RS256Options{Revocation: store})(handler)

// In a handler, after RS256Auth
claims, ok := middleware.GetClaims(r)

// Logout: revoke until the token would have expired
err = store.Revoke(ctx, jti, time.Until(expiresAt))
```

If the Redis revocation check errors, the request gets 503 with `Retry-After: 5`. Set `RS256Options{FailOpen: true}` to let such requests through.

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
