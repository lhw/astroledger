package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

// MarketHandler handles market-related endpoints.
type MarketHandler struct {
	queries *db.Queries
	svc     *service.MarketService
}

// NewMarketHandler creates a MarketHandler.
func NewMarketHandler(q *db.Queries, svc *service.MarketService) *MarketHandler {
	return &MarketHandler{queries: q, svc: svc}
}

// List returns a paginated list of markets filtered by status and category.
func (h *MarketHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	category := r.URL.Query().Get("category")

	offset := int64(0)
	if v, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && v >= 0 {
		offset = v
	}

	count, err := h.queries.CountMarkets(r.Context(), db.CountMarketsParams{
		Status:   status,
		Column2:  category,
		Category: category,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	markets, err := h.queries.ListMarkets(r.Context(), db.ListMarketsParams{
		Status:   status,
		Column2:  category,
		Category: category,
		Limit:    20,
		Offset:   offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"total":   count,
		"markets": markets,
		"offset":  offset,
	})
}

// Get returns a single market by ID with current prices.
func (h *MarketHandler) Get(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	yesPrice := service.PriceCents(market.LiquidityParam, market.YesShares, market.NoShares)

	respondJSON(w, http.StatusOK, map[string]any{
		"market":    market,
		"yes_price": yesPrice,
		"no_price":  100 - yesPrice,
	})
}

// Create submits a new market for moderation review.
func (h *MarketHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Title              string `json:"title"`
		Description        string `json:"description"`
		Category           string `json:"category"`
		ResolutionCriteria string `json:"resolution_criteria"`
		Deadline           string `json:"deadline"` // RFC3339
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	deadline, err := time.Parse(time.RFC3339, body.Deadline)
	if err != nil {
		respondError(w, http.StatusBadRequest, "deadline must be RFC3339 format")
		return
	}

	market, err := h.svc.CreateMarket(r.Context(), service.CreateMarketInput{
		Title:              body.Title,
		Description:        body.Description,
		Category:           body.Category,
		ResolutionCriteria: body.ResolutionCriteria,
		Deadline:           deadline,
		CreatedBy:          claims.UserID,
	})
	if errors.Is(err, service.ErrRejected) {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, market)
}

// GetPriceHistory returns the price history of a market for charting.
func (h *MarketHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	history, err := h.queries.GetMarketPriceHistory(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// GetTrades returns recent trades for a market.
func (h *MarketHandler) GetTrades(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	offset := int64(0)
	if v, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && v >= 0 {
		offset = v
	}

	trades, err := h.queries.GetMarketTrades(r.Context(), db.GetMarketTradesParams{
		MarketID: id,
		Limit:    50,
		Offset:   offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respondJSON(w, http.StatusOK, trades)
}

// --- Moderation endpoints ---

// ListPending returns markets awaiting moderation review.
func (h *MarketHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	markets, err := h.queries.ListPendingMarkets(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, markets)
}

// Approve approves a pending market.
func (h *MarketHandler) Approve(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	if err := h.svc.ApproveMarket(r.Context(), id, claims.UserID); errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	} else if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// Reject rejects a pending market.
func (h *MarketHandler) Reject(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	if err := h.svc.RejectMarket(r.Context(), id, claims.UserID); errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	} else if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// Resolve resolves an active market as yes or no.
func (h *MarketHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	var body struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.svc.ResolveMarket(r.Context(), service.ResolveInput{
		MarketID:   id,
		Resolution: body.Resolution,
		ModID:      claims.UserID,
	}); errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	} else if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}
