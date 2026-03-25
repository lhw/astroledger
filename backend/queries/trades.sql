-- name: CreateTrade :one
INSERT INTO trades (user_id, market_id, side, action, shares, cost, price_at_trade)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserPosition :one
SELECT user_id, market_id, yes_shares, no_shares
FROM positions
WHERE user_id = ? AND market_id = ?
LIMIT 1;

-- name: UpsertPosition :exec
INSERT INTO positions (user_id, market_id, yes_shares, no_shares)
VALUES (?, ?, ?, ?)
ON CONFLICT (user_id, market_id) DO UPDATE SET
    yes_shares = positions.yes_shares + excluded.yes_shares,
    no_shares  = positions.no_shares  + excluded.no_shares;

-- name: GetMarketTrades :many
SELECT t.*, u.display_name AS trader_name
FROM trades t
JOIN users u ON u.id = t.user_id
WHERE t.market_id = ?
ORDER BY t.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetUserTrades :many
SELECT t.*, m.title AS market_title
FROM trades t
JOIN markets m ON m.id = t.market_id
WHERE t.user_id = ?
ORDER BY t.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetUserPositions :many
SELECT p.*, m.title AS market_title, m.status AS market_status,
       m.yes_shares AS pool_yes, m.no_shares AS pool_no, m.liquidity_param
FROM positions p
JOIN markets m ON m.id = p.market_id
WHERE p.user_id = ?
  AND (p.yes_shares > 0 OR p.no_shares > 0);

-- name: GetPositionsForResolution :many
SELECT user_id, yes_shares, no_shares
FROM positions
WHERE market_id = ?
  AND (yes_shares > 0 OR no_shares > 0);

-- name: GetMarketStats :one
-- Returns aggregate trade statistics for a single market.
SELECT
    COALESCE(SUM(cost), 0)          AS total_volume,
    COUNT(DISTINCT user_id)         AS trader_count,
    COUNT(*)                        AS trade_count
FROM trades
WHERE market_id = ?;
