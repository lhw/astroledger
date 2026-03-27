package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// outcomeResp is the JSON shape for a market outcome with its current LMSR price.
type outcomeResp struct {
	ID     int64   `json:"id"`
	Label  string  `json:"label"`
	Price  int64   `json:"price"`
	Shares float64 `json:"shares"`
}

// buildOutcomeResps fetches outcomes for a market and computes LMSR prices.
func (h *MarketHandler) buildOutcomeResps(r *http.Request, marketID int64, liqParam float64) []outcomeResp {
	outcomes, err := h.queries.GetOutcomesByMarketID(r.Context(), marketID)
	if err != nil {
		return []outcomeResp{}
	}
	allShares := make([]float64, len(outcomes))
	for i, o := range outcomes {
		allShares[i] = o.Shares
	}
	resps := make([]outcomeResp, len(outcomes))
	for i, o := range outcomes {
		resps[i] = outcomeResp{
			ID:     o.ID,
			Label:  o.Label,
			Price:  service.OutcomeProbCents(liqParam, allShares, i),
			Shares: o.Shares,
		}
	}
	return resps
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

	type listMarketWithOutcomes struct {
		db.ListMarketsRow
		Outcomes []outcomeResp `json:"outcomes"`
	}
	result := make([]listMarketWithOutcomes, len(markets))
	for i, m := range markets {
		result[i] = listMarketWithOutcomes{
			ListMarketsRow: m,
			Outcomes:       h.buildOutcomeResps(r, m.ID, m.LiquidityParam),
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"total":   count,
		"markets": result,
		"offset":  offset,
	})
}

// Get returns a single market by ID with current prices and (if authenticated) the caller's position.
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

	outcomeResps := h.buildOutcomeResps(r, id, market.LiquidityParam)

	// Fetch trade statistics (volume, trader count, trade count).
	stats, _ := h.queries.GetMarketStats(r.Context(), id)

	type marketWithOutcomes struct {
		db.GetMarketByIDRow
		Outcomes []outcomeResp `json:"outcomes"`
	}

	resp := map[string]any{
		"market":       marketWithOutcomes{GetMarketByIDRow: market, Outcomes: outcomeResps},
		"total_volume": stats.TotalVolume,
		"trader_count": stats.TraderCount,
		"trade_count":  stats.TradeCount,
	}

	// Include the caller's positions (one per outcome) when authenticated.
	if claims := middleware.GetClaims(r); claims != nil {
		type posResp struct {
			OutcomeID int64   `json:"outcome_id"`
			Label     string  `json:"label"`
			Shares    float64 `json:"shares"`
		}
		var myPositions []posResp
		for _, o := range outcomeResps {
			pPos, _ := h.queries.GetUserPosition(r.Context(), db.GetUserPositionParams{
				UserID:    claims.UserID,
				MarketID:  id,
				OutcomeID: o.ID,
			})
			if pPos.Shares > 0 {
				myPositions = append(myPositions, posResp{
					OutcomeID: o.ID,
					Label:     o.Label,
					Shares:    pPos.Shares,
				})
			}
		}
		if myPositions == nil {
			myPositions = []posResp{}
		}
		resp["my_positions"] = myPositions
	}

	respondJSON(w, http.StatusOK, resp)
}

// Create submits a new market for moderation review.
func (h *MarketHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Title               string   `json:"title"`
		Description         string   `json:"description"`
		Category            string   `json:"category"`
		ResolutionCriteria  string   `json:"resolution_criteria"`
		Deadline            string   `json:"deadline"`             // RFC3339
		ResolutionType      string   `json:"resolution_type"`      // binary|date|numeric
		ResolutionThreshold *string  `json:"resolution_threshold"` // ISO date or numeric value
		Outcomes            []string `json:"outcomes"`             // optional; defaults to [YES, NO]
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
		Title:               body.Title,
		Description:         body.Description,
		Category:            body.Category,
		ResolutionCriteria:  body.ResolutionCriteria,
		Deadline:            deadline,
		CreatedBy:           claims.UserID,
		ResolutionType:      body.ResolutionType,
		ResolutionThreshold: body.ResolutionThreshold,
		Outcomes:            body.Outcomes,
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
	rules, err := h.queries.GetEnabledAutofilterRules(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	markets, err := h.queries.ListPendingMarkets(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	type pendingMarketWithMatches struct {
		db.ListPendingMarketsRow
		AutoFilterMatches []string `json:"auto_filter_matches"`
	}

	result := make([]pendingMarketWithMatches, len(markets))
	for i, m := range markets {
		result[i] = pendingMarketWithMatches{
			ListPendingMarketsRow: m,
			AutoFilterMatches:     autoFilterMatches(rules, m.Title+" "+m.Description),
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// ListDeadlinePassed returns active-but-expired markets awaiting moderator resolution.
func (h *MarketHandler) ListDeadlinePassed(w http.ResponseWriter, r *http.Request) {
	markets, err := h.queries.ListDeadlinePassedMarkets(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	type deadlinePassedWithOutcomes struct {
		db.ListDeadlinePassedMarketsRow
		Outcomes []outcomeResp `json:"outcomes"`
	}

	result := make([]deadlinePassedWithOutcomes, len(markets))
	for i, m := range markets {
		result[i] = deadlinePassedWithOutcomes{
			ListDeadlinePassedMarketsRow: m,
			Outcomes:                     h.buildOutcomeResps(r, m.ID, m.LiquidityParam),
		}
	}

	respondJSON(w, http.StatusOK, result)
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

// Resolve resolves an active (or resolution_requested) market as yes or no.
func (h *MarketHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	var body struct {
		WinningOutcomeID int64   `json:"winning_outcome_id"`
		EvidenceLink     *string `json:"evidence_link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.svc.ResolveMarket(r.Context(), service.ResolveInput{
		MarketID:         id,
		WinningOutcomeID: body.WinningOutcomeID,
		ModID:            claims.UserID,
		EvidenceLink:     body.EvidenceLink,
	}); errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
		return
	} else if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// RequestResolution lets the market creator ask the mod team to resolve their market.
func (h *MarketHandler) RequestResolution(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	var body struct {
		Link string `json:"link"`
		Note string `json:"note"`
	}
	// Body is optional — ignore decode errors (no body is fine).
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.svc.RequestResolution(r.Context(), service.RequestResolutionInput{
		MarketID: id,
		CallerID: claims.UserID,
		Link:     body.Link,
		Note:     body.Note,
	}); errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "market not found")
	} else if errors.Is(err, service.ErrForbidden) {
		respondError(w, http.StatusForbidden, "you must hold shares to request resolution")
	} else if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
	} else {
		respondJSON(w, http.StatusOK, map[string]string{"status": "resolution_requested"})
	}
}

// ListResolutionRequested returns markets where the creator has requested resolution (mod only).
func (h *MarketHandler) ListResolutionRequested(w http.ResponseWriter, r *http.Request) {
	markets, err := h.queries.ListResolutionRequestedMarkets(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	type resolutionRequestWithOutcomes struct {
		db.ResolutionRequestRow
		Outcomes []outcomeResp `json:"outcomes"`
	}
	result := make([]resolutionRequestWithOutcomes, len(markets))
	for i, m := range markets {
		result[i] = resolutionRequestWithOutcomes{
			ResolutionRequestRow: m,
			Outcomes:             h.buildOutcomeResps(r, m.ID, m.LiquidityParam),
		}
	}
	respondJSON(w, http.StatusOK, result)
}

// DenyResolution rejects a resolution request and sets the market back to active.
func (h *MarketHandler) DenyResolution(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}
	market, err := h.queries.GetMarketByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "market not found")
		return
	}
	if market.Status != "resolution_requested" {
		respondError(w, http.StatusBadRequest, "market is not awaiting resolution")
		return
	}
	if err := h.queries.UpdateMarketStatus(r.Context(), db.UpdateMarketStatusParams{Status: "active", ID: id}); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	// Clean up the request details so a fresh request can be filed later.
	_ = h.queries.DeleteResolutionRequestDetails(r.Context(), id)

	note := "resolution request denied"
	if err := h.queries.LogModAudit(r.Context(), db.LogModAuditParams{
		ActionType: "deny_resolution",
		TargetType: "market",
		TargetID:   id,
		ModUserID:  claims.UserID,
		Note:       &note,
	}); err != nil {
		slog.Warn("mod audit log failed", "action", "deny_resolution", "market_id", id, "mod_id", claims.UserID, "err", err)
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func autoFilterMatches(rules []db.GetEnabledAutofilterRulesRow, text string) []string {
	lower := strings.ToLower(text)
	trimmed := strings.TrimSpace(text)
	matches := make([]string, 0)

	for _, rule := range rules {
		switch rule.RuleType {
		case "keyword":
			if strings.Contains(lower, strings.ToLower(rule.Value)) {
				matches = append(matches, fmt.Sprintf("keyword: %s", rule.Value))
			}
		case "regex":
			re, err := regexp.Compile(rule.Value)
			if err != nil {
				continue
			}
			if re.MatchString(lower) {
				matches = append(matches, fmt.Sprintf("regex: %s", rule.Value))
			}
		case "min_length":
			var minLen int
			if _, err := fmt.Sscanf(rule.Value, "%d", &minLen); err == nil {
				if utf8.RuneCountInString(trimmed) < minLen {
					matches = append(matches, fmt.Sprintf("min_length: %d", minLen))
				}
			}
		}
	}

	return matches
}
