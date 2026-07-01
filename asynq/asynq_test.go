package asynq_test

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goasynq "github.com/piresc/gokit/asynq"
)

func TestNewClient_CreatesClient(t *testing.T) {
	c := goasynq.NewClient("localhost:6379", "")
	require.NotNil(t, c)
	// Close should not panic (may error if no Redis, that's OK).
	_ = c.Close()
}

func TestNewServer_CreatesServer(t *testing.T) {
	s := goasynq.NewServer("localhost:6379", "", 5)
	require.NotNil(t, s)
}

func TestNewServer_DefaultConcurrency(t *testing.T) {
	s := goasynq.NewServer("localhost:6379", "", 0)
	require.NotNil(t, s)
}

func TestServer_Register(t *testing.T) {
	s := goasynq.NewServer("localhost:6379", "", 2)
	// Register with a real handler should not panic.
	assert.NotPanics(t, func() {
		s.Register("email:welcome", asynq.HandlerFunc(func(_ context.Context, _ *asynq.Task) error {
			return nil
		}))
	})
}
