package ratelimit_test

import (
	"testing"
	"time"

	"github.com/piresc/gokit/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := ratelimit.DefaultConfig()
	assert.Greater(t, cfg.RequestsPerSecond, 0)
	assert.Greater(t, cfg.BurstSize, 0)
}

func TestStore_Allow_UnderLimit(t *testing.T) {
	store := ratelimit.NewStore(ratelimit.Config{
		RequestsPerSecond: 10,
		BurstSize:         5,
	})
	defer store.Stop()

	for i := 0; i < 5; i++ {
		assert.True(t, store.Allow("client-a"), "request %d should be allowed", i+1)
	}
}

func TestStore_Allow_OverLimit(t *testing.T) {
	store := ratelimit.NewStore(ratelimit.Config{
		RequestsPerSecond: 1,
		BurstSize:         1,
	})
	defer store.Stop()

	assert.True(t, store.Allow("x"))
	// Burst exhausted — next request should be rejected.
	assert.False(t, store.Allow("x"))
}

func TestStore_Allow_PerKey(t *testing.T) {
	store := ratelimit.NewStore(ratelimit.Config{
		RequestsPerSecond: 1,
		BurstSize:         1,
	})
	defer store.Stop()

	assert.True(t, store.Allow("a"))
	assert.False(t, store.Allow("a")) // a exhausted
	assert.True(t, store.Allow("b")) // b has fresh bucket
}

func TestStore_Cleanup(t *testing.T) {
	store := ratelimit.NewStore(ratelimit.Config{
		RequestsPerSecond: 10,
		BurstSize:         10,
		CleanupInterval:   10 * time.Millisecond,
	})
	defer store.Stop()

	store.Allow("cleanup-key")
	// After 2x the cleanup interval the entry should have been evicted.
	time.Sleep(30 * time.Millisecond)
	// A fresh Allow call after eviction should succeed (new bucket).
	assert.True(t, store.Allow("cleanup-key"))
}
