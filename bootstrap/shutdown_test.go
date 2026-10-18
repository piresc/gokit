package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/server"
)

const shutdownTestYAML = `
app:
  name: test-svc
  environment: local
  version: v0.0.1
server:
  port: 0
  shutdown_timeout: 1
logger:
  level: info
  format: text
tracing:
  enabled: false
  sample_rate: 0.0
grpc:
  server:
    port: 0
    shutdown_timeout: 1s
    request_timeout: 5s
`

type shutdownSvc struct{}

func setupShutdownConfig(t *testing.T) {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "config", "local")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "testsvc.yaml"), []byte(shutdownTestYAML), 0o644))

	require.NoError(t, os.Chdir(dir))
	t.Setenv("APP_ENV", "local")
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
}

func newTestBase(order *[]string) *baseApp {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := &baseApp{log: log, shutdownMgr: server.NewShutdownManager(log)}
	b.shutdownMgr.Register(func(context.Context) error {
		*order = append(*order, "infra")
		return nil
	})
	return b
}

func TestShutdown_HooksRunBeforeInfraInReverse(t *testing.T) {
	var order []string
	b := newTestBase(&order)
	b.onShutdown(func(context.Context) error { order = append(order, "h1"); return nil })
	b.onShutdown(func(context.Context) error { order = append(order, "h2"); return nil })

	require.NoError(t, b.shutdown(context.Background()))
	assert.Equal(t, []string{"h2", "h1", "infra"}, order)
}

func TestShutdown_HookErrorDoesNotStopOthers(t *testing.T) {
	errA := errors.New("a failed")
	var order []string
	b := newTestBase(&order)
	b.onShutdown(func(context.Context) error { order = append(order, "h1"); return errA })
	b.onShutdown(func(context.Context) error { order = append(order, "h2"); return nil })

	err := b.shutdown(context.Background())
	assert.Equal(t, []string{"h2", "h1", "infra"}, order)
	assert.ErrorIs(t, err, errA)
}

func TestHTTPApp_OnShutdown(t *testing.T) {
	setupShutdownConfig(t)

	app, err := NewHTTPService[shutdownSvc]("testsvc")
	require.NoError(t, err)

	calls := 0
	app.OnShutdown(func(context.Context) error { calls++; return nil })

	require.NoError(t, app.Shutdown(context.Background()))
	assert.Equal(t, 1, calls)
}

func TestGRPCApp_OnShutdown(t *testing.T) {
	setupShutdownConfig(t)

	app, err := NewGRPCService[shutdownSvc]("testsvc")
	require.NoError(t, err)

	calls := 0
	app.OnShutdown(func(context.Context) error { calls++; return nil })

	require.NoError(t, app.Shutdown(context.Background()))
	assert.Equal(t, 1, calls)
}
