-- +goose Up
-- Seed data moved to cmd/seed. This migration is intentionally empty.
-- The version number is kept so existing databases are not re-migrated.
SELECT 1;

-- +goose Down
SELECT 1;
