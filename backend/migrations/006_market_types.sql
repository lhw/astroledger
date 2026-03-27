-- +goose Up
-- resolution_type and resolution_threshold are now defined in 002_markets_trades.sql.
-- This migration is intentionally a no-op (kept so existing DB version counters stay valid).
SELECT 1;

-- +goose Down
SELECT 1;
