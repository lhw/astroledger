-- +goose Up
CREATE TABLE IF NOT EXISTS detected_patches (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    title         TEXT NOT NULL,
    patch_version TEXT NOT NULL,
    thread_url    TEXT NOT NULL UNIQUE,
    first_seen_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    notified      INTEGER NOT NULL DEFAULT 0 -- 1 once a mod has seen it
);

-- +goose Down
DROP TABLE IF EXISTS detected_patches;
