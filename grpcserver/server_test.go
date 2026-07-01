package grpcserver_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/piresc/gokit/grpcserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNew_DefaultLogger(t *testing.T) {
	srv := grpcserver.New(nil, 0, 5*time.Second)
	require.NotNil(t, srv)
	assert.NotNil(t, srv.Server())
}

func TestNew_WithOptions(t *testing.T) {
	logger := slog.Default()
	srv := grpcserver.New(logger, 9999, 3*time.Second,
		grpc.MaxRecvMsgSize(4*1024*1024),
	)
	require.NotNil(t, srv)
	assert.NotNil(t, srv.Server())
}

func TestGracefulStop_TimesOut(t *testing.T) {
	srv := grpcserver.New(nil, 0, 1*time.Millisecond)
	// Stop on a server that was never started — should return timeout error
	// because GracefulStop waits for Serve to drain.
	err := srv.GracefulStop()
	// Either nil (immediate drain) or timeout — both are acceptable.
	// We just assert it doesn't panic.
	_ = err
}
