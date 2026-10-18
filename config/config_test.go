package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/piresc/gokit/config"
)

type TestServiceConfig struct {
	MaxItems int `mapstructure:"max_items"`
}

func TestLoad_LocalYAML(t *testing.T) {
	dir := t.TempDir()
	localDir := filepath.Join(dir, "config", "local")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yaml := `
app:
  name: test-service
  environment: local
server:
  port: 8080
logger:
  level: info
  format: json
jwt:
  secret: "test-secret-that-is-at-least-32-chars-long"
  expiration: 3600
service:
  max_items: 42
`
	if err := os.WriteFile(filepath.Join(localDir, "test.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long")

	cfg, err := config.LoadFrom[TestServiceConfig]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.App.Name != "test-service" {
		t.Errorf("want test-service, got %s", cfg.App.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("want 8080, got %d", cfg.Server.Port)
	}
	if cfg.Service.MaxItems != 42 {
		t.Errorf("want 42, got %d", cfg.Service.MaxItems)
	}
}

func TestLoad_EnvOverridesSecrets(t *testing.T) {
	dir := t.TempDir()
	localDir := filepath.Join(dir, "config", "local")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yaml := `
app:
  name: test
  environment: local
server:
  port: 8080
logger:
  level: info
jwt:
  secret: "placeholder"
database:
  host: yaml-host
`
	if err := os.WriteFile(filepath.Join(localDir, "test.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("DB_HOST", "env-host")
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long")

	type Empty struct{}
	cfg, err := config.LoadFrom[Empty]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.Database.Host != "env-host" {
		t.Errorf("want env-host, got %s", cfg.Database.Host)
	}
}

func writeAuthTestConfig(t *testing.T, env, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	envDir := filepath.Join(dir, "config", env)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "test.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoad_AuthDefaults(t *testing.T) {
	dir := writeAuthTestConfig(t, "production", `
app:
  name: test
  environment: production
server:
  port: 8080
`)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.LoadFrom[TestServiceConfig]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.Auth.AccessTTL != 15*time.Minute {
		t.Errorf("want 15m, got %v", cfg.Auth.AccessTTL)
	}
	if cfg.Auth.RefreshTTL != 720*time.Hour {
		t.Errorf("want 720h, got %v", cfg.Auth.RefreshTTL)
	}
}

func TestLoad_AuthFromEnv(t *testing.T) {
	dir := writeAuthTestConfig(t, "local", `
app:
  name: test
  environment: local
server:
  port: 8080
`)
	t.Setenv("APP_ENV", "local")
	t.Setenv("JWT_PRIVATE_KEY_PEM", "priv-pem")
	t.Setenv("JWT_PRIVATE_KEY_PATH", "/keys/priv.pem")
	t.Setenv("JWT_PUBLIC_KEY_PEM", "pub-pem")
	t.Setenv("JWT_PUBLIC_KEY_PATH", "/keys/pub.pem")
	t.Setenv("JWT_ISSUER", "env-issuer")
	t.Setenv("ACCESS_TOKEN_TTL", "5m")
	t.Setenv("REFRESH_TOKEN_TTL", "48h")

	cfg, err := config.LoadFrom[TestServiceConfig]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	a := cfg.Auth
	if a.PrivateKeyPEM != "priv-pem" || a.PrivateKeyPath != "/keys/priv.pem" ||
		a.PublicKeyPEM != "pub-pem" || a.PublicKeyPath != "/keys/pub.pem" ||
		a.Issuer != "env-issuer" {
		t.Errorf("unexpected auth config: %+v", a)
	}
	if a.AccessTTL != 5*time.Minute {
		t.Errorf("want 5m, got %v", a.AccessTTL)
	}
	if a.RefreshTTL != 48*time.Hour {
		t.Errorf("want 48h, got %v", a.RefreshTTL)
	}
}

func TestLoad_AuthFromYAML(t *testing.T) {
	dir := writeAuthTestConfig(t, "local", `
app:
  name: test
  environment: local
server:
  port: 8080
auth:
  issuer: user-service
  access_ttl: 10m
`)
	t.Setenv("APP_ENV", "local")

	cfg, err := config.LoadFrom[TestServiceConfig]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.Auth.Issuer != "user-service" {
		t.Errorf("want user-service, got %s", cfg.Auth.Issuer)
	}
	if cfg.Auth.AccessTTL != 10*time.Minute {
		t.Errorf("want 10m, got %v", cfg.Auth.AccessTTL)
	}
	if cfg.Auth.RefreshTTL != 720*time.Hour {
		t.Errorf("want 720h, got %v", cfg.Auth.RefreshTTL)
	}
}
