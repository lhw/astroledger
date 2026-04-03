-- name: GetUserBySub :one
SELECT id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at,
       rsi_handle, rsi_verified_at, rsi_enlisted, rsi_citizen_record, avatar_url, is_rsi_verified
FROM users
WHERE scid_sub = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at,
       rsi_handle, rsi_verified_at, rsi_enlisted, rsi_citizen_record, avatar_url, is_rsi_verified
FROM users
WHERE id = ?
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (scid_sub, display_name, email)
VALUES (?, ?, ?)
RETURNING id, scid_sub, display_name, email, balance, is_moderator, is_admin, created_at, last_login_at,
          rsi_handle, rsi_verified_at, rsi_enlisted, rsi_citizen_record, avatar_url, is_rsi_verified;

-- name: UpdateUserBalance :exec
UPDATE users
SET balance = balance + ?
WHERE id = ?;

-- name: SearchUsers :many
SELECT id, display_name, rsi_handle, balance
FROM users
WHERE display_name LIKE sqlc.arg(pattern)
   OR (rsi_handle IS NOT NULL AND rsi_handle LIKE sqlc.arg(pattern))
ORDER BY display_name
LIMIT 10;

-- name: GetLeaderboard :many
WITH user_portfolio AS (
    SELECT u.id, u.display_name, u.balance,
           COALESCE(SUM(
               CASE
                   WHEN p.shares > 0 AND m.status = 'active' THEN
                       CAST(p.shares *
                           exp(o.shares / m.liquidity_param) /
                           (SELECT SUM(exp(o2.shares / m.liquidity_param))
                            FROM market_outcomes o2
                            WHERE o2.market_id = m.id)
                       * 100 AS INTEGER)
                   ELSE 0
               END
           ), 0) AS portfolio_value
    FROM users u
    LEFT JOIN positions p ON p.user_id = u.id
    LEFT JOIN markets m ON m.id = p.market_id
    LEFT JOIN market_outcomes o ON o.id = p.outcome_id
    GROUP BY u.id
)
SELECT id, display_name, balance, portfolio_value
FROM user_portfolio
ORDER BY (balance + portfolio_value) DESC
LIMIT ?;
