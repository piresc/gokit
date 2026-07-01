package testkit

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TruncateTables truncates each of the specified tables using TRUNCATE TABLE ... CASCADE.
// Each table is truncated in the order provided, so callers should pass tables in
// dependency order (children before parents) to respect foreign-key constraints.
//
// Example:
//   TruncateTables(db, "presence_logs", "reservations", "drivers")
func TruncateTables(db *sqlx.DB, tables ...string) error {
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}
