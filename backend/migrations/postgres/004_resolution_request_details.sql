-- +goose Up
CREATE TABLE IF NOT EXISTS resolution_request_details (
    market_id    BIGINT PRIMARY KEY REFERENCES markets(id),
    requested_by BIGINT NOT NULL REFERENCES users(id),
    link         TEXT,
    note         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS resolution_request_details;
