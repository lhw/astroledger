package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
)

func withRouteParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestUpdateAndArchiveBadgeRelease(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "", nil)

	releaseID, err := queries.CreateBadgeRelease(ctx, db.CreateBadgeReleaseParams{
		BadgeKey:   "aurora_pilot",
		Price:      500,
		ReleasedAt: time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC),
		Insurance:  "6w",
	})
	if err != nil {
		t.Fatalf("CreateBadgeRelease seed: %v", err)
	}

	updateReq := adminRequest(t, http.MethodPut, "/api/admin/badge-releases/1", map[string]any{
		"price":       750,
		"stock":       10,
		"active":      true,
		"released_at": "2026-05-01T08:00:00Z",
		"expires_at":  "2026-05-31T23:59:00Z",
		"notes":       "Invictus rerun",
		"insurance":   "120w",
	}, admin.ID)
	updateReq = withRouteParam(updateReq, "id", strconv.FormatInt(releaseID, 10))
	updateRec := httptest.NewRecorder()
	h.UpdateBadgeRelease(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("UpdateBadgeRelease status: want %d, got %d, body=%s", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	var updated badgeReleaseResponse
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Price != 750 {
		t.Fatalf("updated price: want 750, got %d", updated.Price)
	}
	if updated.Stock == nil || *updated.Stock != 10 {
		t.Fatalf("updated stock: want 10, got %#v", updated.Stock)
	}
	if updated.Insurance != "120w" {
		t.Fatalf("updated insurance: want 120w, got %q", updated.Insurance)
	}

	stored, err := queries.GetBadgeReleaseByID(ctx, releaseID)
	if err != nil {
		t.Fatalf("GetBadgeReleaseByID after update: %v", err)
	}
	if stored.Price != 750 || stored.Insurance != "120w" {
		t.Fatalf("stored release mismatch after update: %+v", stored)
	}
	if stored.Stock == nil || *stored.Stock != 10 {
		t.Fatalf("stored stock after update: %#v", stored.Stock)
	}

	archiveReq := adminRequest(t, http.MethodDelete, "/api/admin/badge-releases/1", nil, admin.ID)
	archiveReq = withRouteParam(archiveReq, "id", strconv.FormatInt(releaseID, 10))
	archiveRec := httptest.NewRecorder()
	h.ArchiveBadgeRelease(archiveRec, archiveReq)

	if archiveRec.Code != http.StatusOK {
		t.Fatalf("ArchiveBadgeRelease status: want %d, got %d, body=%s", http.StatusOK, archiveRec.Code, archiveRec.Body.String())
	}

	archived, err := queries.GetBadgeReleaseByID(ctx, releaseID)
	if err != nil {
		t.Fatalf("GetBadgeReleaseByID after archive: %v", err)
	}
	if archived.Active {
		t.Fatalf("expected archived release to be inactive, got active=%v", archived.Active)
	}
}

func TestCreateAndUpdateBadgeDefinition(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "", nil)

	createReq := adminRequest(t, http.MethodPost, "/api/admin/badge-definitions", map[string]any{
		"key":         "test_badge_custom",
		"title":       "Custom Test Badge",
		"description": "Awarded for validating admin badge handlers.",
		"tier":        3,
		"icon":        "TB",
		"insurance":   "lti",
	}, admin.ID)
	createRec := httptest.NewRecorder()
	h.CreateBadgeDefinition(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("CreateBadgeDefinition status: want %d, got %d, body=%s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}

	var created badgeDefinitionResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Key != "test_badge_custom" {
		t.Fatalf("created key: want test_badge_custom, got %q", created.Key)
	}
	if created.Insurance != "lti" {
		t.Fatalf("created insurance: want lti, got %q", created.Insurance)
	}

	updateReq := adminRequest(t, http.MethodPut, "/api/admin/badge-definitions/test_badge_custom", map[string]any{
		"title":       "Updated Test Badge",
		"description": "Updated definition after split.",
		"tier":        4,
		"icon":        "UTB",
		"insurance":   "120w",
	}, admin.ID)
	updateReq = withRouteParam(updateReq, "key", "test_badge_custom")
	updateRec := httptest.NewRecorder()
	h.UpdateBadgeDefinition(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("UpdateBadgeDefinition status: want %d, got %d, body=%s", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	var updated badgeDefinitionResponse
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Title != "Updated Test Badge" || updated.Tier != 4 || updated.Icon != "UTB" || updated.Insurance != "120w" {
		t.Fatalf("updated badge definition mismatch: %+v", updated)
	}

	listRec := httptest.NewRecorder()
	h.ListBadgeDefinitions(listRec, adminRequest(t, http.MethodGet, "/api/admin/badge-definitions", nil, admin.ID))
	if listRec.Code != http.StatusOK {
		t.Fatalf("ListBadgeDefinitions status: want %d, got %d, body=%s", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	var listed []badgeDefinitionResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	found := false
	for _, def := range listed {
		if def.Key == "test_badge_custom" {
			found = true
			if def.Title != "Updated Test Badge" || def.Insurance != "120w" {
				t.Fatalf("listed badge definition mismatch: %+v", def)
			}
		}
	}
	if !found {
		t.Fatal("updated badge definition not returned by ListBadgeDefinitions")
	}
}

func TestCreateBadgeDefinitionRejectsHardcodedKey(t *testing.T) {
	ctx := context.Background()
	_, queries := newAdminHandlerTestDB(t)
	admin := createAdminUser(t, ctx, queries)
	h := NewAdminHandler(queries, nil, "", "", nil)

	req := adminRequest(t, http.MethodPost, "/api/admin/badge-definitions", map[string]any{
		"key":         "first_blood",
		"title":       "Should Fail",
		"description": "Conflict with hardcoded badge.",
		"tier":        2,
		"icon":        "CF",
		"insurance":   "6w",
	}, admin.ID)
	rec := httptest.NewRecorder()
	h.CreateBadgeDefinition(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("CreateBadgeDefinition hardcoded conflict status: want %d, got %d, body=%s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	defs, err := queries.GetAllBadgeDefinitions(ctx)
	if err != nil {
		t.Fatalf("GetAllBadgeDefinitions after conflict: %v", err)
	}
	for _, def := range defs {
		if def.Key == "first_blood" && !def.IsHardcoded {
			t.Fatal("unexpected custom first_blood badge definition was created")
		}
	}
}
