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
	"sync"
	"time"

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

// AnalyticsProxy aggregates GoatCounter data for the admin analytics dashboard.
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
		data, err := h.gcFetch(r.Context(), fmt.Sprintf("/api/v0/stats/hits?start=%s&end=%s&daily=true&limit=200", start, end))
		hitsCh <- hitsResult{data: data, err: err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), fmt.Sprintf("/api/v0/stats/refs?%s", statParams))
		refsCh <- refsResult{data: data, err: err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/browsers?"+statParams)
		browsersCh <- listResult{data: data, err: err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/systems?"+statParams)
		systemsCh <- listResult{data: data, err: err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/locations?"+statParams)
		locationsCh <- listResult{data: data, err: err}
	}()
	go func() {
		data, err := h.gcFetch(r.Context(), "/api/v0/stats/languages?"+statParams)
		languagesCh <- listResult{data: data, err: err}
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

	dailyViews := make(map[string]int, days)
	for index := 0; index < days; index++ {
		date := now.AddDate(0, 0, -(days - 1 - index)).Format("2006-01-02")
		dailyViews[date] = 0
	}
	for _, hit := range hitsResp.Hits {
		for _, stat := range hit.Stats {
			dailyViews[stat.Day] += stat.Daily
		}
	}

	dates := make([]string, 0, days)
	for date := range dailyViews {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	daily := make([]analyticsDayStat, 0, len(dates))
	totalViews := 0
	for _, date := range dates {
		views := dailyViews[date]
		totalViews += views
		daily = append(daily, analyticsDayStat{Date: date, Views: views})
	}

	totalUnique := 0
	for _, hit := range hitsResp.Hits {
		totalUnique += hit.CountUnique
	}

	sortedHits := make([]gcHit, len(hitsResp.Hits))
	copy(sortedHits, hitsResp.Hits)
	sort.Slice(sortedHits, func(i, j int) bool {
		return sortedHits[i].Count > sortedHits[j].Count
	})
	if len(sortedHits) > 10 {
		sortedHits = sortedHits[:10]
	}
	topPages := make([]analyticsPage, 0, len(sortedHits))
	for _, hit := range sortedHits {
		topPages = append(topPages, analyticsPage{
			Path:   hit.Path,
			Title:  hit.Title,
			Views:  hit.Count,
			Unique: hit.CountUnique,
		})
	}

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

	parseStatList := func(data []byte, err error) []analyticsStat {
		out := []analyticsStat{}
		if err != nil {
			return out
		}
		var resp gcStatListResponse
		if jsonErr := json.Unmarshal(data, &resp); jsonErr != nil {
			return out
		}
		for _, stat := range resp.Stats {
			name := stat.Name
			if name == "" {
				name = stat.ID
			}
			if name == "" {
				name = "(unknown)"
			}
			out = append(out, analyticsStat{Name: name, Count: stat.Count})
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GoatCounter returned %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

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
