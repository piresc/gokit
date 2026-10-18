package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_ShouldReturnError_WhenConnectionFails(t *testing.T) {
	cfg := config.RedisConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	client, err := NewClient(cfg)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestNewClient_ShouldSucceed_WithMiniredis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host:     mr.Host(),
		Port:     int(mr.Server().Addr().Port),
		Password: "",
		DB:       0,
		PoolSize: 5,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer func() { _ = client.Close() }()
}

func TestClient_GetSet_ShouldWork(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	err = client.Set(ctx, "foo", "bar", time.Minute)
	require.NoError(t, err)

	val, err := client.Get(ctx, "foo")
	require.NoError(t, err)
	assert.Equal(t, "bar", val)
}

func TestClient_Delete_ShouldWork(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	_ = client.Set(ctx, "key", "val", time.Minute)
	err = client.Delete(ctx, "key")
	assert.NoError(t, err)
}

func TestClient_SetNX_ShouldWork(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	ok, err := client.SetNX(ctx, "nx-key", "val", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = client.SetNX(ctx, "nx-key", "val2", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestClient_HMSetHGetAll_ShouldWork(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	err = client.HMSet(ctx, "hash", map[string]interface{}{"f1": "v1", "f2": "v2"})
	require.NoError(t, err)

	result, err := client.HGetAll(ctx, "hash")
	require.NoError(t, err)
	assert.Equal(t, "v1", result["f1"])
	assert.Equal(t, "v2", result["f2"])
}

func TestClient_Expire_ShouldWork(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	_ = client.Set(ctx, "exp-key", "val", 0)
	err = client.Expire(ctx, "exp-key", time.Minute)
	assert.NoError(t, err)
}

func TestNewTracedRedisClient_ShouldReturnValidClient(t *testing.T) {
	tracer := tracing.NewNoOpTracer()
	redisClient := &Client{client: nil}

	traced := NewTracedRedisClient(redisClient, tracer)

	require.NotNil(t, traced)
	assert.Equal(t, redisClient, traced.Client)
}

func TestTracedRedisClient_GetClient_ShouldReturnUnderlying(t *testing.T) {
	redisClient := &Client{client: nil}
	assert.Nil(t, redisClient.GetClient())
}
