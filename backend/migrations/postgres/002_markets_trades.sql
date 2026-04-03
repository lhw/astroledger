-- +goose Up
-- +goose StatementBegin

-- markets: resolved_outcome_id FK added after market_outcomes exists (see ALTER TABLE below)
CREATE TABLE IF NOT EXISTS markets (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title                TEXT             NOT NULL,
    description          TEXT             NOT NULL DEFAULT '',
    category             TEXT             NOT NULL,
    resolution_criteria  TEXT             NOT NULL DEFAULT '',
    resolution_deadline  TIMESTAMPTZ      NOT NULL,
    status               TEXT             NOT NULL DEFAULT 'pending_review',
    resolved_outcome_id  BIGINT,
    created_by           BIGINT           NOT NULL REFERENCES users(id),
    resolved_by          BIGINT           REFERENCES users(id),
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    resolved_at          TIMESTAMPTZ,
    liquidity_param      DOUBLE PRECISION NOT NULL DEFAULT 100.0,
    resolution_type      TEXT             NOT NULL DEFAULT 'binary',
    resolution_threshold TEXT,
    resolution_evidence  TEXT
);

CREATE INDEX IF NOT EXISTS idx_markets_status          ON markets(status);
CREATE INDEX IF NOT EXISTS idx_markets_created_by      ON markets(created_by);
CREATE INDEX IF NOT EXISTS idx_markets_deadline        ON markets(resolution_deadline);
CREATE INDEX IF NOT EXISTS idx_markets_status_created  ON markets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_markets_status_deadline ON markets(status, resolution_deadline);

CREATE TABLE IF NOT EXISTS market_outcomes (
    id         BIGINT           GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    market_id  BIGINT           NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    label      TEXT             NOT NULL,
    shares     DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    sort_order BIGINT           NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_outcomes_market_id ON market_outcomes(market_id);

ALTER TABLE markets
    ADD CONSTRAINT fk_markets_resolved_outcome
    FOREIGN KEY (resolved_outcome_id) REFERENCES market_outcomes(id);

CREATE TABLE IF NOT EXISTS trades (
    id             BIGINT           GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        BIGINT           NOT NULL REFERENCES users(id),
    market_id      BIGINT           NOT NULL REFERENCES markets(id),
    outcome_id     BIGINT           NOT NULL REFERENCES market_outcomes(id),
    action         TEXT             NOT NULL,
    shares         DOUBLE PRECISION NOT NULL,
    cost           BIGINT           NOT NULL,
    price_at_trade DOUBLE PRECISION NOT NULL,
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_user_id   ON trades(user_id);
CREATE INDEX IF NOT EXISTS idx_trades_market_id ON trades(market_id);

CREATE TABLE IF NOT EXISTS positions (
    user_id    BIGINT           NOT NULL REFERENCES users(id),
    market_id  BIGINT           NOT NULL REFERENCES markets(id),
    outcome_id BIGINT           NOT NULL REFERENCES market_outcomes(id),
    shares     DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    PRIMARY KEY (user_id, market_id, outcome_id)
);

CREATE INDEX IF NOT EXISTS idx_positions_user_id ON positions(user_id);

CREATE TABLE IF NOT EXISTS moderation_actions (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    market_id    BIGINT      NOT NULL REFERENCES markets(id),
    moderator_id BIGINT      NOT NULL REFERENCES users(id),
    action       TEXT        NOT NULL,
    reason       TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reports (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reporter_id BIGINT      NOT NULL REFERENCES users(id),
    market_id   BIGINT      NOT NULL REFERENCES markets(id),
    reason      TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS autofilter_rules (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_type  TEXT        NOT NULL,
    value      TEXT        NOT NULL,
    enabled    INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO autofilter_rules (rule_type, value) VALUES
    ('keyword',    'real money'),
    ('keyword',    'real cash'),
    ('keyword',    'doxxing'),
    ('min_length', '20');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE markets DROP CONSTRAINT IF EXISTS fk_markets_resolved_outcome;
DROP TABLE IF EXISTS autofilter_rules;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS moderation_actions;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS market_outcomes;
DROP TABLE IF EXISTS markets;
-- +goose StatementEnd
