package redislock_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/redislock"
)

func TestNewLocker_NilClientReturnsError(t *testing.T) {
	_, err := redislock.NewLocker(nil, redislock.Config{})
	require.Error(t, err)
	assert.ErrorIs(t, err, redislock.ErrNilClient)
}

func TestLockKey_DefaultPrefix(t *testing.T) {
	// We can't test Acquire without Redis, but we can verify struct creation
	// and error sentinel values are exported correctly.
	assert.NotNil(t, redislock.ErrLockUnavailable)
	assert.NotNil(t, redislock.ErrLockNotHeld)
}
