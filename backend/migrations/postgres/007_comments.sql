-- +goose Up
CREATE TABLE comments (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    market_id        BIGINT           NOT NULL REFERENCES markets(id),
    user_id          BIGINT           NOT NULL REFERENCES users(id),
    content          TEXT             NOT NULL,
    hidden           INTEGER          NOT NULL DEFAULT 0,
    toxicity_score   DOUBLE PRECISION,
    moderation_flags TEXT,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_market_id ON comments(market_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_comments_market_id;
DROP TABLE IF EXISTS comments;
