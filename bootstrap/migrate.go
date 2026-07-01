package bootstrap

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/piresc/gokit/config"
)

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

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)

	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
