package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/piresc/gokit/config"
)

type PostgresClient struct {
	db *sqlx.DB
}

const pingTimeout = 10 * time.Second

func NewPostgresClient(cfg config.DatabaseConfig) (*PostgresClient, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.PathEscape(cfg.Username),
		url.PathEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	rawDB, err := otelsql.Open("pgx", dsn,
		otelsql.WithAttributes(semconv.DBSystemNamePostgreSQL),
		otelsql.WithTracerProvider(redactingTracerProvider{}),
		otelsql.WithSpanErrorAttributesGetter(sqlStateAttributes),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
			OmitRows:             true,
			OmitConnectorConnect: true,
			SpanFilter:           hasParentSpan,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}
	db := sqlx.NewDb(rawDB, "pgx")

	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
	} else {
		db.SetMaxOpenConns(25)
	}
	if cfg.IdleConns > 0 {
		db.SetMaxIdleConns(cfg.IdleConns)
	} else {
		db.SetMaxIdleConns(5)
	}
	if cfg.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute)
	} else {
		db.SetConnMaxLifetime(30 * time.Minute)
	}
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to ping postgres: %w", err), db.Close())
	}

	return &PostgresClient{db: db}, nil
}

// hasParentSpan limits span creation to calls whose ctx already holds a span.
func hasParentSpan(ctx context.Context, _ otelsql.Method, _ string, _ []driver.NamedValue) bool {
	return trace.SpanContextFromContext(ctx).IsValid()
}

// sqlStateAttributes exposes only the SQLSTATE of a Postgres error. The error
// message can quote user-supplied values, so it never reaches a span.
func sqlStateAttributes(err error) []attribute.KeyValue {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code != "" {
		return []attribute.KeyValue{semconv.DBResponseStatusCode(pgErr.Code)}
	}
	return nil
}

// redactingTracerProvider wraps the global tracer provider so spans created by
// otelsql mark failures as Error without recording the error as an exception
// event. otelsql's SpanOptions.RecordError cannot do this: returning false
// skips both the event and the Error status.
type redactingTracerProvider struct {
	trace.TracerProvider // satisfies the sealed interface; Tracer is overridden below
}

func (redactingTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return redactingTracer{Tracer: otel.GetTracerProvider().Tracer(name, opts...)}
}

type redactingTracer struct {
	trace.Tracer
}

func (t redactingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := t.Tracer.Start(ctx, name, opts...)
	return ctx, redactingSpan{Span: span}
}

type redactingSpan struct {
	trace.Span
}

// RecordError drops the error text and only flags the span as failed.
func (s redactingSpan) RecordError(error, ...trace.EventOption) {
	s.SetStatus(codes.Error, "")
}

func (p *PostgresClient) GetDB() *sqlx.DB {
	return p.db
}

func (p *PostgresClient) Close() error {
	return p.db.Close()
}

func (p *PostgresClient) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
