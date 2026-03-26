-- +goose Up
CREATE TABLE admin_balance_adjustments (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    admin_id   BIGINT      NOT NULL REFERENCES users(id),
    user_id    BIGINT      NOT NULL REFERENCES users(id),
    amount     BIGINT      NOT NULL,
    reason     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_adj_user  ON admin_balance_adjustments(user_id);
CREATE INDEX idx_admin_adj_admin ON admin_balance_adjustments(admin_id);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_adj_admin;
DROP INDEX IF EXISTS idx_admin_adj_user;
DROP TABLE IF EXISTS admin_balance_adjustments;
