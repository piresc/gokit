package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/piresc/gokit/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalYAML returns a minimal valid config with no external services.
// DB/Redis/NATS all absent → those fields stay nil after construction.
const minimalYAML = `
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

// setupConfig writes a minimal YAML into a temp dir and chdirs into it.
// Returns a cleanup func that restores the original working directory.
func setupConfig(t *testing.T, name string) func() {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "config", "local")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, name+".yaml"), []byte(minimalYAML), 0o644))

	require.NoError(t, os.Chdir(dir))
	t.Setenv("APP_ENV", "local")

	return func() {
		os.Chdir(orig) //nolint:errcheck
	}
}

// --- GRPCApp tests ---

type emptySvc struct{}

func TestNewGRPCService_MinimalConfig_NoPanic(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewGRPCService[emptySvc]("testsvc")
	require.NoError(t, err)
	require.NotNil(t, app)
}

func TestNewGRPCService_NilDBWhenUnconfigured(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewGRPCService[emptySvc]("testsvc")
	require.NoError(t, err)
	assert.Nil(t, app.DB, "DB should be nil when database.host absent")
	assert.Nil(t, app.Redis, "Redis should be nil when redis.host absent")
	assert.Nil(t, app.NATSClient, "NATSClient should be nil when nats.enabled false")
}

func TestNewGRPCService_MailSenderIsLogSender(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewGRPCService[emptySvc]("testsvc")
	require.NoError(t, err)
	// No mail.provider → LogSender; just verify non-nil
	require.NotNil(t, app.MailSender)
}

func TestGRPCApp_ShutdownClean(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewGRPCService[emptySvc]("testsvc")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5)
	defer cancel()
	err = app.Shutdown(ctx)
	assert.NoError(t, err)
}

// --- HTTPApp tests ---

func TestNewHTTPService_MinimalConfig_NoPanic(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewHTTPService[emptySvc]("testsvc")
	require.NoError(t, err)
	require.NotNil(t, app)
}

func TestNewHTTPService_EngineNonNil(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewHTTPService[emptySvc]("testsvc")
	require.NoError(t, err)
	assert.NotNil(t, app.Engine)
}

func TestHTTPApp_NilDBWhenUnconfigured(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewHTTPService[emptySvc]("testsvc")
	require.NoError(t, err)
	assert.Nil(t, app.DB)
	assert.Nil(t, app.Redis)
}

func TestHTTPApp_GRPCConn_PanicsOnMissingName(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewHTTPService[emptySvc]("testsvc")
	require.NoError(t, err)

	assert.Panics(t, func() {
		app.GRPCConn("nonexistent-service")
	})
}

func TestHTTPApp_ShutdownClean(t *testing.T) {
	restore := setupConfig(t, "testsvc")
	defer restore()

	app, err := bootstrap.NewHTTPService[emptySvc]("testsvc")
	require.NoError(t, err)

	err = app.Shutdown(context.Background())
	assert.NoError(t, err)
}
