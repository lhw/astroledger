-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN is_banned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN is_shadow_banned INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN is_shadow_banned;
ALTER TABLE users DROP COLUMN is_banned;
-- +goose StatementEnd
