package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/jmoiron/sqlx"
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

func (p *PostgresClient) GetDB() *sqlx.DB {
	return p.db
}

func (p *PostgresClient) Close() error {
	return p.db.Close()
}

func (p *PostgresClient) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
