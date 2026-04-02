package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lhw/astroledger/internal/database"
	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
)

func newAdminHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	sqlDB, err := database.Open(context.Background(), t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB, db.New(sqlDB)
}

func createAdminUser(t *testing.T, ctx context.Context, q *db.Queries) db.User {
	t.Helper()
	created, err := q.CreateUser(ctx, db.CreateUserParams{
		ScidSub:     "test:admin",
		DisplayName: "Admin Tester",
		Email:       "admin@test.example",
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := q.UpdateUserGroups(ctx, created.ID, 0, 1, 0); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	user, err := q.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("re-fetch admin user: %v", err)
	}
	return db.User{
		ID:          user.ID,
		ScidSub:     user.ScidSub,
		DisplayName: user.DisplayName,
		Email:       user.Email,
	}
}

func adminRequest(t *testing.T, method, path string, body any, userID int64) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, &middleware.Claims{UserID: userID})
	return req.WithContext(ctx)
}

func TestCreateBadgeReleasePersistsAndReturnsCreatedRow(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "")

	body := map[string]any{
		"badge_key":   "aurora_pilot",
		"price":       500,
		"stock":       25,
		"released_at": "2026-04-02T10:00:00Z",
		"expires_at":  "2026-04-30T23:59:00Z",
		"notes":       "CitizenCon drop",
		"insurance":   "6w",
	}

	rec := httptest.NewRecorder()
	h.CreateBadgeRelease(rec, adminRequest(t, http.MethodPost, "/api/admin/badge-releases", body, admin.ID))

	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateBadgeRelease status: want %d, got %d, body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var created badgeReleaseResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.BadgeKey != "aurora_pilot" {
		t.Fatalf("created badge_key: want aurora_pilot, got %q", created.BadgeKey)
	}
	if created.Price != 500 {
		t.Fatalf("created price: want 500, got %d", created.Price)
	}
	if created.Insurance != "6w" {
		t.Fatalf("created insurance: want 6w, got %q", created.Insurance)
	}

	releases, err := queries.ListBadgeReleases(ctx)
	if err != nil {
		t.Fatalf("ListBadgeReleases: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("persisted releases: want 1, got %d", len(releases))
	}
	if releases[0].BadgeKey != "aurora_pilot" || releases[0].Insurance != "6w" || releases[0].Price != 500 {
		t.Fatalf("persisted release mismatch: %+v", releases[0])
	}

	listRec := httptest.NewRecorder()
	h.ListBadgeReleases(listRec, adminRequest(t, http.MethodGet, "/api/admin/badge-releases", nil, admin.ID))
	if listRec.Code != http.StatusOK {
		t.Fatalf("ListBadgeReleases status: want %d, got %d, body=%s", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	var listed []badgeReleaseResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed releases: want 1, got %d", len(listed))
	}
	if listed[0].ID != created.ID {
		t.Fatalf("listed ID: want %d, got %d", created.ID, listed[0].ID)
	}
}

func TestCreateBadgeReleaseRejectsInvalidInsurance(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "")

	body := map[string]any{
		"badge_key":   "aurora_pilot",
		"price":       500,
		"released_at": "2026-04-02T10:00:00Z",
		"insurance":   "",
	}

	rec := httptest.NewRecorder()
	h.CreateBadgeRelease(rec, adminRequest(t, http.MethodPost, "/api/admin/badge-releases", body, admin.ID))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("CreateBadgeRelease invalid insurance status: want %d, got %d, body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	releases, err := queries.ListBadgeReleases(ctx)
	if err != nil {
		t.Fatalf("ListBadgeReleases after invalid request: %v", err)
	}
	if len(releases) != 0 {
		t.Fatalf("persisted releases after invalid request: want 0, got %d", len(releases))
	}
}
