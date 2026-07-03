package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	sqldir "github.com/julianahrens/balanceraybackend/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations checks and applies all pending database upgrades on startup
func RunMigrations(db *sql.DB) error {
	// We read the embedded FS from the sql package, targeting the subfolder "migrations"
	sourceDriver, err := iofs.New(sqldir.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres instance driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database schema is up to date. No migrations applied.")
			return nil
		}
		return fmt.Errorf("migration run failed: %w", err)
	}

	log.Println("Database migrations successfully executed!")
	return nil
}
