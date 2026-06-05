-- +goose Up
ALTER TABLE api_tokens ADD COLUMN can_create_markets INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE api_tokens DROP COLUMN can_create_markets;
