package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireTrustedOriginAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	mw := RequireTrustedOrigin([]string{"https://app.example.com"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/trades", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRequireTrustedOriginAllowsRefererFallback(t *testing.T) {
	t.Parallel()

	mw := RequireTrustedOrigin([]string{"https://app.example.com"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/markets", nil)
	req.Header.Set("Referer", "https://app.example.com/markets/new")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRequireTrustedOriginRejectsMissingOrigin(t *testing.T) {
	t.Parallel()

	mw := RequireTrustedOrigin([]string{"https://app.example.com"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/reports", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireTrustedOriginSkipsSafeMethods(t *testing.T) {
	t.Parallel()

	mw := RequireTrustedOrigin([]string{"https://app.example.com"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/markets", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}
