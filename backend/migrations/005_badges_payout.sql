-- +goose Up
CREATE TABLE IF NOT EXISTS user_badges (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key  TEXT    NOT NULL,
    awarded_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (user_id, badge_key)
);

CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);

-- Tracks which weekly payouts have already run to prevent double-paying.
CREATE TABLE IF NOT EXISTS weekly_payout_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    week_key   TEXT    NOT NULL UNIQUE, -- 'YYYY-WW' e.g. '2026-12'
    user_count INTEGER NOT NULL DEFAULT 0,
    paid_at    DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose Down
DROP TABLE IF EXISTS weekly_payout_log;
DROP TABLE IF EXISTS user_badges;
