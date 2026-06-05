-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, can_read, can_trade, can_create_markets)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, user_id, name, token_prefix, can_read, can_trade, can_create_markets, created_at, last_used_at, revoked_at;

-- name: ListUserAPITokens :many
SELECT id, user_id, name, token_prefix, can_read, can_trade, can_create_markets, created_at, last_used_at, revoked_at
FROM api_tokens
WHERE user_id = ?
  AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetAPITokenByHash :one
SELECT id, user_id, name, token_prefix, token_hash, can_read, can_trade, can_create_markets, created_at, last_used_at, revoked_at
FROM api_tokens
WHERE token_hash = ?
  AND revoked_at IS NULL
LIMIT 1;

-- name: TouchAPITokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: RevokeUserAPIToken :one
UPDATE api_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE id = ?
  AND user_id = ?
  AND revoked_at IS NULL
RETURNING id;
