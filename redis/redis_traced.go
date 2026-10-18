package redis

import (
	"context"
	"time"

	"github.com/piresc/gokit/tracing"
)

// TracedRedisClient wraps Client.
//
// Deprecated: since v0.5.0 the clients from NewPostgresClient / NewClient create spans themselves.
type TracedRedisClient struct {
	*Client
	tracer tracing.Tracer
}

// NewTracedRedisClient wraps client. The tracer is kept for API compatibility;
// spans are created by the client's own hook.
//
// Deprecated: since v0.5.0 the clients from NewPostgresClient / NewClient create spans themselves.
func NewTracedRedisClient(client *Client, tracer tracing.Tracer) *TracedRedisClient {
	return &TracedRedisClient{Client: client, tracer: tracer}
}

func (r *TracedRedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key)
}

func (r *TracedRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration)
}

func (r *TracedRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, key, value, expiration)
}

func (r *TracedRedisClient) Delete(ctx context.Context, key string) error {
	return r.Client.Delete(ctx, key)
}

func (r *TracedRedisClient) Close() error {
	return r.Client.Close()
}
