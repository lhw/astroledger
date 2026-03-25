package db

import "context"

// UpdateUserGroups updates the is_moderator and is_admin flags for a user.
// Called after OIDC login when group membership is resolved from the id_token groups claim.
func (q *Queries) UpdateUserGroups(ctx context.Context, id, isModerator, isAdmin int64) error {
	const sql = `UPDATE users SET is_moderator = ?, is_admin = ? WHERE id = ?`
	_, err := q.db.ExecContext(ctx, sql, isModerator, isAdmin, id)
	return err
}
