package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lhw/scolymarket/migrations"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
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

	if err := migrate(db); err != nil {
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

func migrate(db *sql.DB) error {
	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.Info("database migrations applied")
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
