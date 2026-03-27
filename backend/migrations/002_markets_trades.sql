-- +goose Up
-- +goose StatementBegin

-- Markets table. Binary markets have exactly 2 outcomes (YES/NO); multi-outcome
-- markets have 3+ outcomes. The AMM runs independent LMSR pools per outcome.
-- resolved_outcome_id is NULL until the market is resolved; it points to the
-- winning outcome row in market_outcomes.
CREATE TABLE IF NOT EXISTS markets (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    title                TEXT    NOT NULL,
    description          TEXT    NOT NULL DEFAULT '',
    category             TEXT    NOT NULL,  -- bug_fixes|feature_delivery|patch_timing|cig_drama|community_events|meta
    resolution_criteria  TEXT    NOT NULL DEFAULT '',
    resolution_deadline  DATETIME NOT NULL,
    status               TEXT    NOT NULL DEFAULT 'pending_review',  -- pending_review|active|resolved|cancelled|resolution_requested|deadline_passed
    resolved_outcome_id  INTEGER REFERENCES market_outcomes(id),  -- NULL until resolved
    created_by           INTEGER NOT NULL REFERENCES users(id),
    resolved_by          INTEGER REFERENCES users(id),
    created_at           DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    resolved_at          DATETIME,
    liquidity_param      REAL    NOT NULL DEFAULT 100.0,
    resolution_type      TEXT    NOT NULL DEFAULT 'binary',  -- binary|date|numeric|multi
    resolution_threshold TEXT,   -- ISO date or numeric threshold (binary/date/numeric only)
    resolution_evidence  TEXT    -- mod-provided evidence link on resolution
);

CREATE INDEX IF NOT EXISTS idx_markets_status     ON markets(status);
CREATE INDEX IF NOT EXISTS idx_markets_created_by ON markets(created_by);
CREATE INDEX IF NOT EXISTS idx_markets_deadline   ON markets(resolution_deadline);
CREATE INDEX IF NOT EXISTS idx_markets_status_created  ON markets(status, created_at);
CREATE INDEX IF NOT EXISTS idx_markets_status_deadline ON markets(status, resolution_deadline);

-- Each outcome of a market. Binary markets always have outcomes labelled 'YES'
-- and 'NO'. Multi-outcome markets can have any labels (e.g. patch versions).
-- shares is the outstanding share pool for this outcome's LMSR.
CREATE TABLE IF NOT EXISTS market_outcomes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id  INTEGER NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    label      TEXT    NOT NULL,   -- e.g. 'YES', 'NO', '4.1', '4.2', 'Never'
    shares     REAL    NOT NULL DEFAULT 0.0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_outcomes_market_id ON market_outcomes(market_id);

CREATE TABLE IF NOT EXISTS trades (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    market_id      INTEGER NOT NULL REFERENCES markets(id),
    outcome_id     INTEGER NOT NULL REFERENCES market_outcomes(id),
    action         TEXT    NOT NULL,  -- buy|sell
    shares         REAL    NOT NULL,
    cost           INTEGER NOT NULL,  -- ScollyBucks (integer, always)
    price_at_trade REAL    NOT NULL,  -- 0.0–1.0 probability at time of trade
    created_at     DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_trades_user_id   ON trades(user_id);
CREATE INDEX IF NOT EXISTS idx_trades_market_id ON trades(market_id);

CREATE TABLE IF NOT EXISTS positions (
    user_id    INTEGER NOT NULL REFERENCES users(id),
    market_id  INTEGER NOT NULL REFERENCES markets(id),
    outcome_id INTEGER NOT NULL REFERENCES market_outcomes(id),
    shares     REAL    NOT NULL DEFAULT 0.0,
    PRIMARY KEY (user_id, market_id, outcome_id)
);

CREATE INDEX IF NOT EXISTS idx_positions_user_id ON positions(user_id);

CREATE TABLE IF NOT EXISTS moderation_actions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id    INTEGER NOT NULL REFERENCES markets(id),
    moderator_id INTEGER NOT NULL REFERENCES users(id),
    action       TEXT    NOT NULL,  -- approve|reject|edit|cancel|resolve
    reason       TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS reports (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter_id INTEGER NOT NULL REFERENCES users(id),
    market_id   INTEGER NOT NULL REFERENCES markets(id),
    reason      TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'pending',  -- pending|reviewed|dismissed
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS autofilter_rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_type  TEXT    NOT NULL,  -- keyword|regex|rate_limit|min_length
    value      TEXT    NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Seed some default auto-filter rules.
INSERT INTO autofilter_rules (rule_type, value) VALUES
    ('keyword',    'real money'),
    ('keyword',    'real cash'),
    ('keyword',    'doxxing'),
    ('min_length', '20');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS autofilter_rules;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS moderation_actions;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS market_outcomes;
DROP TABLE IF EXISTS markets;
-- +goose StatementEnd
