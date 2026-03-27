-- +goose Up

-- Admin-controlled badge releases: each row makes a badge available in the FOMO store
-- for a configurable price, stock, and time window.
CREATE TABLE badge_releases (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    badge_key   TEXT    NOT NULL,
    price       INTEGER NOT NULL CHECK (price >= 0),
    stock       INTEGER,            -- NULL = unlimited
    released_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  TEXT,               -- NULL = no expiry
    active      INTEGER NOT NULL DEFAULT 1,
    notes       TEXT,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Store the bUEC price paid when a badge was purchased, used for admiral rank
-- spend calculations. Earned badges keep the default of 0.
ALTER TABLE user_badges ADD COLUMN purchase_price INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE user_badges DROP COLUMN purchase_price;
DROP TABLE badge_releases;
