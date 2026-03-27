-- +goose Up

CREATE TABLE IF NOT EXISTS mod_audit (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    action_type TEXT        NOT NULL,
    target_type TEXT        NOT NULL,
    target_id   BIGINT      NOT NULL,
    mod_user_id BIGINT      NOT NULL REFERENCES users(id),
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mod_audit_created_at ON mod_audit(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mod_audit_mod_user_id ON mod_audit(mod_user_id);
CREATE INDEX IF NOT EXISTS idx_mod_audit_target ON mod_audit(target_type, target_id);

-- +goose Down

DROP INDEX IF EXISTS idx_mod_audit_target;
DROP INDEX IF EXISTS idx_mod_audit_mod_user_id;
DROP INDEX IF EXISTS idx_mod_audit_created_at;
DROP TABLE IF EXISTS mod_audit;
