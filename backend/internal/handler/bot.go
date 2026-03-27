package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

// BotHandler manages API tokens and bot-scoped read/trade endpoints.
type BotHandler struct {
	queries *db.Queries
	trading *service.TradingService
}

// NewBotHandler creates a BotHandler.
func NewBotHandler(q *db.Queries, trading *service.TradingService) *BotHandler {
	return &BotHandler{queries: q, trading: trading}
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func generateBotToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	full := "smt_" + encoded
	if len(full) < 12 {
		return "", "", fmt.Errorf("generated token too short")
	}
	prefix := full[:12]
	return full, prefix, nil
}

func parseBearerToken(r *http.Request) (string, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return "", fmt.Errorf("missing Authorization header")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("Authorization must be Bearer <token>")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}

func (h *BotHandler) authenticateToken(r *http.Request) (*db.GetAPITokenByHashRow, error) {
	token, err := parseBearerToken(r)
	if err != nil {
		return nil, err
	}

	row, err := h.queries.GetAPITokenByHash(r.Context(), hashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("token lookup: %w", err)
	}

	_ = h.queries.TouchAPITokenLastUsed(r.Context(), row.ID)
	return &row, nil
}

// CreateToken creates a new bot API token for the authenticated user.
func (h *BotHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Name     string `json:"name"`
		CanRead  *bool  `json:"can_read"`
		CanTrade bool   `json:"can_trade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	name := strings.TrimSpace(body.Name)
	if len(name) < 3 || len(name) > 64 {
		respondError(w, http.StatusBadRequest, "name must be 3-64 characters")
		return
	}

	canRead := true
	if body.CanRead != nil {
		canRead = *body.CanRead
	}
	if !canRead && body.CanTrade {
		canRead = true
	}

	rawToken, prefix, err := generateBotToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	row, err := h.queries.CreateAPIToken(r.Context(), db.CreateAPITokenParams{
		UserID:      claims.UserID,
		Name:        name,
		TokenHash:   hashToken(rawToken),
		TokenPrefix: prefix,
		CanRead:     boolToInt64(canRead),
		CanTrade:    boolToInt64(body.CanTrade),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save token")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":           row.ID,
		"name":         row.Name,
		"token_prefix": row.TokenPrefix,
		"can_read":     row.CanRead == 1,
		"can_trade":    row.CanTrade == 1,
		"created_at":   row.CreatedAt,
		"token":        rawToken,
	})
}

// ListTokens lists active bot tokens for the authenticated user.
func (h *BotHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rows, err := h.queries.ListUserAPITokens(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":           row.ID,
			"name":         row.Name,
			"token_prefix": row.TokenPrefix,
			"can_read":     row.CanRead == 1,
			"can_trade":    row.CanTrade == 1,
			"created_at":   row.CreatedAt,
			"last_used_at": row.LastUsedAt,
		})
	}

	respondJSON(w, http.StatusOK, out)
}

// RevokeToken revokes one of the authenticated user's bot tokens.
func (h *BotHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tokenID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tokenID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid token id")
		return
	}

	_, err = h.queries.RevokeUserAPIToken(r.Context(), db.RevokeUserAPITokenParams{
		ID:     tokenID,
		UserID: claims.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "token not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// Me returns basic user account data when authenticated via bot token (read scope).
func (h *BotHandler) Me(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanRead != 1 {
		respondError(w, http.StatusForbidden, "token missing read scope")
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), tokenRow.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusUnauthorized, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"id":           user.ID,
		"display_name": user.DisplayName,
		"balance":      user.Balance,
	})
}

// Trade executes a buy/sell order using a bot API token (trade scope).
func (h *BotHandler) Trade(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanTrade != 1 {
		respondError(w, http.StatusForbidden, "token missing trade scope")
		return
	}

	var body struct {
		MarketID  int64   `json:"market_id"`
		OutcomeID int64   `json:"outcome_id"`
		Action    string  `json:"action"`
		Shares    float64 `json:"shares"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if body.MarketID <= 0 {
		respondError(w, http.StatusBadRequest, "market_id required")
		return
	}
	if body.OutcomeID <= 0 {
		respondError(w, http.StatusBadRequest, "outcome_id required")
		return
	}
	if body.Shares <= 0 {
		respondError(w, http.StatusBadRequest, "shares must be positive")
		return
	}
	if body.Shares > 10000 {
		respondError(w, http.StatusBadRequest, "shares exceeds maximum per trade (10000)")
		return
	}

	result, err := h.trading.Execute(r.Context(), service.TradeInput{
		UserID:    tokenRow.UserID,
		MarketID:  body.MarketID,
		OutcomeID: body.OutcomeID,
		Action:    body.Action,
		Shares:    body.Shares,
	})
	if errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
