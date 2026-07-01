// Package retry provides exponential-backoff retry logic.
package retry

import (
	"context"
	"fmt"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxAttempts int
	Delay       time.Duration
	MaxDelay    time.Duration
}

// DefaultConfig returns a sensible default retry config.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}
}

// Do executes fn up to cfg.MaxAttempts times with exponential backoff.
// Returns ctx.Err() if context is cancelled, or a wrapped last error on exhaustion.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		delay := cfg.Delay * (1 << attempt) // exponential: 1x, 2x, 4x, ...
		if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
