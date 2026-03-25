package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	// Enrich with static title/description.
	type richBadge struct {
		BadgeKey    string `json:"badge_key"`
		AwardedAt   string `json:"awarded_at"`
		Title       string `json:"title"`
		Description string `json:"description"`
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
		})
	}
	respondJSON(w, http.StatusOK, result)
}
