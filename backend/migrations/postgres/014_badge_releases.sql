-- +goose Up

-- Admin-controlled badge releases: each row makes a badge available in the FOMO store
-- for a configurable price, stock, and time window.
-- released_at and created_at are TIMESTAMPTZ; the Go layer (scanBadgeRelease) uses
-- the isPG() branch to scan them directly as time.Time.
CREATE TABLE badge_releases (
    id          BIGINT           GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    badge_key   TEXT             NOT NULL,
    price       BIGINT           NOT NULL CHECK (price >= 0),
    stock       BIGINT,
    released_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,
    active      INTEGER          NOT NULL DEFAULT 1,
    notes       TEXT,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- Store the bUEC price paid when a badge was purchased, used for admiral rank
-- spend calculations. Earned badges keep the default of 0.
ALTER TABLE user_badges ADD COLUMN IF NOT EXISTS purchase_price INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE user_badges DROP COLUMN IF EXISTS purchase_price;
DROP TABLE IF EXISTS badge_releases;
