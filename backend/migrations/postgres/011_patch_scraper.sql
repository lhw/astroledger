-- +goose Up
CREATE TABLE IF NOT EXISTS detected_patches (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title         TEXT        NOT NULL,
    patch_version TEXT        NOT NULL,
    thread_url    TEXT        NOT NULL UNIQUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notified      BOOLEAN     NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE IF EXISTS detected_patches;
