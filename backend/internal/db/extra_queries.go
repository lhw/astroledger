package db

import (
	"context"
	"database/sql"
	"time"
)

// UpdateUserGroups updates the is_moderator, is_admin, and is_rsi_verified flags for a user.
// Called after OIDC login when group membership is resolved from the id_token groups claim.
func (q *Queries) UpdateUserGroups(ctx context.Context, id, isModerator, isAdmin, isRsiVerified int64) error {
	const stmt = `UPDATE users SET is_moderator = ?, is_admin = ?, is_rsi_verified = ? WHERE id = ?`
	_, err := q.db.ExecContext(ctx, stmt, isModerator, isAdmin, isRsiVerified, id)
	return err
}

// ResolutionRequestRow is the result of ListResolutionRequestedMarkets.
// It is the full market record plus resolution-request metadata.
type ResolutionRequestRow struct {
	ID                 int64      `json:"id"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Category           string     `json:"category"`
	ResolutionCriteria string     `json:"resolution_criteria"`
	ResolutionDeadline time.Time  `json:"resolution_deadline"`
	Status             string     `json:"status"`
	Resolution         *string    `json:"resolution"`
	CreatedBy          int64      `json:"created_by"`
	ResolvedBy         *int64     `json:"resolved_by"`
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	LiquidityParam     float64    `json:"liquidity_param"`
	YesShares          float64    `json:"yes_shares"`
	NoShares           float64    `json:"no_shares"`
	CreatorName        string     `json:"creator_name"`
	// Resolution-request specific fields (from resolution_request_details).
	RequestedBy   int64     `json:"requested_by"`
	RequesterName string    `json:"requester_name"`
	RequestLink   *string   `json:"request_link"`
	RequestNote   *string   `json:"request_note"`
	RequestedAt   time.Time `json:"requested_at"`
}

// ListResolutionRequestedMarkets returns markets that have had a resolution request
// filed, together with the requester's details and any link/note provided.
func (q *Queries) ListResolutionRequestedMarkets(ctx context.Context) ([]ResolutionRequestRow, error) {
	// SQLite returns strftime-produced TEXT; PostgreSQL returns native timestamptz.
	// We build two variants so the scan matches what the driver returns.
	var query string
	if q.isPG() {
		query = `
SELECT m.id, m.title, m.description, m.category,
       m.resolution_criteria, m.resolution_deadline,
       m.status, m.resolution, m.created_by, m.resolved_by,
       m.created_at, m.resolved_at, m.liquidity_param,
       m.yes_shares, m.no_shares,
       creator.display_name AS creator_name,
       COALESCE(rrd.requested_by, m.created_by) AS requested_by,
       COALESCE(requester.display_name, creator.display_name) AS requester_name,
       rrd.link, rrd.note,
       COALESCE(rrd.created_at, m.created_at) AS requested_at
FROM markets m
JOIN users creator ON creator.id = m.created_by
LEFT JOIN resolution_request_details rrd ON rrd.market_id = m.id
LEFT JOIN users requester ON requester.id = rrd.requested_by
WHERE m.status = 'resolution_requested'
ORDER BY requested_at ASC`
	} else {
		query = `
SELECT m.id, m.title, m.description, m.category,
       m.resolution_criteria, m.resolution_deadline,
       m.status, m.resolution, m.created_by, m.resolved_by,
       m.created_at, m.resolved_at, m.liquidity_param,
       m.yes_shares, m.no_shares,
       creator.display_name AS creator_name,
       COALESCE(rrd.requested_by, m.created_by) AS requested_by,
       COALESCE(requester.display_name, creator.display_name) AS requester_name,
       rrd.link, rrd.note,
       strftime('%Y-%m-%dT%H:%M:%SZ', COALESCE(rrd.created_at, m.created_at)) AS requested_at
FROM markets m
JOIN users creator ON creator.id = m.created_by
LEFT JOIN resolution_request_details rrd ON rrd.market_id = m.id
LEFT JOIN users requester ON requester.id = rrd.requested_by
WHERE m.status = 'resolution_requested'
ORDER BY requested_at ASC`
	}

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ResolutionRequestRow, 0)
	for rows.Next() {
		var i ResolutionRequestRow
		if q.isPG() {
			// PostgreSQL returns timestamptz — scan directly as time.Time.
			if err := rows.Scan(
				&i.ID, &i.Title, &i.Description, &i.Category,
				&i.ResolutionCriteria, &i.ResolutionDeadline,
				&i.Status, &i.Resolution, &i.CreatedBy, &i.ResolvedBy,
				&i.CreatedAt, &i.ResolvedAt, &i.LiquidityParam,
				&i.YesShares, &i.NoShares, &i.CreatorName,
				&i.RequestedBy, &i.RequesterName, &i.RequestLink, &i.RequestNote,
				&i.RequestedAt,
			); err != nil {
				return nil, err
			}
		} else {
			// SQLite returns the strftime-produced TEXT string — parse manually.
			var requestedAtStr string
			if err := rows.Scan(
				&i.ID, &i.Title, &i.Description, &i.Category,
				&i.ResolutionCriteria, &i.ResolutionDeadline,
				&i.Status, &i.Resolution, &i.CreatedBy, &i.ResolvedBy,
				&i.CreatedAt, &i.ResolvedAt, &i.LiquidityParam,
				&i.YesShares, &i.NoShares, &i.CreatorName,
				&i.RequestedBy, &i.RequesterName, &i.RequestLink, &i.RequestNote,
				&requestedAtStr,
			); err != nil {
				return nil, err
			}
			for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
				if t, err2 := time.Parse(layout, requestedAtStr); err2 == nil {
					i.RequestedAt = t.UTC()
					break
				}
			}
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// UpsertResolutionRequestDetails inserts or replaces the link/note for a resolution request.
func (q *Queries) UpsertResolutionRequestDetails(ctx context.Context, marketID, requestedBy int64, link, note *string) error {
	const stmt = `
INSERT INTO resolution_request_details (market_id, requested_by, link, note)
VALUES (?, ?, ?, ?)
ON CONFLICT(market_id) DO UPDATE SET
    requested_by = excluded.requested_by,
    link         = excluded.link,
    note         = excluded.note,
    created_at   = CURRENT_TIMESTAMP`
	_, err := q.db.ExecContext(ctx, stmt, marketID, requestedBy, link, note)
	return err
}

// DeleteResolutionRequestDetails removes resolution-request metadata when a request
// is denied (market returns to active) or when the market is resolved.
func (q *Queries) DeleteResolutionRequestDetails(ctx context.Context, marketID int64) error {
	_, err := q.db.ExecContext(ctx,
		`DELETE FROM resolution_request_details WHERE market_id = ?`, marketID)
	return err
}

// GetUserPositionOrZero returns the user's position in a market, or zeros if none exists.
func (q *Queries) GetUserPositionOrZero(ctx context.Context, userID, marketID int64) (Position, error) {
	pos, err := q.GetUserPosition(ctx, GetUserPositionParams{UserID: userID, MarketID: marketID})
	if err == sql.ErrNoRows {
		return Position{UserID: userID, MarketID: marketID}, nil
	}
	return pos, err
}

// ─── Reports ──────────────────────────────────────────────────────────────────

// ReportRow represents a submitted report with reporter and market name joins.
type ReportRow struct {
	ID           int64     `json:"id"`
	ReporterID   int64     `json:"reporter_id"`
	ReporterName string    `json:"reporter_name"`
	MarketID     int64     `json:"market_id"`
	MarketTitle  string    `json:"market_title"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateReport inserts a new report from a user about a market.
func (q *Queries) CreateReport(ctx context.Context, reporterID, marketID int64, reason string) (int64, error) {
	const stmt = `INSERT INTO reports (reporter_id, market_id, reason) VALUES (?, ?, ?)`
	res, err := q.db.ExecContext(ctx, stmt, reporterID, marketID, reason)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListPendingReports returns all reports that are still pending moderator review.
func (q *Queries) ListPendingReports(ctx context.Context) ([]ReportRow, error) {
	const query = `
SELECT r.id, r.reporter_id, u.display_name AS reporter_name,
       r.market_id, m.title AS market_title,
       r.reason, r.status, r.created_at
FROM reports r
JOIN users u   ON u.id = r.reporter_id
JOIN markets m ON m.id = r.market_id
WHERE r.status = 'pending'
ORDER BY r.created_at ASC`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ReportRow, 0)
	for rows.Next() {
		var i ReportRow
		if err := rows.Scan(&i.ID, &i.ReporterID, &i.ReporterName,
			&i.MarketID, &i.MarketTitle, &i.Reason, &i.Status, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// UpdateReportStatus sets a report's status to 'reviewed' or 'dismissed'.
func (q *Queries) UpdateReportStatus(ctx context.Context, reportID int64, status string) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE reports SET status = ? WHERE id = ?`, status, reportID)
	return err
}

// ─── Badges ───────────────────────────────────────────────────────────────────

// BadgeRow holds a badge awarded to a user.
type BadgeRow struct {
	BadgeKey  string    `json:"badge_key"`
	AwardedAt time.Time `json:"awarded_at"`
}

// GetUserBadges returns all badges for a user, newest first.
func (q *Queries) GetUserBadges(ctx context.Context, userID int64) ([]BadgeRow, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT badge_key, awarded_at FROM user_badges WHERE user_id = ? ORDER BY awarded_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BadgeRow, 0)
	for rows.Next() {
		var b BadgeRow
		if err := rows.Scan(&b.BadgeKey, &b.AwardedAt); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

// AwardBadgeIfNew inserts a badge for a user, ignoring duplicates (UNIQUE constraint).
func (q *Queries) AwardBadgeIfNew(ctx context.Context, userID int64, badgeKey string) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_badges (user_id, badge_key) VALUES (?, ?)`,
		userID, badgeKey)
	return err
}

// CountUserTrades returns the total number of trades a user has made.
func (q *Queries) CountUserTrades(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM trades WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// CountCorrectPredictions returns how many resolved markets the user predicted correctly
// (held more shares on the winning side when the market closed).
func (q *Queries) CountCorrectPredictions(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM positions p
JOIN markets m ON m.id = p.market_id
WHERE p.user_id = ?
  AND m.status = 'resolved'
  AND (
      (m.resolution = 'yes' AND p.yes_shares > p.no_shares) OR
      (m.resolution = 'no'  AND p.no_shares  > p.yes_shares)
  )`, userID).Scan(&n)
	return n, err
}

// CountMarketsWithYES returns how many distinct markets the user has bought YES shares in.
func (q *Queries) CountMarketsWithYES(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM positions WHERE user_id = ? AND yes_shares > 0`, userID).Scan(&n)
	return n, err
}

// CountMarketsWithNO returns how many distinct markets the user has bought NO shares in.
func (q *Queries) CountMarketsWithNO(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM positions WHERE user_id = ? AND no_shares > 0`, userID).Scan(&n)
	return n, err
}

// ─── Dialect-aware queries (strftime / INSERT OR IGNORE differ by backend) ────
//
// These were removed from the sqlc sources so that both SQLite and PostgreSQL
// can be supported at runtime via the pgDB/pgTx ? → $N rewriting wrapper.

// UpdateUserLastLoginParams mirrors the struct sqlc would have generated.
type UpdateUserLastLoginParams struct {
	DisplayName      string  `json:"display_name"`
	Email            string  `json:"email"`
	RsiHandle        *string `json:"rsi_handle"`
	RsiVerifiedAt    *string `json:"rsi_verified_at"`
	RsiEnlisted      *string `json:"rsi_enlisted"`
	RsiCitizenRecord *string `json:"rsi_citizen_record"`
	AvatarUrl        *string `json:"avatar_url"`
	IsRsiVerified    int64   `json:"is_rsi_verified"`
	ID               int64   `json:"id"`
}

// UpdateUserLastLogin bumps last_login_at to the current time and refreshes the
// SCID profile fields. Uses NOW() on PostgreSQL and strftime on SQLite.
func (q *Queries) UpdateUserLastLogin(ctx context.Context, arg UpdateUserLastLoginParams) error {
	ts := "strftime('%Y-%m-%dT%H:%M:%SZ', 'now')"
	if q.isPG() {
		ts = "NOW()"
	}
	stmt := `UPDATE users SET last_login_at = ` + ts + `,
    display_name       = ?,
    email              = ?,
    rsi_handle         = ?,
    rsi_verified_at    = ?,
    rsi_enlisted       = ?,
    rsi_citizen_record = ?,
    avatar_url         = ?,
    is_rsi_verified    = ?
WHERE id = ?`
	_, err := q.db.ExecContext(ctx, stmt,
		arg.DisplayName, arg.Email,
		arg.RsiHandle, arg.RsiVerifiedAt, arg.RsiEnlisted, arg.RsiCitizenRecord,
		arg.AvatarUrl, arg.IsRsiVerified, arg.ID,
	)
	return err
}

// ResolveMarketParams mirrors the struct sqlc would have generated.
type ResolveMarketParams struct {
	Resolution         *string `json:"resolution"`
	ResolvedBy         *int64  `json:"resolved_by"`
	ResolutionEvidence *string `json:"resolution_evidence"`
	ID                 int64   `json:"id"`
}

// ResolveMarket marks a market as resolved. Uses NOW() on PostgreSQL and
// strftime on SQLite for the resolved_at timestamp.
func (q *Queries) ResolveMarket(ctx context.Context, arg ResolveMarketParams) error {
	ts := "strftime('%Y-%m-%dT%H:%M:%SZ', 'now')"
	if q.isPG() {
		ts = "NOW()"
	}
	stmt := `UPDATE markets
SET status              = 'resolved',
    resolution          = ?,
    resolved_by         = ?,
    resolution_evidence = ?,
    resolved_at         = ` + ts + `
WHERE id = ?`
	_, err := q.db.ExecContext(ctx, stmt,
		arg.Resolution, arg.ResolvedBy, arg.ResolutionEvidence, arg.ID,
	)
	return err
}

// ExpirePendingMarkets auto-cancels markets that have been in pending_review for
// more than 14 days. Uses an INTERVAL expression on PostgreSQL.
func (q *Queries) ExpirePendingMarkets(ctx context.Context) error {
	var stmt string
	if q.isPG() {
		stmt = `UPDATE markets SET status = 'cancelled'
WHERE status = 'pending_review' AND created_at < NOW() - INTERVAL '14 days'`
	} else {
		stmt = `UPDATE markets SET status = 'cancelled'
WHERE status = 'pending_review'
  AND created_at < strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-14 days')`
	}
	_, err := q.db.ExecContext(ctx, stmt)
	return err
}

// ExpireOverdueActiveMarkets moves active markets past their deadline into
// deadline_passed state.
func (q *Queries) ExpireOverdueActiveMarkets(ctx context.Context) error {
	var stmt string
	if q.isPG() {
		stmt = `UPDATE markets SET status = 'deadline_passed'
WHERE status = 'active' AND resolution_deadline < NOW()`
	} else {
		stmt = `UPDATE markets SET status = 'deadline_passed'
WHERE status = 'active'
  AND resolution_deadline < strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`
	}
	_, err := q.db.ExecContext(ctx, stmt)
	return err
}

// InsertPatchParams mirrors the struct sqlc would have generated.
type InsertPatchParams struct {
	Title        string `json:"title"`
	PatchVersion string `json:"patch_version"`
	ThreadUrl    string `json:"thread_url"`
}

// InsertPatch inserts a new detected patch, ignoring duplicates (keyed on
// thread_url UNIQUE constraint). Uses INSERT OR IGNORE on SQLite and
// ON CONFLICT DO NOTHING on PostgreSQL.
func (q *Queries) InsertPatch(ctx context.Context, arg InsertPatchParams) error {
	var stmt string
	if q.isPG() {
		stmt = `INSERT INTO detected_patches (title, patch_version, thread_url)
VALUES (?, ?, ?) ON CONFLICT DO NOTHING`
	} else {
		stmt = `INSERT OR IGNORE INTO detected_patches (title, patch_version, thread_url)
VALUES (?, ?, ?)`
	}
	_, err := q.db.ExecContext(ctx, stmt, arg.Title, arg.PatchVersion, arg.ThreadUrl)
	return err
}

// ─── Weekly payout ────────────────────────────────────────────────────────────

// WeeklyPayoutAlreadyRan returns true if the weekly payout for the given week key
// has already been executed (idempotent guard).
func (q *Queries) WeeklyPayoutAlreadyRan(ctx context.Context, weekKey string) (bool, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM weekly_payout_log WHERE week_key = ?`, weekKey).Scan(&n)
	return n > 0, err
}

// RunWeeklyPayout adds 200 bUEC to every user and records the run.
func (q *Queries) RunWeeklyPayout(ctx context.Context, weekKey string) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE users SET balance = balance + 200`)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	_, err = q.db.ExecContext(ctx,
		`INSERT INTO weekly_payout_log (week_key, user_count) VALUES (?, ?)`,
		weekKey, count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
