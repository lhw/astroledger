-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS rsi_handle         TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS rsi_verified_at    TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS rsi_enlisted       TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS rsi_citizen_record TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url         TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_rsi_verified    BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS is_rsi_verified;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS rsi_citizen_record;
ALTER TABLE users DROP COLUMN IF EXISTS rsi_enlisted;
ALTER TABLE users DROP COLUMN IF EXISTS rsi_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS rsi_handle;
