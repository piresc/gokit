package server_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/piresc/gokit/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

func TestNewGracefulServer_NotNil(t *testing.T) {
	engine := gin.New()
	srv := server.NewGracefulServer(engine, slog.Default(), 8080, 0)
	require.NotNil(t, srv)
}

func TestShutdownManager_NoFunctions(t *testing.T) {
	sm := server.NewShutdownManager(slog.Default())
	err := sm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestShutdownManager_AllSucceed(t *testing.T) {
	sm := server.NewShutdownManager(slog.Default())
	called := 0
	for i := 0; i < 3; i++ {
		sm.Register(func(_ context.Context) error { called++; return nil })
	}
	require.NoError(t, sm.Shutdown(context.Background()))
	assert.Equal(t, 3, called)
}

func TestShutdownManager_FirstErrorReturned(t *testing.T) {
	sm := server.NewShutdownManager(slog.Default())
	boom := errors.New("boom")
	sm.Register(func(_ context.Context) error { return boom })
	sm.Register(func(_ context.Context) error { return nil }) // still runs
	err := sm.Shutdown(context.Background())
	assert.ErrorIs(t, err, boom)
}

func TestShutdownManager_CancelledContext(t *testing.T) {
	sm := server.NewShutdownManager(slog.Default())
	sm.Register(func(_ context.Context) error { return nil })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := sm.Shutdown(ctx)
	assert.Error(t, err)
}
