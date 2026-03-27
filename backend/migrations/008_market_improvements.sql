-- +goose Up
-- resolution_evidence and the status+deadline/created composite indexes are
-- now defined in 002_markets_trades.sql. This migration is a no-op.
SELECT 1;

-- +goose Down
SELECT 1;
