-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS active_badge_key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS active_badge_key;
