-- name: AdminAdjustUserBalance :one
UPDATE users
SET balance = balance + ?
WHERE id = ?
RETURNING balance;

-- name: LogAdminAdjustment :exec
INSERT INTO admin_balance_adjustments (admin_id, user_id, amount, reason)
VALUES (?, ?, ?, ?);
