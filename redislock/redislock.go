// Package redislock provides distributed locking via Redis using bsm/redislock.
package redislock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"

	pkgredis "github.com/piresc/gokit/redis"
)

var (
	// ErrLockUnavailable is returned when the lock cannot be acquired.
	ErrLockUnavailable = errors.New("lock unavailable")
	// ErrNilClient is returned when a nil redis client is passed.
	ErrNilClient = errors.New("redis client must not be nil")
	// ErrLockNotHeld is returned when releasing a lock not held by this owner.
	ErrLockNotHeld = errors.New("lock not held by this owner")
)

// Config holds locker configuration.
type Config struct {
	Prefix        string
	TTL           time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// Locker manages distributed locks.
type Locker struct {
	client *redislock.Client
	cfg    Config
}

// Lock represents a held distributed lock.
type Lock struct {
	inner *redislock.Lock
	key   string
}

// NewLocker creates a Locker backed by the given redis client.
func NewLocker(client *pkgredis.Client, cfg Config) (*Locker, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &Locker{
		client: redislock.New(client.GetClient()),
		cfg:    cfg,
	}, nil
}

// Acquire obtains the lock for key. Returns ErrLockUnavailable if not acquired.
func (l *Locker) Acquire(ctx context.Context, key string) (*Lock, error) {
	lockKey := l.lockKey(key)

	opts := &redislock.Options{}
	if l.cfg.RetryAttempts > 0 && l.cfg.RetryDelay > 0 {
		opts.RetryStrategy = redislock.LimitRetry(
			redislock.LinearBackoff(l.cfg.RetryDelay),
			l.cfg.RetryAttempts,
		)
	}

	lock, err := l.client.Obtain(ctx, lockKey, l.cfg.TTL, opts)
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return nil, ErrLockUnavailable
		}
		return nil, fmt.Errorf("redis error during lock acquisition: %w", err)
	}

	return &Lock{inner: lock, key: lockKey}, nil
}

// Release releases the lock. Returns ErrLockNotHeld if already expired/released.
func (lock *Lock) Release(ctx context.Context) error {
	err := lock.inner.Release(ctx)
	if err != nil {
		if errors.Is(err, redislock.ErrLockNotHeld) {
			return ErrLockNotHeld
		}
		return fmt.Errorf("redis error during lock release: %w", err)
	}
	return nil
}

// Key returns the full Redis key for this lock.
func (lock *Lock) Key() string { return lock.key }

// Value returns the unique token for this lock instance.
func (lock *Lock) Value() string { return lock.inner.Token() }

func (l *Locker) lockKey(key string) string {
	if l.cfg.Prefix == "" {
		return "lock:" + key
	}
	return l.cfg.Prefix + ":lock:" + key
}
