package health

import (
	"context"
	"fmt"

	"github.com/piresc/gokit/database"
	"github.com/piresc/gokit/redis"
)

const (
	CheckerNamePostgres = "postgres"
	CheckerNameRedis    = "redis"
)

// postgresChecker pings a PostgresClient.
type postgresChecker struct {
	db *database.PostgresClient
}

// NewPostgresChecker returns a Checker that pings the given PostgresClient.
func NewPostgresChecker(db *database.PostgresClient) Checker {
	return &postgresChecker{db: db}
}

func (c *postgresChecker) Check(ctx context.Context) error {
	if err := c.db.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}

func (c *postgresChecker) Name() string { return CheckerNamePostgres }

// redisChecker pings a Redis Client.
type redisChecker struct {
	redis *redis.Client
}

// NewRedisChecker returns a Checker that pings the given Redis Client.
func NewRedisChecker(rc *redis.Client) Checker {
	return &redisChecker{redis: rc}
}

func (c *redisChecker) Check(ctx context.Context) error {
	if _, err := c.redis.GetClient().Ping(ctx).Result(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

func (c *redisChecker) Name() string { return CheckerNameRedis }
