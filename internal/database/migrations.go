package database

import (
	"fmt"
	"log/slog"

	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver for migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file source driver for migrate
)

// RunMigrations runs all pending SQL migrations from the given directory.
func RunMigrations(databaseURL, migrationsPath string, logger *slog.Logger) error {
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Warn("migrate close source error", "error", srcErr)
		}
		if dbErr != nil {
			logger.Warn("migrate close db error", "error", dbErr)
		}
	}()

	logger.Info("running database migrations", "source", migrationsPath)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	logger.Info("migrations complete", "version", version, "dirty", dirty)

	return nil
}
