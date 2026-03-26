-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scid_sub      TEXT        NOT NULL UNIQUE,
    display_name  TEXT        NOT NULL,
    email         TEXT        NOT NULL DEFAULT '',
    balance       BIGINT      NOT NULL DEFAULT 1000,
    is_moderator  INTEGER     NOT NULL DEFAULT 0,
    is_admin      INTEGER     NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_scid_sub ON users(scid_sub);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
