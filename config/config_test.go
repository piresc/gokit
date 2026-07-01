package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piresc/gokit/config"
)

type TestServiceConfig struct {
	MaxItems int `mapstructure:"max_items"`
}

func TestLoad_LocalYAML(t *testing.T) {
	dir := t.TempDir()
	localDir := filepath.Join(dir, "config", "local")
	os.MkdirAll(localDir, 0o755)

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
	os.WriteFile(filepath.Join(localDir, "test.yaml"), []byte(yaml), 0o644)
	os.Setenv("APP_ENV", "local")
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long")
	defer os.Unsetenv("APP_ENV")
	defer os.Unsetenv("JWT_SECRET")

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
	os.MkdirAll(localDir, 0o755)

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
	os.WriteFile(filepath.Join(localDir, "test.yaml"), []byte(yaml), 0o644)
	os.Setenv("APP_ENV", "local")
	os.Setenv("DB_HOST", "env-host")
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long")
	defer os.Unsetenv("APP_ENV")
	defer os.Unsetenv("DB_HOST")
	defer os.Unsetenv("JWT_SECRET")

	type Empty struct{}
	cfg, err := config.LoadFrom[Empty]("test", dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.Database.Host != "env-host" {
		t.Errorf("want env-host, got %s", cfg.Database.Host)
	}
}
