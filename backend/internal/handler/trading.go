package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lhw/astroledger/internal/middleware"
	"github.com/lhw/astroledger/internal/service"
)

// TradingHandler handles buy/sell endpoints.
type TradingHandler struct {
	svc *service.TradingService
}

// NewTradingHandler creates a TradingHandler.
func NewTradingHandler(svc *service.TradingService) *TradingHandler {
	return &TradingHandler{svc: svc}
}

// Trade handles both buy and sell requests via POST /api/trades.
func (h *TradingHandler) Trade(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		MarketID  int64   `json:"market_id"`
		OutcomeID int64   `json:"outcome_id"` // FK into market_outcomes
		Action    string  `json:"action"`     // "buy" | "sell"
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

	result, err := h.svc.Execute(r.Context(), service.TradeInput{
		UserID:    claims.UserID,
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
		slog.Warn("trade failed", "user_id", claims.UserID, "market_id", body.MarketID, "action", body.Action, "shares", body.Shares, "err", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("trade executed", "user_id", claims.UserID, "market_id", body.MarketID, "action", body.Action, "shares", body.Shares, "cost", result.Cost, "trade_id", result.TradeID)
	respondJSON(w, http.StatusOK, result)
}
