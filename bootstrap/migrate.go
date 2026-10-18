package bootstrap

import (
	"embed"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/piresc/gokit/config"
)

// migrationURL builds the postgres URL with properly escaped credentials.
func migrationURL(cfg config.DatabaseConfig) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.Username, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:     "/" + cfg.Database,
		RawQuery: "sslmode=" + url.QueryEscape(cfg.SSLMode),
	}
	return u.String()
}

// RunMigrations runs embedded SQL migrations against the service database on boot.
// Pass the embed.FS and the directory inside it that contains migration files.
//
// Example:
//
//	//go:embed migrations
//	var migFS embed.FS
//
//	bootstrap.RunMigrations(migFS, "migrations", cfg.Database)
func RunMigrations(fsys embed.FS, dir string, cfg config.DatabaseConfig) error {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrationURL(cfg))
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
