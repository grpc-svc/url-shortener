package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/lib/logger/slogcute"

	"github.com/golang-migrate/migrate/v4"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	directionUp   = "up"
	directionDown = "down"
)

func main() {
	var direction string

	flag.StringVar(&direction, "direction", directionUp, "Direction to migrate (up or down)")
	cfg := config.MustLoad()

	// Setup logger
	log := setupLogger()

	log.Info("starting migrator",
		slog.String("env", cfg.Env),
		slog.String("storage_path", cfg.StoragePath),
		slog.String("migrations_path", cfg.Migrations.MigrationsPath),
		slog.String("migration_table", cfg.Migrations.MigrationTable),
		slog.String("direction", direction),
	)

	// Validate direction
	if err := validateDirection(direction); err != nil {
		log.Error("invalid direction", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Run migrations
	if err := runMigrations(log, cfg.StoragePath, cfg.Migrations.MigrationsPath, cfg.Migrations.MigrationTable, direction); err != nil {
		log.Error("migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("migrations completed successfully")
}

func setupLogger() *slog.Logger {
	opts := slogcute.CuteHandlerOptions{
		SlogOptions: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewCuteHandler(os.Stdout)

	return slog.New(handler)
}

func validateDirection(direction string) error {
	if direction != directionUp && direction != directionDown {
		return fmt.Errorf("invalid direction '%s', must be 'up' or 'down'", direction)
	}
	return nil
}

func runMigrations(log *slog.Logger, storagePath, migrationsPath, migrationTable, direction string) error {
	// Construct migration source and database URLs
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	databaseURL := fmt.Sprintf("sqlite3://%s?x-migrations-table=%s", storagePath, migrationTable)

	log.Info("initializing migrator",
		slog.String("source", sourceURL),
		slog.String("database", databaseURL),
	)

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Error("failed to close migration source", slog.String("error", sourceErr.Error()))
		}
		if dbErr != nil {
			log.Error("failed to close database", slog.String("error", dbErr.Error()))
		}
	}()

	// Apply migrations based on direction
	switch direction {
	case directionUp:
		log.Info("applying migrations up")
		err = m.Up()
	case directionDown:
		log.Info("applying migrations down")
		err = m.Down()
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
