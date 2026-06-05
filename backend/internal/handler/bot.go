package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
	"github.com/lhw/astroledger/internal/service"
)

// BotHandler manages API tokens and bot-scoped read/trade/create endpoints.
type BotHandler struct {
	queries   *db.Queries
	trading   *service.TradingService
	marketSvc *service.MarketService
}

// NewBotHandler creates a BotHandler.
func NewBotHandler(q *db.Queries, trading *service.TradingService, marketSvc *service.MarketService) *BotHandler {
	return &BotHandler{queries: q, trading: trading, marketSvc: marketSvc}
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

	// Reject bot requests if the token owner is banned.
	if banStatus, banErr := h.queries.GetUserBanStatus(r.Context(), row.UserID); banErr == nil && banStatus.IsBanned == 1 {
		return nil, fmt.Errorf("account suspended")
	}

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
		Name             string `json:"name"`
		CanRead          *bool  `json:"can_read"`
		CanTrade         bool   `json:"can_trade"`
		CanCreateMarkets bool   `json:"can_create_markets"`
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
	// Create markets scope requires read scope.
	if body.CanCreateMarkets && !canRead {
		canRead = true
	}

	rawToken, prefix, err := generateBotToken()
	if err != nil {
		slog.Error("bot: generateBotToken", "user_id", claims.UserID, "err", err)
		respondError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	row, err := h.queries.CreateAPIToken(r.Context(), db.CreateAPITokenParams{
		UserID:           claims.UserID,
		Name:             name,
		TokenHash:        hashToken(rawToken),
		TokenPrefix:      prefix,
		CanRead:          boolToInt64(canRead),
		CanTrade:         boolToInt64(body.CanTrade),
		CanCreateMarkets: boolToInt64(body.CanCreateMarkets),
	})
	if err != nil {
		slog.Error("bot: CreateAPIToken", "user_id", claims.UserID, "name", name, "err", err)
		respondError(w, http.StatusInternalServerError, "failed to save token")
		return
	}

	slog.Info("bot token created", "user_id", claims.UserID, "token_id", row.ID, "name", name, "can_read", canRead, "can_trade", body.CanTrade, "can_create_markets", body.CanCreateMarkets)

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":                 row.ID,
		"name":               row.Name,
		"token_prefix":       row.TokenPrefix,
		"can_read":           row.CanRead == 1,
		"can_trade":          row.CanTrade == 1,
		"can_create_markets": row.CanCreateMarkets == 1,
		"created_at":         row.CreatedAt,
		"token":              rawToken,
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
		slog.Error("bot: ListUserAPITokens", "user_id", claims.UserID, "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                 row.ID,
			"name":               row.Name,
			"token_prefix":       row.TokenPrefix,
			"can_read":           row.CanRead == 1,
			"can_trade":          row.CanTrade == 1,
			"can_create_markets": row.CanCreateMarkets == 1,
			"created_at":         row.CreatedAt,
			"last_used_at":       row.LastUsedAt,
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
		slog.Error("bot: RevokeUserAPIToken", "user_id", claims.UserID, "token_id", tokenID, "err", err)
		respondError(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}

	slog.Info("bot token revoked", "user_id", claims.UserID, "token_id", tokenID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// Me returns basic user account data when authenticated via bot token (read scope).
func (h *BotHandler) Me(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "me", "err", err)
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
		slog.Error("bot: Me: GetUserByID", "user_id", tokenRow.UserID, "err", err)
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
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "trade", "err", err)
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
		slog.Warn("bot trade failed", "user_id", tokenRow.UserID, "token_id", tokenRow.ID, "market_id", body.MarketID, "action", body.Action, "shares", body.Shares, "err", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("bot trade executed", "user_id", tokenRow.UserID, "token_id", tokenRow.ID, "market_id", body.MarketID, "action", body.Action, "shares", body.Shares, "cost", result.Cost, "trade_id", result.TradeID)

	respondJSON(w, http.StatusOK, result)
}

// ListMarkets returns active markets (requires read scope).
func (h *BotHandler) ListMarkets(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "list_markets", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanRead != 1 {
		respondError(w, http.StatusForbidden, "token missing read scope")
		return
	}

	category := r.URL.Query().Get("category")
	offset := int64(0)
	if v, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && v >= 0 {
		offset = v
	}

	count, err := h.queries.CountMarkets(r.Context(), db.CountMarketsParams{
		Status:   "active",
		Column2:  category,
		Category: category,
	})
	if err != nil {
		slog.Error("bot: ListMarkets: CountMarkets", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	markets, err := h.queries.ListMarkets(r.Context(), db.ListMarketsParams{
		Status:   "active",
		Column2:  category,
		Category: category,
		Limit:    50,
		Offset:   offset,
	})
	if err != nil {
		slog.Error("bot: ListMarkets: ListMarkets", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"total":   count,
		"markets": markets,
		"offset":  offset,
	})
}

// GetMarket returns a single market by ID (requires read scope).
func (h *BotHandler) GetMarket(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "get_market", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanRead != 1 {
		respondError(w, http.StatusForbidden, "token missing read scope")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	market, err := h.queries.GetMarketByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	}
	if err != nil {
		slog.Error("bot: GetMarket: GetMarketByID", "market_id", id, "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, market)
}

// GetUserTrades returns trades for the authenticated user (requires read scope).
func (h *BotHandler) GetUserTrades(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "get_trades", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanRead != 1 {
		respondError(w, http.StatusForbidden, "token missing read scope")
		return
	}

	offset := int64(0)
	if v, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && v >= 0 {
		offset = v
	}

	trades, err := h.queries.GetUserTrades(r.Context(), db.GetUserTradesParams{
		UserID: tokenRow.UserID,
		Limit:  50,
		Offset: offset,
	})
	if err != nil {
		slog.Error("bot: GetUserTrades", "user_id", tokenRow.UserID, "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, trades)
}

// GetUserPositions returns positions for the authenticated user (requires read scope).
func (h *BotHandler) GetUserPositions(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "get_positions", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanRead != 1 {
		respondError(w, http.StatusForbidden, "token missing read scope")
		return
	}

	positions, err := h.queries.GetUserPositions(r.Context(), tokenRow.UserID)
	if err != nil {
		slog.Error("bot: GetUserPositions", "user_id", tokenRow.UserID, "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, positions)
}

// CreateMarket creates a new market using a bot API token (requires can_create_markets scope).
// Only moderators and admins can create markets via the bot API.
func (h *BotHandler) CreateMarket(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "create_market", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if tokenRow.CanCreateMarkets != 1 {
		respondError(w, http.StatusForbidden, "token missing create_markets scope")
		return
	}

	// Only moderators and admins can create markets via bot API.
	user, err := h.queries.GetUserByID(r.Context(), tokenRow.UserID)
	if err != nil {
		slog.Error("bot: CreateMarket: GetUserByID", "user_id", tokenRow.UserID, "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if user.IsModerator != 1 && user.IsAdmin != 1 {
		respondError(w, http.StatusForbidden, "only moderators and admins can create markets via bot API")
		return
	}

	var body struct {
		Title              string   `json:"title"`
		Description        string   `json:"description"`
		Category           string   `json:"category"`
		ResolutionCriteria string   `json:"resolution_criteria"`
		Deadline           string   `json:"deadline"` // patch version (e.g. "4.9.0") or RFC3339 date
		Outcomes           []string `json:"outcomes"` // optional; defaults to ["YES", "NO"]
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate required fields.
	if strings.TrimSpace(body.Title) == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if strings.TrimSpace(body.Description) == "" {
		respondError(w, http.StatusBadRequest, "description is required")
		return
	}
	if strings.TrimSpace(body.Category) == "" {
		respondError(w, http.StatusBadRequest, "category is required")
		return
	}
	if strings.TrimSpace(body.ResolutionCriteria) == "" {
		respondError(w, http.StatusBadRequest, "resolution_criteria is required")
		return
	}
	if strings.TrimSpace(body.Deadline) == "" {
		respondError(w, http.StatusBadRequest, "deadline is required")
		return
	}

	// Parse deadline: try patch version first, then RFC3339.
	deadlineStr := strings.TrimSpace(body.Deadline)
	var deadline time.Time
	isPatchVersion := false

	if t, err := time.Parse(time.RFC3339, deadlineStr); err == nil {
		deadline = t
	} else if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, deadlineStr); matched {
		// Patch version — set deadline 2 years out; resolution criteria records the target patch.
		isPatchVersion = true
		deadline = time.Now().Add(2 * 365 * 24 * time.Hour)
	} else {
		respondError(w, http.StatusBadRequest, "deadline must be an RFC3339 date or patch version (e.g. '4.9.0')")
		return
	}

	// Build resolution criteria with patch prefix if needed.
	resolutionCriteria := strings.TrimSpace(body.ResolutionCriteria)
	if isPatchVersion {
		resolutionCriteria = fmt.Sprintf("Resolves when patch %s ships. %s", deadlineStr, resolutionCriteria)
	}

	// Default to binary YES/NO.
	outcomes := body.Outcomes
	if len(outcomes) == 0 {
		outcomes = []string{"YES", "NO"}
	}

	// Create the market using the MarketService.
	market, err := h.marketSvc.CreateMarket(r.Context(), service.CreateMarketInput{
		Title:              strings.TrimSpace(body.Title),
		Description:        strings.TrimSpace(body.Description),
		Category:           strings.TrimSpace(body.Category),
		ResolutionCriteria: resolutionCriteria,
		Deadline:           deadline,
		CreatedBy:          tokenRow.UserID,
		Outcomes:           outcomes,
	})
	if err != nil {
		slog.Warn("bot create market failed", "user_id", tokenRow.UserID, "token_id", tokenRow.ID, "err", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("bot market created", "user_id", tokenRow.UserID, "token_id", tokenRow.ID, "market_id", market.ID, "title", market.Title)

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":     market.ID,
		"title":  market.Title,
		"status": market.Status,
	})
}

// ListPendingMarkets returns markets awaiting moderation review (requires mod/admin).
func (h *BotHandler) ListPendingMarkets(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "list_pending", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if h.requireMod(tokenRow.UserID) != nil {
		respondError(w, http.StatusForbidden, "moderator or admin role required")
		return
	}

	markets, err := h.queries.ListPendingMarkets(r.Context())
	if err != nil {
		slog.Error("bot: ListPendingMarkets", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, markets)
}

// ApproveMarket approves a pending market (requires mod/admin).
func (h *BotHandler) ApproveMarket(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "approve_market", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if h.requireMod(tokenRow.UserID) != nil {
		respondError(w, http.StatusForbidden, "moderator or admin role required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	if err := h.marketSvc.ApproveMarket(r.Context(), id, tokenRow.UserID); err != nil {
		slog.Warn("bot approve failed", "user_id", tokenRow.UserID, "market_id", id, "err", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// RejectMarket rejects a pending market (requires mod/admin).
func (h *BotHandler) RejectMarket(w http.ResponseWriter, r *http.Request) {
	tokenRow, err := h.authenticateToken(r)
	if err != nil {
		slog.Warn("bot auth failed", "remote_addr", r.RemoteAddr, "endpoint", "reject_market", "err", err)
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if h.requireMod(tokenRow.UserID) != nil {
		respondError(w, http.StatusForbidden, "moderator or admin role required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	if err := h.marketSvc.RejectMarket(r.Context(), id, tokenRow.UserID); err != nil {
		slog.Warn("bot reject failed", "user_id", tokenRow.UserID, "market_id", id, "err", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// requireMod checks if the user has moderator or admin role. Returns error if not.
func (h *BotHandler) requireMod(userID int64) error {
	user, err := h.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("database error")
	}
	if user.IsModerator != 1 && user.IsAdmin != 1 {
		return fmt.Errorf("moderator or admin role required")
	}
	return nil
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
