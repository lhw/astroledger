-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_reporter_market_pending
    ON reports(reporter_id, market_id) WHERE status = 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_reports_reporter_market_pending;
-- +goose StatementEnd
