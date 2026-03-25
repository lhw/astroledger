-- +goose Up
-- Add market resolution type (binary yes/no, date prediction, numeric/price prediction).
-- This is additive-only — existing markets default to 'binary'.
ALTER TABLE markets ADD COLUMN resolution_type TEXT NOT NULL DEFAULT 'binary';
-- resolution_threshold stores context for non-binary markets:
--   date markets:    an ISO date string, e.g. "2025-06-30"
--   numeric markets: a stringified number, e.g. "200" (meaning $200 USD)
ALTER TABLE markets ADD COLUMN resolution_threshold TEXT;

-- +goose Down
-- SQLite does not support DROP COLUMN before version 3.35.0.
-- Handled by leaving the columns; the application ignores them on rollback.
SELECT 1;
