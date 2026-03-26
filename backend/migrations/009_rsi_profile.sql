-- +goose Up
-- Add RSI identity fields sourced from the SCID OIDC id_token claims.
-- rsi_handle / rsi_verified_at / rsi_enlisted / rsi_citizen_record are populated
-- after the user has confirmed their RSI identity via the SCID portal.
-- avatar_url stores the picture claim from the OIDC token (may be NULL).
-- is_rsi_verified mirrors group membership: 1 when the "verified" group is present.
ALTER TABLE users ADD COLUMN rsi_handle TEXT;
ALTER TABLE users ADD COLUMN rsi_verified_at TEXT;
ALTER TABLE users ADD COLUMN rsi_enlisted TEXT;
ALTER TABLE users ADD COLUMN rsi_citizen_record TEXT;
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE users ADD COLUMN is_rsi_verified INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite cannot drop columns; migration is intentionally irreversible.
SELECT 1;
