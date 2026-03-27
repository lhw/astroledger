package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
)

// UserHandler handles user-related endpoints.
type UserHandler struct {
	queries *db.Queries
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(q *db.Queries) *UserHandler {
	return &UserHandler{queries: q}
}

// Me returns the currently authenticated user's profile.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	activeBadge, _ := h.queries.GetUserActiveBadge(r.Context(), claims.UserID)

	resp := userResponse(user)
	resp["active_badge_key"] = activeBadge
	respondJSON(w, http.StatusOK, resp)
}

// GetUser returns a user's public profile by ID.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Public profile omits email and sensitive fields.
	respondJSON(w, http.StatusOK, map[string]any{
		"id":           user.ID,
		"display_name": user.DisplayName,
		"balance":      user.Balance,
		"created_at":   user.CreatedAt,
	})
}

// Leaderboard returns the top users ranked by balance + portfolio value.
func (h *UserHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := int64(25)
	if limitStr != "" {
		if v, err := strconv.ParseInt(limitStr, 10, 64); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	rows, err := h.queries.GetLeaderboard(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, rows)
}

// GetUserPositions returns the authenticated user's open market positions,
// enriched with cost basis and resolved market info for P&L display.
func (h *UserHandler) GetUserPositions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	positions, err := h.queries.GetUserPositionsEnriched(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, positions)
}

// GetUserTrades returns the authenticated user's trade history.
func (h *UserHandler) GetUserTrades(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := int64(0)
	if v, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && v >= 0 {
		offset = v
	}

	trades, err := h.queries.GetUserTrades(r.Context(), db.GetUserTradesParams{
		UserID: claims.UserID,
		Limit:  50,
		Offset: offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, trades)
}

// userResponse shapes a user row into an API response (includes private fields for /me).
func userResponse(u db.GetUserByIDRow) map[string]any {
	return map[string]any{
		"id":                 u.ID,
		"display_name":       u.DisplayName,
		"email":              u.Email,
		"balance":            u.Balance,
		"is_moderator":       u.IsModerator == 1,
		"is_admin":           u.IsAdmin == 1,
		"is_rsi_verified":    u.IsRsiVerified == 1,
		"rsi_handle":         u.RsiHandle,
		"rsi_verified_at":    u.RsiVerifiedAt,
		"rsi_enlisted":       u.RsiEnlisted,
		"rsi_citizen_record": u.RsiCitizenRecord,
		"avatar_url":         u.AvatarUrl,
		"created_at":         u.CreatedAt,
		"last_login_at":      u.LastLoginAt,
	}
}
