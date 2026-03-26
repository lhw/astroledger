-- +goose Up
CREATE TABLE admin_balance_adjustments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id   INTEGER NOT NULL REFERENCES users(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    amount     INTEGER NOT NULL,
    reason     TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_admin_adj_user  ON admin_balance_adjustments(user_id);
CREATE INDEX idx_admin_adj_admin ON admin_balance_adjustments(admin_id);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_adj_admin;
DROP INDEX IF EXISTS idx_admin_adj_user;
DROP TABLE IF EXISTS admin_balance_adjustments;
