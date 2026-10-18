package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers postgres container with a sqlx.DB connection.
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	DB        *sqlx.DB
}

// NewPostgresContainer creates and starts a testcontainers postgres container
// and connects to it via sqlx. The caller is responsible for running migrations.
// Returns the container and connection, or an error if setup fails.
//
// Example:
//
//	container, err := NewPostgresContainer(ctx, nil, "")
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer container.Close(ctx)
//	// Run migrations on container.DB
func NewPostgresContainer(ctx context.Context, migrationFS interface{}, dir string) (*PostgresContainer, error) {
	// Note: migrationFS and dir parameters are kept for API compatibility but not used.
	// Callers should run migrations explicitly on the returned DB connection.

	// Create postgres container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("get postgres connection string: %w", err)
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	return &PostgresContainer{
		Container: pgContainer,
		DB:        db,
	}, nil
}

// Close terminates the container and closes the database connection.
func (pc *PostgresContainer) Close(ctx context.Context) error {
	if pc.DB != nil {
		_ = pc.DB.Close()
	}
	if pc.Container != nil {
		return pc.Container.Terminate(ctx)
	}
	return nil
}

// RedisContainer wraps a testcontainers redis container with connection details.
type RedisContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
}

// NewRedisContainer creates and starts a testcontainers redis container.
// Returns the container with host and port, or an error if setup fails.
//
// Example:
//
//	container, err := NewRedisContainer(ctx)
//	defer container.Close(ctx)
//	// Use container.Host and container.Port to connect
func NewRedisContainer(ctx context.Context) (*RedisContainer, error) {
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start redis container: %w", err)
	}

	host, err := redisContainer.Host(ctx)
	if err != nil {
		redisContainer.Terminate(ctx)
		return nil, fmt.Errorf("get redis host: %w", err)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		redisContainer.Terminate(ctx)
		return nil, fmt.Errorf("get redis port: %w", err)
	}

	return &RedisContainer{
		Container: redisContainer,
		Host:      host,
		Port:      port.Port(),
	}, nil
}

// Close terminates the container.
func (rc *RedisContainer) Close(ctx context.Context) error {
	return rc.Container.Terminate(ctx)
}

// ConnectionString returns the redis connection string suitable for most redis clients.
func (rc *RedisContainer) ConnectionString() string {
	return fmt.Sprintf("%s:%s", rc.Host, rc.Port)
}
