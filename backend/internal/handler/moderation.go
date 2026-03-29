package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
	"github.com/lhw/astroledger/internal/service"
)

// ModerationHandler handles report submission and mod-only report management.
type ModerationHandler struct {
	queries  *db.Queries
	sqlDB    *sql.DB
	badgeSvc *service.BadgeService
}

// NewModerationHandler creates a ModerationHandler.
func NewModerationHandler(q *db.Queries, sqlDB *sql.DB, badgeSvc *service.BadgeService) *ModerationHandler {
	return &ModerationHandler{queries: q, sqlDB: sqlDB, badgeSvc: badgeSvc}
}

// SubmitReport lets any authenticated user flag a market for mod review.
// POST /api/reports
func (h *ModerationHandler) SubmitReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		MarketID int64  `json:"market_id"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.MarketID == 0 {
		respondError(w, http.StatusBadRequest, "market_id required")
		return
	}
	if len(body.Reason) < 5 {
		respondError(w, http.StatusBadRequest, "reason must be at least 5 characters")
		return
	}
	if len(body.Reason) > 500 {
		respondError(w, http.StatusBadRequest, "reason too long (max 500 chars)")
		return
	}

	id, err := h.queries.CreateReport(r.Context(), claims.UserID, body.MarketID, body.Reason)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// ListReports returns all pending reports (mod-only).
// GET /api/mod/reports
func (h *ModerationHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.queries.ListPendingReports(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, reports)
}

// ReviewReport marks a report as reviewed (mod-only).
// POST /api/mod/reports/:id/review
func (h *ModerationHandler) ReviewReport(w http.ResponseWriter, r *http.Request) {
	h.setReportStatus(w, r, "reviewed")
}

// DismissReport marks a report as dismissed (mod-only).
// POST /api/mod/reports/:id/dismiss
func (h *ModerationHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	h.setReportStatus(w, r, "dismissed")
}

func (h *ModerationHandler) setReportStatus(w http.ResponseWriter, r *http.Request, status string) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	if err := h.queries.UpdateReportStatus(r.Context(), id, status); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	note := "report marked " + status
	if err := h.queries.LogModAudit(r.Context(), db.LogModAuditParams{
		ActionType: "report_" + status,
		TargetType: "report",
		TargetID:   id,
		ModUserID:  claims.UserID,
		Note:       &note,
	}); err != nil {
		slog.Warn("mod audit log failed", "action", "report_"+status, "report_id", id, "mod_id", claims.UserID, "err", err)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": status})
}

// GetMyBadges returns the badge list for the authenticated user.
// GET /api/me/badges
func (h *ModerationHandler) GetMyBadges(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.sendBadges(w, r, claims.UserID)
}

// GetUserBadges returns the public badge list for any user by ID.
// GET /api/users/:id/badges
func (h *ModerationHandler) GetUserBadges(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	h.sendBadges(w, r, id)
}

func (h *ModerationHandler) sendBadges(w http.ResponseWriter, r *http.Request, userID int64) {
	badges, err := h.queries.GetUserBadges(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Enrich with static definition fields.
	type richBadge struct {
		BadgeKey    string `json:"badge_key"`
		AwardedAt   string `json:"awarded_at"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Tier        int    `json:"tier"`
		Cost        int64  `json:"cost"`
		Purchasable bool   `json:"purchasable"`
	}
	result := make([]richBadge, 0, len(badges))
	for _, b := range badges {
		def, ok := service.BadgeKeysMap[b.BadgeKey]
		if !ok {
			continue // skip unknown badge keys
		}
		result = append(result, richBadge{
			BadgeKey:    b.BadgeKey,
			AwardedAt:   b.AwardedAt.Format("2006-01-02T15:04:05Z"),
			Title:       def.Title,
			Description: def.Description,
			Tier:        def.Tier,
			Cost:        def.Cost,
			Purchasable: def.Purchasable,
		})
	}
	respondJSON(w, http.StatusOK, result)
}

// GetStoreBadges returns the list of purchasable FOMO store badges with owned status.
// Only badges with an active DB release (created via the admin panel) are included.
// GET /api/fomo
func (h *ModerationHandler) GetStoreBadges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := middleware.GetClaims(r)

	// Build owned set (empty if unauthenticated).
	owned := map[string]bool{}
	if claims != nil {
		badges, err := h.queries.GetUserBadges(ctx, claims.UserID)
		if err == nil {
			for _, b := range badges {
				owned[b.BadgeKey] = true
			}
		}
	}

	type storeBadge struct {
		BadgeKey       string  `json:"badge_key"`
		Title          string  `json:"title"`
		Description    string  `json:"description"`
		Tier           int     `json:"tier"`
		Cost           int64   `json:"cost"`
		Owned          bool    `json:"owned"`
		Stock          *int    `json:"stock,omitempty"`
		RemainingStock *int64  `json:"remaining_stock,omitempty"`
		AvailableUntil *string `json:"available_until,omitempty"`
		Expired        bool    `json:"expired"`
	}

	releases, err := h.queries.GetActiveBadgeReleases(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	now := time.Now()
	result := make([]storeBadge, 0, len(releases))
	for _, rel := range releases {
		def, ok := service.BadgeKeysMap[rel.BadgeKey]
		if !ok {
			continue // badge key no longer in catalog — skip gracefully
		}
		sb := storeBadge{
			BadgeKey:    rel.BadgeKey,
			Title:       def.Title,
			Description: def.Description,
			Tier:        def.Tier,
			Cost:        rel.Price,
			Owned:       owned[rel.BadgeKey],
		}
		if rel.Stock != nil {
			sb.Stock = rel.Stock
			sold, err := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
			if err == nil {
				rem := int64(*rel.Stock) - sold
				if rem < 0 {
					rem = 0
				}
				sb.RemainingStock = &rem
			}
		}
		if rel.ExpiresAt != nil {
			s := rel.ExpiresAt.Format("2006-01-02T15:04:05Z")
			sb.AvailableUntil = &s
			sb.Expired = now.After(*rel.ExpiresAt)
		}
		result = append(result, sb)
	}
	respondJSON(w, http.StatusOK, result)
}

// PurchaseBadge lets an authenticated user buy a FOMO store badge.
// POST /api/fomo/purchase
func (h *ModerationHandler) PurchaseBadge(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	var body struct {
		BadgeKey string `json:"badge_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.BadgeKey == "" {
		respondError(w, http.StatusBadRequest, "badge_key is required")
		return
	}

	// Validate against catalog.
	def, ok := service.BadgeKeysMap[body.BadgeKey]
	if !ok {
		respondError(w, http.StatusBadRequest, "badge not available for purchase")
		return
	}

	// Require an active release for this badge key.
	release, hasRelease, err := h.queries.GetActiveBadgeReleaseForKey(ctx, body.BadgeKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !hasRelease {
		respondError(w, http.StatusGone, "this badge is not currently available")
		return
	}

	// Begin transaction for atomicity (prevents TOCTOU races).
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer func() { _ = tx.Rollback() }()

	qTx := h.queries.WithBoundTx(tx)

	// Check stock against release limit (within transaction).
	if release.Stock != nil {
		sold, err := qTx.CountBadgePurchases(ctx, def.Key)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if sold >= int64(*release.Stock) {
			respondError(w, http.StatusConflict, "this badge is sold out")
			return
		}
	}

	// Fetch current balance.
	user, err := qTx.GetUserByID(ctx, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if user.Balance < release.Price {
		respondError(w, http.StatusPaymentRequired, "insufficient balance")
		return
	}

	// Check already owned (within transaction).
	badges, err := qTx.GetUserBadges(ctx, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	for _, b := range badges {
		if b.BadgeKey == def.Key {
			respondError(w, http.StatusConflict, "badge already owned")
			return
		}
	}

	// Deduct balance.
	if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{
		Balance: -release.Price,
		ID:      claims.UserID,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Award badge with purchase price for admiral rank tracking.
	if err := qTx.AwardBadgePurchased(ctx, claims.UserID, def.Key, release.Price); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Commit transaction.
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "purchased"})

	// Non-blocking: check admiral rank milestones after the purchase.
	userID := claims.UserID
	go h.badgeSvc.CheckAndAwardAdmiralRanks(context.Background(), userID)
}

// GetAdmiralRanks returns the admiral rank progression for the authenticated user (or
// anonymous spend=0 if unauthenticated).
// GET /api/admiral
func (h *ModerationHandler) GetAdmiralRanks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := middleware.GetClaims(r)

	var lifetimeSpend int64
	owned := map[string]bool{}
	if claims != nil {
		spend, err := h.badgeSvc.ComputeLifetimeSpend(ctx, claims.UserID)
		if err == nil {
			lifetimeSpend = spend
		}
		if badges, err := h.queries.GetUserBadges(ctx, claims.UserID); err == nil {
			for _, b := range badges {
				owned[b.BadgeKey] = true
			}
		}
	}

	type admiralRank struct {
		BadgeKey       string `json:"badge_key"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		Tier           int    `json:"tier"`
		SpendThreshold int64  `json:"spend_threshold"`
		Owned          bool   `json:"owned"`
	}

	ranks := service.AdmiralRankBadges()
	result := make([]admiralRank, 0, len(ranks))
	for _, b := range ranks {
		result = append(result, admiralRank{
			BadgeKey:       b.Key,
			Title:          b.Title,
			Description:    b.Description,
			Tier:           b.Tier,
			SpendThreshold: b.SpendThreshold,
			Owned:          owned[b.Key],
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"ranks":          result,
		"lifetime_spend": lifetimeSpend,
	})
}

// SetActiveBadge sets the authenticated user's active (displayed) badge.
// PUT /api/me/badge
// Body: {"badge_key": "ensign"}  — send "" to clear.
func (h *ModerationHandler) SetActiveBadge(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		BadgeKey string `json:"badge_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Allow clearing the badge by sending "".
	if body.BadgeKey != "" {
		// Validate: the user must own this badge and it must be a known key.
		if _, ok := service.BadgeKeysMap[body.BadgeKey]; !ok {
			respondError(w, http.StatusBadRequest, "unknown badge key")
			return
		}
		owned, err := h.queries.GetUserBadges(r.Context(), claims.UserID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		found := false
		for _, b := range owned {
			if b.BadgeKey == body.BadgeKey {
				found = true
				break
			}
		}
		if !found {
			respondError(w, http.StatusForbidden, "badge not owned")
			return
		}
	}

	if err := h.queries.SetUserActiveBadge(r.Context(), claims.UserID, body.BadgeKey); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"active_badge_key": body.BadgeKey})
}
