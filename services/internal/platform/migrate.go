package platform

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations は embed.FS からマイグレーションを実行する.
func RunMigrations(databaseURL string, migrationsFS fs.FS) error {
	dbURL := strings.Replace(databaseURL, "postgresql://", "pgx5://", 1)
	dbURL = strings.Replace(dbURL, "postgres://", "pgx5://", 1)

	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
