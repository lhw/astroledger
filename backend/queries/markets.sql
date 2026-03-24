-- name: CreateMarket :one
INSERT INTO markets (title, description, category, resolution_criteria, resolution_deadline, created_by, liquidity_param)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMarketByID :one
SELECT m.*, u.display_name AS creator_name
FROM markets m
JOIN users u ON u.id = m.created_by
WHERE m.id = ?
LIMIT 1;

-- name: ListMarkets :many
SELECT m.*, u.display_name AS creator_name
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

-- name: ResolveMarket :exec
UPDATE markets
SET status      = 'resolved',
    resolution  = ?,
    resolved_by = ?,
    resolved_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
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
