package auth

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/piresc/gokit/redis"
)

// RevocationKeyPrefix is the Redis key prefix for revoked token IDs.
const RevocationKeyPrefix = "revoked:"

// RevocationChecker reports whether a token ID (jti) has been revoked.
type RevocationChecker interface {
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// RedisRevocationStore stores revoked token IDs in Redis.
type RedisRevocationStore struct {
	client *goredis.Client
}

var _ RevocationChecker = (*RedisRevocationStore)(nil)

// NewRedisRevocationStore creates a store backed by the given gokit Redis client.
func NewRedisRevocationStore(client *redis.Client) *RedisRevocationStore {
	return &RedisRevocationStore{client: client.GetClient()}
}

// Revoke marks jti as revoked for ttl. A non-positive ttl or empty jti is a no-op.
func (s *RedisRevocationStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	if err := s.client.Set(ctx, RevocationKeyPrefix+jti, "1", ttl).Err(); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// IsRevoked reports whether jti is currently revoked.
func (s *RedisRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := s.client.Exists(ctx, RevocationKeyPrefix+jti).Result()
	if err != nil {
		return false, fmt.Errorf("check token revocation: %w", err)
	}
	return n > 0, nil
}
