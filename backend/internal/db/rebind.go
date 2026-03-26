package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// rebindPG converts ? positional placeholders (SQLite/sqlc style) to $1, $2, …
// (PostgreSQL/pgx style). Safe to call on any query; returns the original string
// unchanged when no ? is present.
func rebindPG(query string) string {
	if !strings.ContainsRune(query, '?') {
		return query
	}
	var n int
	var b strings.Builder
	b.Grow(len(query) + 8)
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

// pgDB wraps *sql.DB and rewrites ? → $N before forwarding to the PostgreSQL driver.
// This lets the sqlc-generated SQLite-style queries run against PostgreSQL unchanged.
type pgDB struct{ db *sql.DB }

func (w *pgDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return w.db.ExecContext(ctx, rebindPG(q), args...)
}
func (w *pgDB) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	return w.db.PrepareContext(ctx, rebindPG(q))
}
func (w *pgDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return w.db.QueryContext(ctx, rebindPG(q), args...)
}
func (w *pgDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return w.db.QueryRowContext(ctx, rebindPG(q), args...)
}

// pgTx wraps *sql.Tx and rewrites ? → $N before forwarding to the PostgreSQL driver.
type pgTx struct{ tx *sql.Tx }

func (w *pgTx) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return w.tx.ExecContext(ctx, rebindPG(q), args...)
}
func (w *pgTx) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	return w.tx.PrepareContext(ctx, rebindPG(q))
}
func (w *pgTx) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return w.tx.QueryContext(ctx, rebindPG(q), args...)
}
func (w *pgTx) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return w.tx.QueryRowContext(ctx, rebindPG(q), args...)
}

// NewPostgres creates a Queries backed by a PostgreSQL database.
// The underlying *sql.DB is wrapped with a ? → $N placeholder rewriter so that
// the sqlc-generated code (which targets SQLite with ? style) works against
// PostgreSQL without modification.
func NewPostgres(sqlDB *sql.DB) *Queries {
	return &Queries{db: &pgDB{db: sqlDB}}
}

// isPG reports whether this Queries instance is backed by PostgreSQL.
func (q *Queries) isPG() bool {
	switch q.db.(type) {
	case *pgDB, *pgTx:
		return true
	}
	return false
}

// WithBoundTx creates a transaction-bound Queries, preserving the current
// backend's placeholder style. Use this instead of WithTx in service code so
// that both SQLite and PostgreSQL transaction queries receive correct placeholders.
func (q *Queries) WithBoundTx(tx *sql.Tx) *Queries {
	if q.isPG() {
		return &Queries{db: &pgTx{tx: tx}}
	}
	return q.WithTx(tx)
}
