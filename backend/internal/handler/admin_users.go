package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
)

// SearchUsers searches users by display name or RSI handle for the admin UI.
// GET /api/admin/users/search?q=...
func (h *AdminHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		respondJSON(w, http.StatusOK, []db.SearchUsersRow{})
		return
	}

	likePat := "%" + q + "%"
	results, err := h.queries.SearchUsers(r.Context(), db.SearchUsersParams{
		DisplayName: likePat,
		RsiHandle:   &likePat,
	})
	if err != nil {
		slog.Error("admin search users", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, results)
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

// AdjustUserBalance adjusts a user's bUEC balance.
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

	target, err := h.queries.GetUserByID(r.Context(), targetID)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		slog.Error("AdjustUserBalance: GetUserByID", "err", err, "target_id", targetID)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if target.Balance+body.Amount < 0 {
		respondError(w, http.StatusUnprocessableEntity, "balance cannot go below 0")
		return
	}

	newBalance, err := h.queries.AdminAdjustUserBalance(r.Context(), db.AdminAdjustUserBalanceParams{
		Balance: body.Amount,
		ID:      targetID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusUnprocessableEntity, "balance cannot go below 0")
		return
	}
	if err != nil {
		slog.Error("AdjustUserBalance: AdminAdjustUserBalance", "err", err, "target_id", targetID)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

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

// BanUser bans or unbans a user.
// POST /api/admin/users/{id}/ban   body: {"banned": true|false}
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	targetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var body struct {
		Banned bool `json:"banned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	banned := int64(0)
	if body.Banned {
		banned = 1
	}
	if err := h.queries.SetUserBanned(r.Context(), targetID, banned); err != nil {
		slog.Error("BanUser: SetUserBanned", "err", err, "target_id", targetID)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"banned": body.Banned})
}

// ShadowBanUser shadow-bans or un-shadow-bans a user.
// POST /api/admin/users/{id}/shadow-ban   body: {"shadow_banned": true|false}
func (h *AdminHandler) ShadowBanUser(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	targetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var body struct {
		ShadowBanned bool `json:"shadow_banned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	sb := int64(0)
	if body.ShadowBanned {
		sb = 1
	}
	if err := h.queries.SetUserShadowBanned(r.Context(), targetID, sb); err != nil {
		slog.Error("ShadowBanUser: SetUserShadowBanned", "err", err, "target_id", targetID)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"shadow_banned": body.ShadowBanned})
}
