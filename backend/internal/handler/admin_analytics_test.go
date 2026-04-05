package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAnalyticsProxyRejectsUnauthorized(t *testing.T) {
	_, queries := newAdminHandlerTestDB(t)
	h := NewAdminHandler(queries, nil, "", "", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics", nil)
	rec := httptest.NewRecorder()
	h.AnalyticsProxy(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("AnalyticsProxy status: want %d, got %d, body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestAnalyticsProxyReturnsConfiguredFalseWithoutGoatCounter(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "", nil)

	rec := httptest.NewRecorder()
	h.AnalyticsProxy(rec, adminRequest(t, http.MethodGet, "/api/admin/analytics", nil, admin.ID))

	if rec.Code != http.StatusOK {
		t.Fatalf("AnalyticsProxy status: want %d, got %d, body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp analyticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Configured {
		t.Fatal("expected configured=false when GoatCounter is disabled")
	}
}

func TestAnalyticsProxyAggregatesAndCachesResponses(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	var mu sync.Mutex
	requestCounts := map[string]int{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCounts[r.URL.Path]++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/hits"):
			_ = json.NewEncoder(w).Encode(gcHitsResponse{Hits: []gcHit{
				{
					Path:        "/markets/1",
					Title:       "Cargo Elevator",
					Count:       10,
					CountUnique: 4,
					Stats:       []gcDayStat{{Day: yesterday, Daily: 3}, {Day: today, Daily: 7}},
				},
				{
					Path:        "/fomo",
					Title:       "FOMO Store",
					Count:       6,
					CountUnique: 2,
					Stats:       []gcDayStat{{Day: yesterday, Daily: 1}, {Day: today, Daily: 5}},
				},
			}})
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/refs"):
			_ = json.NewEncoder(w).Encode(gcRefsResponse{Refs: []gcRef{{Name: "reddit", Count: 8}, {Name: "spectrum", Count: 3}}})
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/browsers"):
			_ = json.NewEncoder(w).Encode(gcStatListResponse{Stats: []gcStatItem{{Name: "Firefox", Count: 9}}})
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/systems"):
			_ = json.NewEncoder(w).Encode(gcStatListResponse{Stats: []gcStatItem{{Name: "macOS", Count: 7}}})
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/locations"):
			_ = json.NewEncoder(w).Encode(gcStatListResponse{Stats: []gcStatItem{{Name: "EU", Count: 5}}})
		case strings.HasPrefix(r.URL.Path, "/api/v0/stats/languages"):
			_ = json.NewEncoder(w).Encode(gcStatListResponse{Stats: []gcStatItem{{Name: "en", Count: 11}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := NewAdminHandler(queries, nil, server.URL, "test-api-key", nil)

	firstRec := httptest.NewRecorder()
	h.AnalyticsProxy(firstRec, adminRequest(t, http.MethodGet, "/api/admin/analytics?period=7d", nil, admin.ID))
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first AnalyticsProxy status: want %d, got %d, body=%s", http.StatusOK, firstRec.Code, firstRec.Body.String())
	}

	var first analyticsResponse
	if err := json.NewDecoder(firstRec.Body).Decode(&first); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if !first.Configured {
		t.Fatal("expected configured=true when GoatCounter is enabled")
	}
	if first.Period != "7d" {
		t.Fatalf("period: want 7d, got %q", first.Period)
	}
	if first.TotalViews != 16 {
		t.Fatalf("total views: want 16, got %d", first.TotalViews)
	}
	if first.TotalUnique != 6 {
		t.Fatalf("total unique: want 6, got %d", first.TotalUnique)
	}
	if len(first.TopPages) != 2 || first.TopPages[0].Path != "/markets/1" {
		t.Fatalf("top pages not sorted as expected: %+v", first.TopPages)
	}
	if len(first.TopRefs) != 2 || first.TopRefs[0].Name != "reddit" {
		t.Fatalf("top refs mismatch: %+v", first.TopRefs)
	}
	if len(first.Browsers) != 1 || first.Browsers[0].Name != "Firefox" {
		t.Fatalf("browsers mismatch: %+v", first.Browsers)
	}

	secondRec := httptest.NewRecorder()
	h.AnalyticsProxy(secondRec, adminRequest(t, http.MethodGet, "/api/admin/analytics?period=7d", nil, admin.ID))
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second AnalyticsProxy status: want %d, got %d, body=%s", http.StatusOK, secondRec.Code, secondRec.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCounts["/api/v0/stats/hits"] != 1 {
		t.Fatalf("expected cached second call to avoid extra hits fetches, got %d", requestCounts["/api/v0/stats/hits"])
	}
	if requestCounts["/api/v0/stats/refs"] != 1 || requestCounts["/api/v0/stats/browsers"] != 1 || requestCounts["/api/v0/stats/systems"] != 1 || requestCounts["/api/v0/stats/locations"] != 1 || requestCounts["/api/v0/stats/languages"] != 1 {
		t.Fatalf("expected one upstream request per endpoint, got counts=%v", requestCounts)
	}
}
