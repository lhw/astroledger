-- +goose Up
-- Add resolution_evidence for storing mod-provided evidence links upon resolution.
ALTER TABLE markets ADD COLUMN resolution_evidence TEXT;

-- Composite indexes for the background market-expiry job.
CREATE INDEX IF NOT EXISTS idx_markets_status_created  ON markets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_markets_status_deadline ON markets(status, resolution_deadline);

-- +goose Down
DROP INDEX IF EXISTS idx_markets_status_deadline;
DROP INDEX IF EXISTS idx_markets_status_created;
-- SQLite does not support DROP COLUMN before 3.35; columns are left in place.
SELECT 1;
