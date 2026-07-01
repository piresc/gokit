package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/retry"
)

var errTemp = errors.New("temporary error")

func TestDo_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessAfterRetry(t *testing.T) {
	calls := 0
	cfg := retry.Config{MaxAttempts: 3, Delay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	err := retry.Do(context.Background(), cfg, func() error {
		calls++
		if calls < 3 {
			return errTemp
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	cfg := retry.Config{MaxAttempts: 3, Delay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	err := retry.Do(context.Background(), cfg, func() error { return errTemp })
	require.Error(t, err)
	assert.ErrorIs(t, err, errTemp)
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := retry.Do(ctx, retry.DefaultConfig(), func() error { return errTemp })
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDo_ContextCancelledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cfg := retry.Config{MaxAttempts: 10, Delay: 100 * time.Millisecond, MaxDelay: 1 * time.Second}
	err := retry.Do(ctx, cfg, func() error { return errTemp })
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
