-- +goose Up
CREATE TABLE IF NOT EXISTS api_tokens (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    token_hash   TEXT        NOT NULL UNIQUE,
    token_prefix TEXT        NOT NULL,
    can_read     INTEGER     NOT NULL DEFAULT 1,
    can_trade    INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user_active ON api_tokens(user_id, revoked_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_hash_active ON api_tokens(token_hash, revoked_at);

-- +goose Down
DROP INDEX IF EXISTS idx_api_tokens_hash_active;
DROP INDEX IF EXISTS idx_api_tokens_user_active;
DROP TABLE IF EXISTS api_tokens;
