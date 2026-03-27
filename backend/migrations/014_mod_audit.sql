-- +goose Up

CREATE TABLE mod_audit (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type TEXT    NOT NULL,
    target_type TEXT    NOT NULL,
    target_id   INTEGER NOT NULL,
    mod_user_id INTEGER NOT NULL,
    note        TEXT,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (mod_user_id) REFERENCES users(id)
);

CREATE INDEX idx_mod_audit_created_at ON mod_audit(created_at DESC);
CREATE INDEX idx_mod_audit_mod_user_id ON mod_audit(mod_user_id);
CREATE INDEX idx_mod_audit_target ON mod_audit(target_type, target_id);

-- +goose Down

DROP INDEX IF EXISTS idx_mod_audit_target;
DROP INDEX IF EXISTS idx_mod_audit_mod_user_id;
DROP INDEX IF EXISTS idx_mod_audit_created_at;
DROP TABLE mod_audit;
