package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
)

func createRegularUser(t *testing.T, ctx context.Context, q *db.Queries, scidSub, displayName, email string) db.CreateUserRow {
	t.Helper()
	created, err := q.CreateUser(ctx, db.CreateUserParams{
		ScidSub:     scidSub,
		DisplayName: displayName,
		Email:       email,
	})
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}
	return created
}

func TestSearchUsersReturnsMatchesByDisplayName(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	createRegularUser(t, ctx, queries, "test:user:one", "PilotAlpha", "pilot-alpha@test.example")
	createRegularUser(t, ctx, queries, "test:user:two", "CargoMule", "cargo-mule@test.example")
	h := NewAdminHandler(queries, nil, "", "")

	req := adminRequest(t, http.MethodGet, "/api/admin/users/search?q=pilot", nil, admin.ID)
	rec := httptest.NewRecorder()
	h.SearchUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("SearchUsers status: want %d, got %d, body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var results []db.SearchUsersRow
	if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("search results length: want 1, got %d", len(results))
	}
	if results[0].DisplayName != "PilotAlpha" {
		t.Fatalf("search result display_name: want PilotAlpha, got %q", results[0].DisplayName)
	}
}

func TestAdjustUserBalanceRejectsOverdraw(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	target := createRegularUser(t, ctx, queries, "test:user:three", "WalletLight", "wallet-light@test.example")
	h := NewAdminHandler(queries, nil, "", "")

	req := adminRequest(t, http.MethodPost, "/api/admin/users/2/balance", map[string]any{
		"amount": -2000,
		"reason": "test overdraw",
	}, admin.ID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(target.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.AdjustUserBalance(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("AdjustUserBalance status: want %d, got %d, body=%s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}

	stored, err := queries.GetUserByID(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetUserByID after adjustment: %v", err)
	}
	if stored.Balance != target.Balance {
		t.Fatalf("user balance changed unexpectedly: want %d, got %d", target.Balance, stored.Balance)
	}
}

func TestBanAndShadowBanUserUpdateFlags(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	target := createRegularUser(t, ctx, queries, "test:user:four", "MutedPilot", "muted-pilot@test.example")
	h := NewAdminHandler(queries, nil, "", "")

	banReq := adminRequest(t, http.MethodPost, "/api/admin/users/ban", map[string]any{"banned": true}, admin.ID)
	banCtx := chi.NewRouteContext()
	banCtx.URLParams.Add("id", strconv.FormatInt(target.ID, 10))
	banReq = banReq.WithContext(context.WithValue(banReq.Context(), chi.RouteCtxKey, banCtx))
	banRec := httptest.NewRecorder()
	h.BanUser(banRec, banReq)
	if banRec.Code != http.StatusOK {
		t.Fatalf("BanUser status: want %d, got %d, body=%s", http.StatusOK, banRec.Code, banRec.Body.String())
	}

	shadowReq := adminRequest(t, http.MethodPost, "/api/admin/users/shadow-ban", map[string]any{"shadow_banned": true}, admin.ID)
	shadowCtx := chi.NewRouteContext()
	shadowCtx.URLParams.Add("id", strconv.FormatInt(target.ID, 10))
	shadowReq = shadowReq.WithContext(context.WithValue(shadowReq.Context(), chi.RouteCtxKey, shadowCtx))
	shadowRec := httptest.NewRecorder()
	h.ShadowBanUser(shadowRec, shadowReq)
	if shadowRec.Code != http.StatusOK {
		t.Fatalf("ShadowBanUser status: want %d, got %d, body=%s", http.StatusOK, shadowRec.Code, shadowRec.Body.String())
	}

	results, err := queries.SearchUsers(ctx, "%MutedPilot%")
	if err != nil {
		t.Fatalf("SearchUsers after moderation updates: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
	if results[0].IsBanned != 1 {
		t.Fatalf("expected IsBanned=1, got %d", results[0].IsBanned)
	}
	if results[0].IsShadowBanned != 1 {
		t.Fatalf("expected IsShadowBanned=1, got %d", results[0].IsShadowBanned)
	}
}
