package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Migrator handles database migrations using golang-migrate
type Migrator struct {
	migrate *migrate.Migrate
	db      *sql.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(cfg *config.PostgresConfig, migrationPath string) (*Migrator, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database for migration: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		cfg.Database,
		driver,
	)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return &Migrator{
		migrate: m,
		db:      db,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	slog.InfoContext(ctx, "running database migrations up")
	if err := m.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations up: %w", err)
	}
	slog.InfoContext(ctx, "database migrations completed")
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down(ctx context.Context) error {
	slog.InfoContext(ctx, "rolling back database migrations")
	if err := m.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations down: %w", err)
	}
	slog.InfoContext(ctx, "database migrations rolled back")
	return nil
}

// Steps runs N migrations up (positive) or down (negative)
func (m *Migrator) Steps(ctx context.Context, n int) error {
	slog.InfoContext(ctx, "running migration steps", slog.Int("steps", n))
	if err := m.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migration steps: %w", err)
	}
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version(ctx context.Context) (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("get migration version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(ctx context.Context, version int) error {
	slog.WarnContext(ctx, "forcing migration version", slog.Int("version", version))
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("force migration version: %w", err)
	}
	return nil
}

// Close closes the migrator and underlying database connection
func (m *Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// RunMigrations is a convenience function to run migrations from config
func RunMigrations(ctx context.Context, cfg *config.PostgresConfig, migrationPath string) error {
	migrator, err := NewMigrator(cfg, migrationPath)
	if err != nil {
		return err
	}
	defer migrator.Close()

	return migrator.Up(ctx)
}
