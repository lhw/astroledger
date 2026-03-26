package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	queries    *db.Queries
	creditsSvc *service.CreditsService
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(queries *db.Queries, creditsSvc *service.CreditsService) *AdminHandler {
	return &AdminHandler{queries: queries, creditsSvc: creditsSvc}
}

// requireAdmin checks that the caller is an authenticated admin.
// Returns the user on success, or writes an error response and returns nil.
func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) *db.User {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	user, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return nil
	}
	if user.IsAdmin != 1 {
		respondError(w, http.StatusForbidden, "forbidden")
		return nil
	}
	return &user
}

// TriggerWeeklyPayout manually triggers the weekly credit payout.
// POST /api/admin/weekly-payout
func (h *AdminHandler) TriggerWeeklyPayout(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}

	count, alreadyRan, weekKey, err := h.creditsSvc.TriggerWeeklyPayout(r.Context())
	if err != nil {
		slog.Error("admin weekly payout failed", "err", err)
		respondError(w, http.StatusInternalServerError, "payout failed")
		return
	}
	if alreadyRan {
		respondJSON(w, http.StatusConflict, map[string]any{
			"error": "Payout already ran this week",
			"week":  weekKey,
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"users_paid":       count,
		"credits_per_user": db.WeeklyPayoutAmount,
		"message":          "Payout complete",
	})
}

// AdjustUserBalance adjusts a user's ScollyBucks™ balance.
// POST /api/admin/users/:id/balance
func (h *AdminHandler) AdjustUserBalance(w http.ResponseWriter, r *http.Request) {
	admin := h.requireAdmin(w, r)
	if admin == nil {
		return
	}

	targetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Amount == 0 {
		respondError(w, http.StatusBadRequest, "amount must be non-zero")
		return
	}
	if body.Reason == "" {
		respondError(w, http.StatusBadRequest, "reason required")
		return
	}
	if len(body.Reason) > 200 {
		respondError(w, http.StatusBadRequest, "reason too long (max 200 chars)")
		return
	}

	// Verify target user exists and new balance won't go negative.
	target, err := h.queries.GetUserByID(r.Context(), targetID)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if target.Balance+body.Amount < 0 {
		respondError(w, http.StatusUnprocessableEntity, "balance cannot go below 0")
		return
	}

	// Apply adjustment (returns new balance atomically).
	newBalance, err := h.queries.AdminAdjustUserBalance(r.Context(), db.AdminAdjustUserBalanceParams{
		Balance: body.Amount,
		ID:      targetID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		// Race: balance went negative between check and update.
		respondError(w, http.StatusUnprocessableEntity, "balance cannot go below 0")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Log the adjustment — non-fatal if it fails.
	if logErr := h.queries.LogAdminAdjustment(r.Context(), db.LogAdminAdjustmentParams{
		AdminID: admin.ID,
		UserID:  targetID,
		Amount:  body.Amount,
		Reason:  body.Reason,
	}); logErr != nil {
		slog.Error("failed to log admin balance adjustment", "err", logErr, "admin_id", admin.ID, "user_id", targetID)
	}

	respondJSON(w, http.StatusOK, map[string]any{"new_balance": newBalance})
}
