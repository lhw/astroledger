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
	ResolvedOutcomeID  *int64     `json:"resolved_outcome_id"`
	CreatedBy          int64      `json:"created_by"`
	ResolvedBy         *int64     `json:"resolved_by"`
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	LiquidityParam     float64    `json:"liquidity_param"`
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
       m.status, m.resolved_outcome_id, m.created_by, m.resolved_by,
       m.created_at, m.resolved_at, m.liquidity_param,
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
       m.status, m.resolved_outcome_id, m.created_by, m.resolved_by,
       m.created_at, m.resolved_at, m.liquidity_param,
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
				&i.Status, &i.ResolvedOutcomeID, &i.CreatedBy, &i.ResolvedBy,
				&i.CreatedAt, &i.ResolvedAt, &i.LiquidityParam,
				&i.CreatorName,
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
				&i.Status, &i.ResolvedOutcomeID, &i.CreatedBy, &i.ResolvedBy,
				&i.CreatedAt, &i.ResolvedAt, &i.LiquidityParam,
				&i.CreatorName,
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

// GetUserPositionOrZero returns the user's position in a market for a specific outcome,
// or a zero position if none exists. Uses outcome_id=0 as a sentinel "sum all" check
// for callers that only need to know if ANY position exists.
func (q *Queries) GetUserPositionOrZero(ctx context.Context, userID, marketID int64) (Position, error) {
	// Return the first position found for this user+market (any outcome).
	// Used only to check whether the user has any stake in the market.
	const stmt = `SELECT user_id, market_id, outcome_id, shares FROM positions
WHERE user_id = ? AND market_id = ? LIMIT 1`
	var p Position
	err := q.db.QueryRowContext(ctx, stmt, userID, marketID).Scan(&p.UserID, &p.MarketID, &p.OutcomeID, &p.Shares)
	if err == sql.ErrNoRows {
		return Position{UserID: userID, MarketID: marketID}, nil
	}
	return p, err
}

// ─── Reports ──────────────────────────────────────────────────────────────────

// ReportRow represents a submitted report with reporter and market name joins.
type ReportRow struct {
	ID           int64     `json:"id"`
	ReporterID   int64     `json:"reporter_id"`
	ReporterName string    `json:"reporter_name"`
	MarketID     int64     `json:"market_id"`
	MarketTitle  string    `json:"market_title"`
	Category     string    `json:"category"`
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
	   m.category,
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
			&i.MarketID, &i.MarketTitle, &i.Category, &i.Reason, &i.Status, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// LogModAuditParams are the fields persisted in mod_audit.
type LogModAuditParams struct {
	ActionType string  `json:"action_type"`
	TargetType string  `json:"target_type"`
	TargetID   int64   `json:"target_id"`
	ModUserID  int64   `json:"mod_user_id"`
	Note       *string `json:"note"`
}

// LogModAudit writes a moderation audit record.
func (q *Queries) LogModAudit(ctx context.Context, arg LogModAuditParams) error {
	const stmt = `
INSERT INTO mod_audit (action_type, target_type, target_id, mod_user_id, note)
VALUES (?, ?, ?, ?, ?)`
	_, err := q.db.ExecContext(ctx, stmt,
		arg.ActionType,
		arg.TargetType,
		arg.TargetID,
		arg.ModUserID,
		arg.Note,
	)
	return err
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
// (held any position in the winning outcome when the market resolved).
func (q *Queries) CountCorrectPredictions(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT p.market_id) FROM positions p
JOIN markets m ON m.id = p.market_id
WHERE p.user_id = ?
  AND m.status = 'resolved'
  AND p.outcome_id = m.resolved_outcome_id
  AND p.shares > 0`, userID).Scan(&n)
	return n, err
}

// CountMarketsWithYES returns how many distinct markets the user has bought any shares in.
// (legacy compat — now checks any position across all outcomes)
func (q *Queries) CountMarketsWithYES(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT market_id) FROM positions WHERE user_id = ? AND shares > 0`, userID).Scan(&n)
	return n, err
}

// CountMarketsWithNO returns how many distinct markets the user has bought non-winning shares in.
// (legacy compat — now counts markets where user holds any position)
func (q *Queries) CountMarketsWithNO(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT market_id) FROM positions WHERE user_id = ? AND shares > 0`, userID).Scan(&n)
	return n, err
}

// CountLiveMarketsCreatedBy returns how many markets the user has submitted that are
// currently active, resolved, resolution_requested, or deadline_passed (i.e. went live).
func (q *Queries) CountLiveMarketsCreatedBy(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM markets
WHERE created_by = ?
  AND status IN ('active','resolved','resolution_requested','deadline_passed')`, userID).Scan(&n)
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
	ResolvedOutcomeID  *int64  `json:"resolved_outcome_id"`
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
    resolved_outcome_id = ?,
    resolved_by         = ?,
    resolution_evidence = ?,
    resolved_at         = ` + ts + `
WHERE id = ?`
	_, err := q.db.ExecContext(ctx, stmt,
		arg.ResolvedOutcomeID, arg.ResolvedBy, arg.ResolutionEvidence, arg.ID,
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

// WeeklyPayoutAmount is the number of ScollyBucks awarded to every user in the weekly payout.
const WeeklyPayoutAmount int64 = 200

// WeeklyPayoutAlreadyRan returns true if the weekly payout for the given week key
// has already been executed (idempotent guard).
func (q *Queries) WeeklyPayoutAlreadyRan(ctx context.Context, weekKey string) (bool, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM weekly_payout_log WHERE week_key = ?`, weekKey).Scan(&n)
	return n > 0, err
}

// RunWeeklyPayout adds WeeklyPayoutAmount bUEC to every user and records the run.
func (q *Queries) RunWeeklyPayout(ctx context.Context, weekKey string) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE users SET balance = balance + ?`, WeeklyPayoutAmount)
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

// CountBadgePurchases returns how many users currently own the given badge key.
func (q *Queries) CountBadgePurchases(ctx context.Context, badgeKey string) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_badges WHERE badge_key = ?`, badgeKey).Scan(&n)
	return n, err
}

// GetUserTopBadge returns the badge_key of the highest-tier badge the user owns,
// or an empty string if the user has no badges. Tier order is determined by the
// caller via BadgeKeysMap.
func (q *Queries) GetUserBadgeKeys(ctx context.Context, userID int64) ([]string, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT badge_key FROM user_badges WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// GetUserActiveBadge returns the user's chosen active_badge_key (or "" if unset).
func (q *Queries) GetUserActiveBadge(ctx context.Context, userID int64) (string, error) {
	var key string
	err := q.db.QueryRowContext(ctx,
		`SELECT active_badge_key FROM users WHERE id = ?`, userID).Scan(&key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return key, err
}

// SetUserActiveBadge updates the active_badge_key for a user.
func (q *Queries) SetUserActiveBadge(ctx context.Context, userID int64, key string) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE users SET active_badge_key = ? WHERE id = ?`, key, userID)
	return err
}

// GetUsersActiveBadges returns a map of userID → active_badge_key for the given set of user IDs.
func (q *Queries) GetUsersActiveBadges(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	if len(userIDs) == 0 {
		return map[int64]string{}, nil
	}
	// Build a parameterised IN clause.
	args := make([]any, len(userIDs))
	ph := make([]byte, 0, len(userIDs)*2)
	for i, id := range userIDs {
		args[i] = id
		if i > 0 {
			ph = append(ph, ',')
		}
		ph = append(ph, '?')
	}
	query := `SELECT id, active_badge_key FROM users WHERE id IN (` + string(ph) + `)`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]string, len(userIDs))
	for rows.Next() {
		var uid int64
		var key string
		if err := rows.Scan(&uid, &key); err != nil {
			return nil, err
		}
		m[uid] = key
	}
	return m, rows.Err()
}

// ─── Badge Releases ───────────────────────────────────────────────────────────

// BadgeReleaseRow represents a row from the badge_releases table.
type BadgeReleaseRow struct {
	ID         int64      `json:"id"`
	BadgeKey   string     `json:"badge_key"`
	Price      int64      `json:"price"`
	Stock      *int       `json:"stock"`
	ReleasedAt time.Time  `json:"released_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Active     bool       `json:"active"`
	Notes      *string    `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
}

func scanBadgeRelease(row func(dest ...any) error) (BadgeReleaseRow, error) {
	var r BadgeReleaseRow
	var activeInt int64
	err := row(
		&r.ID, &r.BadgeKey, &r.Price, &r.Stock,
		&r.ReleasedAt, &r.ExpiresAt, &activeInt, &r.Notes, &r.CreatedAt,
	)
	if err != nil {
		return r, err
	}
	r.Active = activeInt == 1
	return r, nil
}

const badgeReleaseSelectCols = `id, badge_key, price, stock, released_at, expires_at, active, notes, created_at`

// ListBadgeReleases returns all releases ordered newest-first (admin use).
func (q *Queries) ListBadgeReleases(ctx context.Context) ([]BadgeReleaseRow, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT `+badgeReleaseSelectCols+` FROM badge_releases ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BadgeReleaseRow, 0)
	for rows.Next() {
		r, err := scanBadgeRelease(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// GetBadgeReleaseByID returns a single release by primary key.
func (q *Queries) GetBadgeReleaseByID(ctx context.Context, id int64) (BadgeReleaseRow, error) {
	return scanBadgeRelease(q.db.QueryRowContext(ctx,
		`SELECT `+badgeReleaseSelectCols+` FROM badge_releases WHERE id = ?`, id).Scan)
}

// GetActiveBadgeReleases returns all releases that are currently purchasable:
// active=1, released_at <= now, (expires_at IS NULL OR expires_at > now).
func (q *Queries) GetActiveBadgeReleases(ctx context.Context) ([]BadgeReleaseRow, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	rows, err := q.db.QueryContext(ctx, `
SELECT `+badgeReleaseSelectCols+` FROM badge_releases
WHERE active = 1
  AND released_at <= ?
  AND (expires_at IS NULL OR expires_at > ?)
ORDER BY released_at DESC`, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BadgeReleaseRow, 0)
	for rows.Next() {
		r, err := scanBadgeRelease(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// GetActiveBadgeReleaseForKey returns the active release for a given badge_key, if any.
func (q *Queries) GetActiveBadgeReleaseForKey(ctx context.Context, badgeKey string) (BadgeReleaseRow, bool, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	row, err := scanBadgeRelease(q.db.QueryRowContext(ctx, `
SELECT `+badgeReleaseSelectCols+` FROM badge_releases
WHERE badge_key = ? AND active = 1
  AND released_at <= ?
  AND (expires_at IS NULL OR expires_at > ?)
LIMIT 1`, badgeKey, now, now).Scan)
	if err == sql.ErrNoRows {
		return BadgeReleaseRow{}, false, nil
	}
	if err != nil {
		return BadgeReleaseRow{}, false, err
	}
	return row, true, nil
}

// CreateBadgeReleaseParams holds fields for inserting a new badge release.
type CreateBadgeReleaseParams struct {
	BadgeKey   string
	Price      int64
	Stock      *int
	ReleasedAt time.Time
	ExpiresAt  *time.Time
	Notes      *string
}

// CreateBadgeRelease inserts a new badge release row and returns its ID.
func (q *Queries) CreateBadgeRelease(ctx context.Context, p CreateBadgeReleaseParams) (int64, error) {
	releasedAt := p.ReleasedAt.UTC().Format("2006-01-02T15:04:05Z")
	var expiresAt *string
	if p.ExpiresAt != nil {
		s := p.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
		expiresAt = &s
	}
	res, err := q.db.ExecContext(ctx, `
INSERT INTO badge_releases (badge_key, price, stock, released_at, expires_at, active, notes)
VALUES (?, ?, ?, ?, ?, 1, ?)`,
		p.BadgeKey, p.Price, p.Stock, releasedAt, expiresAt, p.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateBadgeReleaseParams holds mutable fields for an existing badge release.
type UpdateBadgeReleaseParams struct {
	ID         int64
	Price      int64
	Stock      *int
	ReleasedAt time.Time
	ExpiresAt  *time.Time
	Active     bool
	Notes      *string
}

// UpdateBadgeRelease overwrites the mutable fields of an existing release.
func (q *Queries) UpdateBadgeRelease(ctx context.Context, p UpdateBadgeReleaseParams) error {
	releasedAt := p.ReleasedAt.UTC().Format("2006-01-02T15:04:05Z")
	var expiresAt *string
	if p.ExpiresAt != nil {
		s := p.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
		expiresAt = &s
	}
	activeInt := 0
	if p.Active {
		activeInt = 1
	}
	_, err := q.db.ExecContext(ctx, `
UPDATE badge_releases
SET price = ?, stock = ?, released_at = ?, expires_at = ?, active = ?, notes = ?
WHERE id = ?`,
		p.Price, p.Stock, releasedAt, expiresAt, activeInt, p.Notes, p.ID)
	return err
}

// ArchiveBadgeRelease sets active=0 on a release (soft-delete).
func (q *Queries) ArchiveBadgeRelease(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE badge_releases SET active = 0 WHERE id = ?`, id)
	return err
}

// AwardBadgePurchased inserts a purchased badge with the price paid, ignoring duplicates.
func (q *Queries) AwardBadgePurchased(ctx context.Context, userID int64, badgeKey string, price int64) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_badges (user_id, badge_key, purchase_price) VALUES (?, ?, ?)`,
		userID, badgeKey, price)
	return err
}

// GetUserBadgePurchasePrices returns a map of badge_key → purchase_price for all badges owned by a user.
func (q *Queries) GetUserBadgePurchasePrices(ctx context.Context, userID int64) (map[string]int64, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT badge_key, purchase_price FROM user_badges WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int64)
	for rows.Next() {
		var key string
		var price int64
		if err := rows.Scan(&key, &price); err != nil {
			return nil, err
		}
		m[key] = price
	}
	return m, rows.Err()
}

// ─── Enriched User Positions ─────────────────────────────────────────────────

// EnrichedPositionRow extends the basic position with cost basis (from trades)
// and resolved market info so the frontend can compute P&L.
type EnrichedPositionRow struct {
	UserID            int64   `json:"user_id"`
	MarketID          int64   `json:"market_id"`
	OutcomeID         int64   `json:"outcome_id"`
	Shares            float64 `json:"shares"`
	MarketTitle       string  `json:"market_title"`
	MarketStatus      string  `json:"market_status"`
	LiquidityParam    float64 `json:"liquidity_param"`
	OutcomeLabel      string  `json:"outcome_label"`
	ResolvedOutcomeID *int64  `json:"resolved_outcome_id"`
	CostBasis         int64   `json:"cost_basis"`
}

// GetUserPositionsEnriched returns all non-zero positions for a user, enriched with
// cost basis (net bUEC spent on buys minus sell proceeds for this outcome) and the
// market's resolved_outcome_id for P&L display.
func (q *Queries) GetUserPositionsEnriched(ctx context.Context, userID int64) ([]EnrichedPositionRow, error) {
	const query = `
SELECT p.user_id, p.market_id, p.outcome_id, p.shares,
       m.title AS market_title, m.status AS market_status,
       m.liquidity_param, o.label AS outcome_label,
       m.resolved_outcome_id,
       COALESCE(SUM(CASE WHEN t.action = 'buy' THEN t.cost ELSE -t.cost END), 0) AS cost_basis
FROM positions p
JOIN markets m ON m.id = p.market_id
JOIN market_outcomes o ON o.id = p.outcome_id
LEFT JOIN trades t ON t.user_id = p.user_id
                   AND t.market_id = p.market_id
                   AND t.outcome_id = p.outcome_id
WHERE p.user_id = ?
  AND p.shares > 0
GROUP BY p.user_id, p.market_id, p.outcome_id
ORDER BY m.status ASC, m.id DESC`

	rows, err := q.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]EnrichedPositionRow, 0)
	for rows.Next() {
		var r EnrichedPositionRow
		if err := rows.Scan(
			&r.UserID, &r.MarketID, &r.OutcomeID, &r.Shares,
			&r.MarketTitle, &r.MarketStatus, &r.LiquidityParam,
			&r.OutcomeLabel, &r.ResolvedOutcomeID, &r.CostBasis,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Trending ─────────────────────────────────────────────────────────────────

// TrendingMarketRow is a market enriched with recent (last 24h) trading activity.
type TrendingMarketRow struct {
	ID                  int64      `json:"id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Category            string     `json:"category"`
	ResolutionCriteria  string     `json:"resolution_criteria"`
	ResolutionDeadline  time.Time  `json:"resolution_deadline"`
	Status              string     `json:"status"`
	ResolvedOutcomeID   *int64     `json:"resolved_outcome_id"`
	CreatedBy           int64      `json:"created_by"`
	ResolvedBy          *int64     `json:"resolved_by"`
	CreatedAt           time.Time  `json:"created_at"`
	ResolvedAt          *time.Time `json:"resolved_at"`
	LiquidityParam      float64    `json:"liquidity_param"`
	ResolutionType      string     `json:"resolution_type"`
	ResolutionThreshold *string    `json:"resolution_threshold"`
	ResolutionEvidence  *string    `json:"resolution_evidence"`
	CreatorName         string     `json:"creator_name"`
	RecentTradeCount    int64      `json:"recent_trade_count"`
	RecentVolume        int64      `json:"recent_volume"`
}

// ListTrendingMarkets returns up to limit active markets ordered by trade volume
// in the last 24 hours.
func (q *Queries) ListTrendingMarkets(ctx context.Context, limit int64) ([]TrendingMarketRow, error) {
	var window string
	if q.isPG() {
		window = "NOW() - INTERVAL '24 hours'"
	} else {
		window = "datetime('now', '-24 hours')"
	}
	query := `
SELECT m.id, m.title, m.description, m.category,
       m.resolution_criteria, m.resolution_deadline,
       m.status, m.resolved_outcome_id, m.created_by, m.resolved_by,
       m.created_at, m.resolved_at, m.liquidity_param,
       m.resolution_type, m.resolution_threshold, m.resolution_evidence,
       u.display_name AS creator_name,
       COUNT(t.id)              AS recent_trade_count,
       COALESCE(SUM(t.cost), 0) AS recent_volume
FROM markets m
JOIN users u ON u.id = m.created_by
LEFT JOIN trades t ON t.market_id = m.id AND t.created_at >= ` + window + `
WHERE m.status = 'active'
GROUP BY m.id
ORDER BY recent_trade_count DESC, recent_volume DESC
LIMIT ?`

	rows, err := q.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrendingMarketRow, 0)
	for rows.Next() {
		var i TrendingMarketRow
		if err := rows.Scan(
			&i.ID, &i.Title, &i.Description, &i.Category,
			&i.ResolutionCriteria, &i.ResolutionDeadline,
			&i.Status, &i.ResolvedOutcomeID, &i.CreatedBy, &i.ResolvedBy,
			&i.CreatedAt, &i.ResolvedAt, &i.LiquidityParam,
			&i.ResolutionType, &i.ResolutionThreshold, &i.ResolutionEvidence,
			&i.CreatorName,
			&i.RecentTradeCount, &i.RecentVolume,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
