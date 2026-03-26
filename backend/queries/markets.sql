-- name: CreateMarket :one
INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, created_by, liquidity_param, resolution_type, resolution_threshold)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMarketByID :one
SELECT m.*, u.display_name AS creator_name, ru.display_name AS resolver_name
FROM markets m
JOIN users u ON u.id = m.created_by
LEFT JOIN users ru ON ru.id = m.resolved_by
WHERE m.id = ?
LIMIT 1;

-- name: ListMarkets :many
SELECT m.*, u.display_name AS creator_name,
       (SELECT COUNT(*) FROM comments c WHERE c.market_id = m.id AND c.hidden = 0) AS comment_count
FROM markets m
JOIN users u ON u.id = m.created_by
WHERE m.status = ?
  AND (? = '' OR m.category = ?)
ORDER BY m.created_at DESC
LIMIT ? OFFSET ?;

-- name: CountMarkets :one
SELECT COUNT(*) FROM markets
WHERE status = ?
  AND (? = '' OR category = ?);

-- name: UpdateMarketStatus :exec
UPDATE markets SET status = ? WHERE id = ?;

-- name: UpdateMarketAMMState :exec
UPDATE markets
SET yes_shares = ?,
    no_shares  = ?
WHERE id = ?;

-- name: ListPendingMarkets :many
SELECT m.*, u.display_name AS creator_name
FROM markets m
JOIN users u ON u.id = m.created_by
WHERE m.status = 'pending_review'
ORDER BY m.created_at ASC;

-- name: GetMarketPriceHistory :many
SELECT price_at_trade, side, created_at
FROM trades
WHERE market_id = ?
ORDER BY created_at ASC;

-- name: GetActivePendingMarketTitles :many
-- Returns titles of all non-archived markets for duplicate-title detection.
SELECT title FROM markets
WHERE status IN ('pending_review', 'active', 'resolution_requested');


