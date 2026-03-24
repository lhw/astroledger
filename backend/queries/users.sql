-- name: GetUserBySub :one
SELECT id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at
FROM users
WHERE scid_sub = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (scid_sub, display_name, email)
VALUES (?, ?, ?)
RETURNING id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    display_name  = ?,
    email         = ?
WHERE id = ?;

-- name: UpdateUserBalance :exec
UPDATE users
SET balance = balance + ?
WHERE id = ?;

-- name: GetLeaderboard :many
SELECT u.id, u.display_name, u.balance,
       COALESCE(SUM(
           CASE
               WHEN p.yes_shares > 0 AND m.status = 'active' THEN
                   CAST(p.yes_shares * (m.yes_shares / (m.yes_shares + m.no_shares + 0.0001) * 100) AS INTEGER)
               WHEN p.no_shares > 0 AND m.status = 'active' THEN
                   CAST(p.no_shares * ((1 - m.yes_shares / (m.yes_shares + m.no_shares + 0.0001)) * 100) AS INTEGER)
               ELSE 0
           END
       ), 0) AS portfolio_value
FROM users u
LEFT JOIN positions p ON p.user_id = u.id
LEFT JOIN markets m ON m.id = p.market_id
GROUP BY u.id
ORDER BY (u.balance + portfolio_value) DESC
LIMIT ?;
