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
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
	"github.com/lhw/astroledger/internal/service"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	queries    *db.Queries
	creditsSvc *service.CreditsService
	gcURL      string // GoatCounter internal base URL
	gcAPIKey   string // GoatCounter API Bearer token
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(queries *db.Queries, creditsSvc *service.CreditsService, gcURL, gcAPIKey string) *AdminHandler {
	return &AdminHandler{queries: queries, creditsSvc: creditsSvc, gcURL: gcURL, gcAPIKey: gcAPIKey}
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

	respondJSON(w, http.StatusOK, analyticsResponse{
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
	})
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
	CreatedAt   string  `json:"created_at"`
	Sold        int64   `json:"sold"`
}

// badgeCatalogEntry is one entry from AllBadges returned to the admin UI.
type badgeCatalogEntry struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
}

func toReleaseResponse(rel db.BadgeReleaseRow, sold int64) badgeReleaseResponse {
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
	}
	return r
}

// GetBadgeCatalog returns all badge definitions the admin can release.
// GET /api/admin/badge-catalog
func (h *AdminHandler) GetBadgeCatalog(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}
	entries := make([]badgeCatalogEntry, 0, len(service.AllBadges))
	for _, b := range service.AllBadges {
		// Exclude admiral rank badges — they're auto-awarded by spend, not released.
		if b.SpendThreshold > 0 {
			continue
		}
		entries = append(entries, badgeCatalogEntry{
			Key:         b.Key,
			Title:       b.Title,
			Description: b.Description,
			Tier:        b.Tier,
		})
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
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	result := make([]badgeReleaseResponse, 0, len(releases))
	for _, rel := range releases {
		sold, _ := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
		result = append(result, toReleaseResponse(rel, sold))
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
		respondError(w, http.StatusBadRequest, "unknown badge_key")
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
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	rel, err := h.queries.GetBadgeReleaseByID(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusCreated, toReleaseResponse(rel, 0))
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
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	rel, _ := h.queries.GetBadgeReleaseByID(ctx, releaseID)
	sold, _ := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
	respondJSON(w, http.StatusOK, toReleaseResponse(rel, sold))
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
