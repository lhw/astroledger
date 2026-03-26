-- name: CreateComment :one
INSERT INTO comments (market_id, user_id, content, hidden, toxicity_score, moderation_flags)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetCommentsByMarket :many
-- Returns all visible comments plus any hidden comments belonging to the viewer.
-- Pass viewer_id=0 for anonymous requests (hidden comments never shown).
SELECT
    c.id,
    c.market_id,
    c.user_id,
    c.content,
    c.hidden,
    c.toxicity_score,
    c.moderation_flags,
    c.created_at,
    u.display_name AS author_name
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.market_id = ?
  AND (c.hidden = 0 OR c.user_id = ?)
ORDER BY c.created_at ASC
LIMIT 100;

-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = ?;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = ?;
