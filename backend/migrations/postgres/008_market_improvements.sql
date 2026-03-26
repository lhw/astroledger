-- +goose Up
ALTER TABLE markets ADD COLUMN IF NOT EXISTS resolution_evidence TEXT;

CREATE INDEX IF NOT EXISTS idx_markets_status_created  ON markets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_markets_status_deadline ON markets(status, resolution_deadline);

-- +goose Down
DROP INDEX IF EXISTS idx_markets_status_deadline;
DROP INDEX IF EXISTS idx_markets_status_created;
ALTER TABLE markets DROP COLUMN IF EXISTS resolution_evidence;
