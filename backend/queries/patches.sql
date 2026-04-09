-- name: ListNewPatches :many
SELECT * FROM detected_patches WHERE notified = 0 ORDER BY id DESC;

-- name: ListAllPatches :many
SELECT * FROM detected_patches ORDER BY id DESC LIMIT 50;

-- name: MarkPatchNotified :exec
UPDATE detected_patches SET notified = 1 WHERE id = ?;
