-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN active_badge_key TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN before 3.35; leave column in place.
-- +goose StatementEnd
