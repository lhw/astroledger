// Package pgmigrations provides the embedded PostgreSQL migration files.
package pgmigrations

import "embed"

//go:embed *.sql
var FS embed.FS
