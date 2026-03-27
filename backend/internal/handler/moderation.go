package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

// ModerationHandler handles report submission and mod-only report management.
type ModerationHandler struct {
	queries  *db.Queries
	badgeSvc *service.BadgeService
}

// NewModerationHandler creates a ModerationHandler.
func NewModerationHandler(q *db.Queries, badgeSvc *service.BadgeService) *ModerationHandler {
	return &ModerationHandler{queries: q, badgeSvc: badgeSvc}
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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	if err := h.queries.UpdateReportStatus(r.Context(), id, status); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
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

	now := time.Now()
	store := service.StoreBadges()
	result := make([]storeBadge, 0, len(store))
	for _, b := range store {
		sb := storeBadge{
			BadgeKey:    b.Key,
			Title:       b.Title,
			Description: b.Description,
			Tier:        b.Tier,
			Cost:        b.Cost,
			Owned:       owned[b.Key],
		}
		if b.Stock != nil {
			sb.Stock = b.Stock
			sold, err := h.queries.CountBadgePurchases(ctx, b.Key)
			if err == nil {
				rem := int64(*b.Stock) - sold
				if rem < 0 {
					rem = 0
				}
				sb.RemainingStock = &rem
			}
		}
		if b.AvailableUntil != nil {
			s := b.AvailableUntil.Format("2006-01-02T15:04:05Z")
			sb.AvailableUntil = &s
			sb.Expired = now.After(*b.AvailableUntil)
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

	def, ok := service.BadgeKeysMap[body.BadgeKey]
	if !ok || !def.Purchasable {
		respondError(w, http.StatusBadRequest, "badge not available for purchase")
		return
	}

	// Check time-limited availability.
	if def.AvailableUntil != nil && time.Now().After(*def.AvailableUntil) {
		respondError(w, http.StatusGone, "this badge is no longer available")
		return
	}

	// Check stock.
	if def.Stock != nil {
		sold, err := h.queries.CountBadgePurchases(ctx, def.Key)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if sold >= int64(*def.Stock) {
			respondError(w, http.StatusConflict, "this badge is sold out")
			return
		}
	}

	// Fetch current balance.
	user, err := h.queries.GetUserByID(ctx, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if user.Balance < def.Cost {
		respondError(w, http.StatusPaymentRequired, "insufficient balance")
		return
	}

	// Check already owned.
	badges, err := h.queries.GetUserBadges(ctx, claims.UserID)
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

	// Deduct balance, then award. Both are idempotent so no transaction needed for badge award.
	if err := h.queries.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{
		Balance: -def.Cost,
		ID:      claims.UserID,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := h.queries.AwardBadgeIfNew(ctx, claims.UserID, def.Key); err != nil {
		// Refund on failure.
		_ = h.queries.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: def.Cost, ID: claims.UserID})
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "purchased"})

	// Non-blocking: check admiral rank milestones after the purchase.
	// Use context.Background() since the request context may be cancelled by the time the goroutine runs.
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
