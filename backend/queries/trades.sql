-- name: CreateTrade :one
INSERT INTO trades (user_id, market_id, outcome_id, action, shares, cost, price_at_trade)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserPosition :one
SELECT user_id, market_id, outcome_id, shares
FROM positions
WHERE user_id = ? AND market_id = ? AND outcome_id = ?
LIMIT 1;

-- name: UpsertPosition :exec
INSERT INTO positions (user_id, market_id, outcome_id, shares)
VALUES (?, ?, ?, ?)
ON CONFLICT (user_id, market_id, outcome_id) DO UPDATE SET
    shares = positions.shares + excluded.shares;

-- name: GetMarketTrades :many
SELECT t.*, u.display_name AS trader_name, o.label AS outcome_label
FROM trades t
JOIN users u ON u.id = t.user_id
JOIN market_outcomes o ON o.id = t.outcome_id
WHERE t.market_id = ?
ORDER BY t.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetUserTrades :many
SELECT t.*, m.title AS market_title, o.label AS outcome_label
FROM trades t
JOIN markets m ON m.id = t.market_id
JOIN market_outcomes o ON o.id = t.outcome_id
WHERE t.user_id = ?
ORDER BY t.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetUserPositions :many
SELECT p.*, m.title AS market_title, m.status AS market_status,
       m.liquidity_param, o.label AS outcome_label
FROM positions p
JOIN markets m ON m.id = p.market_id
JOIN market_outcomes o ON o.id = p.outcome_id
WHERE p.user_id = ?
  AND p.shares > 0;

-- name: GetPositionsForResolution :many
SELECT p.user_id, p.outcome_id, p.shares
FROM positions p
WHERE p.market_id = ?
  AND p.shares > 0;

-- name: GetMarketStats :one
-- Returns aggregate trade statistics for a single market.
SELECT
    COALESCE(SUM(cost), 0)          AS total_volume,
    COUNT(DISTINCT user_id)         AS trader_count,
    COUNT(*)                        AS trade_count
FROM trades
WHERE market_id = ?;
