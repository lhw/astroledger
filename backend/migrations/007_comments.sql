-- +goose Up
CREATE TABLE comments (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id        INTEGER  NOT NULL REFERENCES markets(id),
    user_id          INTEGER  NOT NULL REFERENCES users(id),
    content          TEXT     NOT NULL,
    -- shadow-hide flag: 1 = comment is hidden from other users (abuse detected).
    -- The author can still see their own hidden comment.
    hidden           INTEGER  NOT NULL DEFAULT 0,
    -- Perspective API TOXICITY score [0.0, 1.0], null when moderation is disabled.
    toxicity_score   REAL,
    -- JSON object of per-attribute Perspective scores, null when moderation is disabled.
    moderation_flags TEXT,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_market_id ON comments(market_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_comments_market_id;
DROP TABLE IF EXISTS comments;
