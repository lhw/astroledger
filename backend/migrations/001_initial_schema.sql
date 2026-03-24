-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    scid_sub     TEXT    NOT NULL UNIQUE,
    display_name TEXT    NOT NULL,
    email        TEXT    NOT NULL DEFAULT '',
    balance      INTEGER NOT NULL DEFAULT 1000,
    is_moderator INTEGER NOT NULL DEFAULT 0,  -- SQLite has no BOOLEAN
    is_admin     INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_login_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_users_scid_sub ON users(scid_sub);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
