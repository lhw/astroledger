package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
	"github.com/lhw/astroledger/internal/service"
)

// analyticsTTL is how long aggregated GoatCounter responses are cached.
const analyticsTTL = 5 * time.Minute

type analyticsCacheEntry struct {
	data     *analyticsResponse
	cachedAt time.Time
}

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	queries    *db.Queries
	creditsSvc *service.CreditsService
	gcURL      string // GoatCounter internal base URL
	gcAPIKey   string // GoatCounter API Bearer token
	cacheMu    sync.RWMutex
	cache      map[string]*analyticsCacheEntry // keyed by period ("7d", "30d")
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(queries *db.Queries, creditsSvc *service.CreditsService, gcURL, gcAPIKey string) *AdminHandler {
	return &AdminHandler{
		queries:    queries,
		creditsSvc: creditsSvc,
		gcURL:      gcURL,
		gcAPIKey:   gcAPIKey,
		cache:      make(map[string]*analyticsCacheEntry),
	}
}

// ── GoatCounter API types ─────────────────────────────────────────────────

type gcHitsResponse struct {
	Hits []gcHit `json:"hits"`
}

type gcHit struct {
	Path        string      `json:"path"`
	Title       string      `json:"title"`
	Count       int         `json:"count"`
	CountUnique int         `json:"count_unique"`
	Stats       []gcDayStat `json:"stats"`
}

type gcDayStat struct {
	Day   string `json:"day"`
	Daily int    `json:"daily"`
}

type gcRefsResponse struct {
	Refs []gcRef `json:"refs"`
}

type gcRef struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// gcStatItem is the common element in GoatCounter list-stat responses
// (browsers, systems, locations, languages, sizes).
type gcStatItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type gcStatListResponse struct {
	Stats []gcStatItem `json:"stats"`
}

// ── Analytics response types returned to the frontend ────────────────────

type analyticsResponse struct {
	Configured  bool               `json:"configured"`
	Period      string             `json:"period"`
	TotalViews  int                `json:"total_views"`
	TotalUnique int                `json:"total_unique"`
	Daily       []analyticsDayStat `json:"daily"`
	TopPages    []analyticsPage    `json:"top_pages"`
	TopRefs     []analyticsRef     `json:"top_refs"`
	Browsers    []analyticsStat    `json:"browsers"`
	Systems     []analyticsStat    `json:"systems"`
	Locations   []analyticsStat    `json:"locations"`
	Languages   []analyticsStat    `json:"languages"`
}

type analyticsStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type analyticsDayStat struct {
	Date   string `json:"date"`
	Views  int    `json:"views"`
	Unique int    `json:"unique"`
}

type analyticsPage struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Views  int    `json:"views"`
	Unique int    `json:"unique"`
}

type analyticsRef struct {
	Name  string `json:"name"`
	Views int    `json:"views"`
}

// ── Analytics handler ─────────────────────────────────────────────────────

// AnalyticsProxy aggregates GoatCounter data and returns it for the admin
// analytics dashboard. Returns {"configured":false} when no API key is set.
// GET /api/admin/analytics?period=7d|30d
func (h *AdminHandler) AnalyticsProxy(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}

	if h.gcURL == "" || h.gcAPIKey == "" {
		respondJSON(w, http.StatusOK, analyticsResponse{Configured: false})
		return
	}

	period := r.URL.Query().Get("period")
	if period != "30d" {
		period = "7d"
	}

	// Return cached response if still fresh — avoids hammering GoatCounter.
	h.cacheMu.RLock()
	cached := h.cache[period]
	h.cacheMu.RUnlock()
	if cached != nil && time.Since(cached.cachedAt) < analyticsTTL {
		respondJSON(w, http.StatusOK, cached.data)
		return
	}

	days := 7
	if period == "30d" {
		days = 30
	}

	now := time.Now()
	end := now.Format("2006-01-02")
	start := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")

	// Fetch per-page hits with daily breakdown in parallel.
	type hitsResult struct {
		data []byte
		err  error
	}
	type refsResult struct {
		data []byte
		err  error
	}

	type listResult struct {
		data []byte
		err  error
	}

	hitsCh := make(chan hitsResult, 1)
	refsCh := make(chan refsResult, 1)
	browsersCh := make(chan listResult, 1)
	systemsCh := make(chan listResult, 1)
	locationsCh := make(chan listResult, 1)
	languagesCh := make(chan listResult, 1)

	statParams := fmt.Sprintf("start=%s&end=%s&limit=10", start, end)

	go func() {
		data, err := h.gcFetch(r.Context(),
			fmt.Sprintf("/api/v0/stats/hits?start=%s&end=%s&daily=true&limit=200", start, end))
		hitsCh <- hitsResult{data, err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(),
			fmt.Sprintf("/api/v0/stats/refs?%s", statParams))
		refsCh <- refsResult{data, err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/browsers?"+statParams)
		browsersCh <- listResult{data, err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/systems?"+statParams)
		systemsCh <- listResult{data, err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/locations?"+statParams)
		locationsCh <- listResult{data, err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/languages?"+statParams)
		languagesCh <- listResult{data, err}
	}()

	hitsRes := <-hitsCh
	refsRes := <-refsCh
	browsersRes := <-browsersCh
	systemsRes := <-systemsCh
	locationsRes := <-locationsCh
	languagesRes := <-languagesCh

	if hitsRes.err != nil {
		slog.Warn("analytics: failed to fetch hits from GoatCounter", "err", hitsRes.err)
		respondError(w, http.StatusBadGateway, "analytics service unavailable")
		return
	}

	var hitsResp gcHitsResponse
	if err := json.Unmarshal(hitsRes.data, &hitsResp); err != nil {
		slog.Warn("analytics: failed to parse hits response", "err", err)
		respondError(w, http.StatusBadGateway, "analytics service error")
		return
	}

	// Build daily totals by aggregating across all pages.
	dailyViews := make(map[string]int, days)
	// Seed every day in the range so missing days show as 0.
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
		dailyViews[d] = 0
	}
	for _, hit := range hitsResp.Hits {
		for _, s := range hit.Stats {
			dailyViews[s.Day] += s.Daily
		}
	}

	// Sort dates and build the daily slice.
	dates := make([]string, 0, days)
	for d := range dailyViews {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	daily := make([]analyticsDayStat, 0, len(dates))
	totalViews := 0
	for _, d := range dates {
		v := dailyViews[d]
		totalViews += v
		daily = append(daily, analyticsDayStat{Date: d, Views: v})
	}

	// Compute total unique from page-level uniques (approximation).
	totalUnique := 0
	for _, hit := range hitsResp.Hits {
		totalUnique += hit.CountUnique
	}

	// Build top 10 pages sorted by views descending.
	sorted := make([]gcHit, len(hitsResp.Hits))
	copy(sorted, hitsResp.Hits)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}
	topPages := make([]analyticsPage, 0, len(sorted))
	for _, h := range sorted {
		topPages = append(topPages, analyticsPage{
			Path:   h.Path,
			Title:  h.Title,
			Views:  h.Count,
			Unique: h.CountUnique,
		})
	}

	// Parse referrers (best-effort; ignore error).
	var topRefs []analyticsRef
	if refsRes.err == nil {
		var refsResp gcRefsResponse
		if err := json.Unmarshal(refsRes.data, &refsResp); err == nil {
			for _, ref := range refsResp.Refs {
				topRefs = append(topRefs, analyticsRef{Name: ref.Name, Views: ref.Count})
			}
		}
	}
	if topRefs == nil {
		topRefs = []analyticsRef{}
	}

	// parseStatList converts a gcStatListResponse to a deduplicated analyticsStat slice.
	parseStatList := func(data []byte, err error) []analyticsStat {
		out := []analyticsStat{}
		if err != nil {
			return out
		}
		var resp gcStatListResponse
		if jsonErr := json.Unmarshal(data, &resp); jsonErr != nil {
			return out
		}
		for _, s := range resp.Stats {
			name := s.Name
			if name == "" {
				name = s.ID
			}
			if name == "" {
				name = "(unknown)"
			}
			out = append(out, analyticsStat{Name: name, Count: s.Count})
		}
		return out
	}

	resp := &analyticsResponse{
		Configured:  true,
		Period:      period,
		TotalViews:  totalViews,
		TotalUnique: totalUnique,
		Daily:       daily,
		TopPages:    topPages,
		TopRefs:     topRefs,
		Browsers:    parseStatList(browsersRes.data, browsersRes.err),
		Systems:     parseStatList(systemsRes.data, systemsRes.err),
		Locations:   parseStatList(locationsRes.data, locationsRes.err),
		Languages:   parseStatList(languagesRes.data, languagesRes.err),
	}

	h.cacheMu.Lock()
	h.cache[period] = &analyticsCacheEntry{data: resp, cachedAt: time.Now()}
	h.cacheMu.Unlock()

	respondJSON(w, http.StatusOK, resp)
}

// gcFetch makes an authenticated GET request to the GoatCounter API.
func (h *AdminHandler) gcFetch(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.gcURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.gcAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GoatCounter returned %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// requireAdmin checks that the caller is an authenticated admin.
// Returns the user on success, or writes an error response and returns nil.
func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) *db.GetUserByIDRow {
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

	results, err := h.queries.SearchUsers(r.Context(), "%"+q+"%")
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

	// Verify target user exists and new balance won't go negative.
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
		slog.Error("AdjustUserBalance: AdminAdjustUserBalance", "err", err, "target_id", targetID)
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

// ─── Badge Release Management ─────────────────────────────────────────────────

// badgeReleaseResponse is the shape returned to the admin UI.
type badgeReleaseResponse struct {
	ID          int64   `json:"id"`
	BadgeKey    string  `json:"badge_key"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Tier        int     `json:"tier"`
	Price       int64   `json:"price"`
	Stock       *int    `json:"stock"`
	ReleasedAt  string  `json:"released_at"`
	ExpiresAt   *string `json:"expires_at"`
	Active      bool    `json:"active"`
	Notes       *string `json:"notes"`
	Insurance   string  `json:"insurance"`
	CreatedAt   string  `json:"created_at"`
	Sold        int64   `json:"sold"`
}

// badgeCatalogEntry is one entry from AllBadges returned to the admin UI.
type badgeCatalogEntry struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Purchasable bool   `json:"purchasable"`
}

func (h *AdminHandler) toReleaseResponse(ctx context.Context, rel db.BadgeReleaseRow, sold int64) badgeReleaseResponse {
	r := badgeReleaseResponse{
		ID:         rel.ID,
		BadgeKey:   rel.BadgeKey,
		Price:      rel.Price,
		Stock:      rel.Stock,
		ReleasedAt: rel.ReleasedAt.Format("2006-01-02T15:04:05Z"),
		Active:     rel.Active,
		Notes:      rel.Notes,
		CreatedAt:  rel.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Sold:       sold,
	}
	if rel.ExpiresAt != nil {
		s := rel.ExpiresAt.Format("2006-01-02T15:04:05Z")
		r.ExpiresAt = &s
	}
	if def, ok := service.BadgeKeysMap[rel.BadgeKey]; ok {
		r.Title = def.Title
		r.Description = def.Description
		r.Tier = def.Tier
	} else if dbDef, err := h.queries.GetBadgeDefinitionByKey(ctx, rel.BadgeKey); err == nil {
		r.Title = dbDef.Title
		r.Description = dbDef.Description
		r.Tier = dbDef.Tier
	}
	r.Insurance = rel.Insurance
	return r
}

// GetBadgeCatalog returns all badge definitions the admin can release.
// Merges hardcoded AllBadges (excluding admiral ranks) with custom DB definitions.
// GET /api/admin/badge-catalog
func (h *AdminHandler) GetBadgeCatalog(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()
	entries := make([]badgeCatalogEntry, 0, len(service.AllBadges))
	for _, b := range service.AllBadges {
		// Exclude admiral rank badges — they're auto-awarded by spend, not released.
		if b.SpendThreshold > 0 {
			continue
		}
		// Only purchasable badges can be scheduled in the FOMO store.
		if !b.Purchasable {
			continue
		}
		entries = append(entries, badgeCatalogEntry{
			Key:         b.Key,
			Title:       b.Title,
			Description: b.Description,
			Tier:        b.Tier,
			Purchasable: true,
		})
	}
	// Append custom (non-hardcoded) DB definitions not already in AllBadges.
	if dbDefs, err := h.queries.GetAllBadgeDefinitions(ctx); err == nil {
		for _, d := range dbDefs {
			if _, inMap := service.BadgeKeysMap[d.Key]; !inMap {
				entries = append(entries, badgeCatalogEntry{
					Key:         d.Key,
					Title:       d.Title,
					Description: d.Description,
					Tier:        d.Tier,
					Purchasable: true,
				})
			}
		}
	}
	respondJSON(w, http.StatusOK, entries)
}

// ListBadgeReleases returns all badge releases (including inactive) for the admin.
// GET /api/admin/badge-releases
func (h *AdminHandler) ListBadgeReleases(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()
	releases, err := h.queries.ListBadgeReleases(ctx)
	if err != nil {
		slog.Error("ListBadgeReleases", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	result := make([]badgeReleaseResponse, 0, len(releases))
	for _, rel := range releases {
		sold, _ := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
		result = append(result, h.toReleaseResponse(ctx, rel, sold))
	}
	respondJSON(w, http.StatusOK, result)
}

// createBadgeReleaseBody is the request body for creating/updating a release.
type createBadgeReleaseBody struct {
	BadgeKey   string  `json:"badge_key"`
	Price      int64   `json:"price"`
	Stock      *int    `json:"stock"`
	ReleasedAt string  `json:"released_at"` // RFC3339 or empty for now
	ExpiresAt  *string `json:"expires_at"`  // RFC3339 or nil
	Notes      *string `json:"notes"`
	Insurance  string  `json:"insurance"` // '6w', '120w', 'lti'
}

// CreateBadgeRelease creates a new badge release.
// POST /api/admin/badge-releases
func (h *AdminHandler) CreateBadgeRelease(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()

	var body createBadgeReleaseBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if _, ok := service.BadgeKeysMap[body.BadgeKey]; !ok {
		// Also allow admin-created badge definitions.
		if _, dbErr := h.queries.GetBadgeDefinitionByKey(ctx, body.BadgeKey); dbErr != nil {
			respondError(w, http.StatusBadRequest, "unknown badge_key")
			return
		}
	}
	if body.Price < 0 {
		respondError(w, http.StatusBadRequest, "price must be >= 0")
		return
	}
	if body.Stock != nil && *body.Stock <= 0 {
		respondError(w, http.StatusBadRequest, "stock must be a positive integer or null for unlimited")
		return
	}
	switch body.Insurance {
	case "6w", "120w", "lti":
		// valid
	default:
		respondError(w, http.StatusBadRequest, "insurance must be '6w', '120w', or 'lti'")
		return
	}

	releasedAt := time.Now().UTC()
	if body.ReleasedAt != "" {
		if t, err := time.Parse(time.RFC3339, body.ReleasedAt); err == nil {
			releasedAt = t.UTC()
		}
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, *body.ExpiresAt); err == nil {
			v := t.UTC()
			expiresAt = &v
		} else {
			respondError(w, http.StatusBadRequest, "expires_at must be RFC3339 or omitted")
			return
		}
	}

	id, err := h.queries.CreateBadgeRelease(ctx, db.CreateBadgeReleaseParams{
		BadgeKey:   body.BadgeKey,
		Price:      body.Price,
		Stock:      body.Stock,
		ReleasedAt: releasedAt,
		ExpiresAt:  expiresAt,
		Notes:      body.Notes,
		Insurance:  body.Insurance,
	})
	if err != nil {
		slog.Error("CreateBadgeRelease: db insert", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	rel, err := h.queries.GetBadgeReleaseByID(ctx, id)
	if err != nil {
		slog.Error("CreateBadgeRelease: fetch after insert", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusCreated, h.toReleaseResponse(ctx, rel, 0))
}

// UpdateBadgeRelease updates price, stock, dates, active flag, and notes on an existing release.
// PUT /api/admin/badge-releases/:id
func (h *AdminHandler) UpdateBadgeRelease(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()

	releaseID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.queries.GetBadgeReleaseByID(ctx, releaseID)
	if err != nil {
		respondError(w, http.StatusNotFound, "release not found")
		return
	}

	// Decode partial update body — reuse createBadgeReleaseBody plus active flag.
	var body struct {
		createBadgeReleaseBody
		Active bool `json:"active"`
	}
	// Default to existing values before decode so only provided fields override.
	body.Price = existing.Price
	body.Stock = existing.Stock
	body.ReleasedAt = existing.ReleasedAt.Format("2006-01-02T15:04:05Z")
	body.Active = existing.Active
	body.Notes = existing.Notes
	body.Insurance = existing.Insurance
	if existing.ExpiresAt != nil {
		s := existing.ExpiresAt.Format("2006-01-02T15:04:05Z")
		body.ExpiresAt = &s
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Price < 0 {
		respondError(w, http.StatusBadRequest, "price must be >= 0")
		return
	}
	if body.Stock != nil && *body.Stock <= 0 {
		respondError(w, http.StatusBadRequest, "stock must be a positive integer or null for unlimited")
		return
	}

	releasedAt := existing.ReleasedAt
	if body.ReleasedAt != "" {
		if t, err2 := time.Parse(time.RFC3339, body.ReleasedAt); err2 == nil {
			releasedAt = t.UTC()
		} else if t, err2 = time.Parse("2006-01-02T15:04:05Z", body.ReleasedAt); err2 == nil {
			releasedAt = t.UTC()
		}
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		if t, err2 := time.Parse(time.RFC3339, *body.ExpiresAt); err2 == nil {
			v := t.UTC()
			expiresAt = &v
		} else if t, err2 := time.Parse("2006-01-02T15:04:05Z", *body.ExpiresAt); err2 == nil {
			v := t.UTC()
			expiresAt = &v
		} else {
			respondError(w, http.StatusBadRequest, "expires_at must be RFC3339 or omitted")
			return
		}
	}

	if err := h.queries.UpdateBadgeRelease(ctx, db.UpdateBadgeReleaseParams{
		ID:         releaseID,
		Price:      body.Price,
		Stock:      body.Stock,
		ReleasedAt: releasedAt,
		ExpiresAt:  expiresAt,
		Active:     body.Active,
		Notes:      body.Notes,
		Insurance:  body.Insurance,
	}); err != nil {
		slog.Error("UpdateBadgeRelease", "err", err, "release_id", releaseID)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	rel, _ := h.queries.GetBadgeReleaseByID(ctx, releaseID)
	sold, _ := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
	respondJSON(w, http.StatusOK, h.toReleaseResponse(ctx, rel, sold))
}

// ArchiveBadgeRelease sets a release to inactive (soft-delete).
// DELETE /api/admin/badge-releases/:id
func (h *AdminHandler) ArchiveBadgeRelease(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	releaseID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.queries.ArchiveBadgeRelease(r.Context(), releaseID); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// ── Badge Definitions ──────────────────────────────────────────────────────

// badgeDefinitionResponse is the API shape for a badge_definitions row.
type badgeDefinitionResponse struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Icon        string `json:"icon"`
	IsHardcoded bool   `json:"is_hardcoded"`
	Purchasable bool   `json:"purchasable"`
	Insurance   string `json:"insurance"`
	CreatedAt   string `json:"created_at"`
}

func toBadgeDefResponse(d db.BadgeDefinitionRow) badgeDefinitionResponse {
	// Determine purchasable: custom (non-hardcoded) defs are always purchasable;
	// hardcoded defs inherit Purchasable from the Go service definition.
	purchasable := true
	if d.IsHardcoded {
		if svc, ok := service.BadgeKeysMap[d.Key]; ok {
			purchasable = svc.Purchasable
		}
	}
	return badgeDefinitionResponse{
		ID:          d.ID,
		Key:         d.Key,
		Title:       d.Title,
		Description: d.Description,
		Tier:        d.Tier,
		Icon:        d.Icon,
		IsHardcoded: d.IsHardcoded,
		Purchasable: purchasable,
		Insurance:   d.Insurance,
		CreatedAt:   d.CreatedAt,
	}
}

// ListBadgeDefinitions returns all rows from badge_definitions.
// GET /api/admin/badge-definitions
func (h *AdminHandler) ListBadgeDefinitions(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	defs, err := h.queries.GetAllBadgeDefinitions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	result := make([]badgeDefinitionResponse, 0, len(defs))
	for _, d := range defs {
		result = append(result, toBadgeDefResponse(d))
	}
	respondJSON(w, http.StatusOK, result)
}

// createBadgeDefinitionBody is the request body for creating a new custom badge.
type createBadgeDefinitionBody struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Icon        string `json:"icon"`
	Insurance   string `json:"insurance"` // '', '6w', '120w', 'lti'
}

// CreateBadgeDefinition creates a new custom (non-hardcoded) badge definition.
// POST /api/admin/badge-definitions
func (h *AdminHandler) CreateBadgeDefinition(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()

	var body createBadgeDefinitionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	body.Key = strings.TrimSpace(body.Key)
	body.Title = strings.TrimSpace(body.Title)
	body.Description = strings.TrimSpace(body.Description)
	body.Icon = strings.TrimSpace(body.Icon)

	if body.Key == "" || body.Title == "" || body.Description == "" {
		respondError(w, http.StatusBadRequest, "key, title and description are required")
		return
	}
	if body.Tier < 1 || body.Tier > 5 {
		respondError(w, http.StatusBadRequest, "tier must be between 1 and 5")
		return
	}
	validInsurance := map[string]bool{"": true, "6w": true, "120w": true, "lti": true}
	if !validInsurance[body.Insurance] {
		respondError(w, http.StatusBadRequest, "insurance must be one of: '', '6w', '120w', 'lti'")
		return
	}

	// Reject keys that collide with hardcoded badges to avoid confusion.
	if _, exists := service.BadgeKeysMap[body.Key]; exists {
		respondError(w, http.StatusConflict, "key already exists as a hardcoded badge")
		return
	}

	id, err := h.queries.CreateBadgeDefinition(ctx, db.CreateBadgeDefinitionParams{
		Key:         body.Key,
		Title:       body.Title,
		Description: body.Description,
		Tier:        body.Tier,
		Icon:        body.Icon,
		Insurance:   body.Insurance,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			respondError(w, http.StatusConflict, "badge key already exists")
			return
		}
		slog.Error("CreateBadgeDefinition", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	def, err := h.queries.GetBadgeDefinitionByKey(ctx, body.Key)
	if err != nil {
		respondJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	respondJSON(w, http.StatusCreated, toBadgeDefResponse(def))
}

// UpdateBadgeDefinition updates title, description, tier and icon on a badge definition.
// Hardcoded badges can also be updated (the is_hardcoded flag is read-only).
// PUT /api/admin/badge-definitions/:key
func (h *AdminHandler) UpdateBadgeDefinition(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	existing, err := h.queries.GetBadgeDefinitionByKey(ctx, key)
	if err != nil {
		respondError(w, http.StatusNotFound, "badge definition not found")
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Tier        int    `json:"tier"`
		Icon        string `json:"icon"`
		Insurance   string `json:"insurance"`
	}
	// Seed with existing values so partial updates work.
	body.Title = existing.Title
	body.Description = existing.Description
	body.Tier = existing.Tier
	body.Icon = existing.Icon
	body.Insurance = existing.Insurance

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	body.Description = strings.TrimSpace(body.Description)
	body.Icon = strings.TrimSpace(body.Icon)

	if body.Title == "" || body.Description == "" {
		respondError(w, http.StatusBadRequest, "title and description are required")
		return
	}
	if body.Tier < 1 || body.Tier > 5 {
		respondError(w, http.StatusBadRequest, "tier must be between 1 and 5")
		return
	}
	validIns := map[string]bool{"": true, "6w": true, "120w": true, "lti": true}
	if !validIns[body.Insurance] {
		respondError(w, http.StatusBadRequest, "insurance must be one of: '', '6w', '120w', 'lti'")
		return
	}

	if err := h.queries.UpdateBadgeDefinition(ctx, db.UpdateBadgeDefinitionParams{
		Key:         key,
		Title:       body.Title,
		Description: body.Description,
		Tier:        body.Tier,
		Icon:        body.Icon,
		Insurance:   body.Insurance,
	}); err != nil {
		slog.Error("UpdateBadgeDefinition", "err", err, "key", key)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	def, err := h.queries.GetBadgeDefinitionByKey(ctx, key)
	if err != nil {
		slog.Error("UpdateBadgeDefinition: fetch after update", "err", err, "key", key)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, toBadgeDefResponse(def))
}
