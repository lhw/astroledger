-- +goose Up
CREATE TABLE IF NOT EXISTS resolution_request_details (
    market_id   INTEGER PRIMARY KEY,
    requested_by INTEGER NOT NULL,
    link        TEXT,
    note        TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (market_id)    REFERENCES markets(id),
    FOREIGN KEY (requested_by) REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS resolution_request_details;
