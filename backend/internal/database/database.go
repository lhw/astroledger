package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver ("pgx")
	"github.com/lhw/astroledger/migrations"
	pgmigrations "github.com/lhw/astroledger/migrations/postgres"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // SQLite driver ("sqlite")
)

// Open opens the SQLite database at path, applies WAL and foreign-key pragmas,
// and runs any pending goose migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := applyPragmas(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	if err := runMigrations(db, "sqlite3", migrations.FS); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// OpenPostgres opens a PostgreSQL database at the given connection URL and runs
// any pending goose migrations using PostgreSQL-compatible migration files.
func OpenPostgres(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	if err := runMigrations(db, "postgres", pgmigrations.FS); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func applyPragmas(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA cache_size = -16000",
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			return fmt.Errorf("pragma: %w", err)
		}
	}
	return nil
}

func runMigrations(db *sql.DB, dialect string, migFS fs.FS) error {
	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migFS)

	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.Info("database migrations applied", "dialect", dialect)
	return nil
}

// MigrateStatus prints current migration status to stdout.
func MigrateStatus(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Status(db, ".")
}
