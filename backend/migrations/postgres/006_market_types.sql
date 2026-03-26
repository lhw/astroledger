-- +goose Up
ALTER TABLE markets ADD COLUMN IF NOT EXISTS resolution_type      TEXT NOT NULL DEFAULT 'binary';
ALTER TABLE markets ADD COLUMN IF NOT EXISTS resolution_threshold TEXT;

-- +goose Down
ALTER TABLE markets DROP COLUMN IF EXISTS resolution_threshold;
ALTER TABLE markets DROP COLUMN IF EXISTS resolution_type;
