package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/service"
)

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
		ID:          rel.ID,
		BadgeKey:    rel.BadgeKey,
		Price:       rel.Price,
		ReleasedAt:  rel.ReleasedAt.Format(time.RFC3339),
		Active:      rel.Active,
		Insurance:   rel.Insurance,
		CreatedAt:   rel.CreatedAt.Format(time.RFC3339),
		Sold:        sold,
		Title:       rel.BadgeKey,
		Description: "",
		Tier:        1,
	}
	if rel.Stock != nil {
		stock := int(*rel.Stock)
		r.Stock = &stock
	}
	if rel.ExpiresAt != nil {
		expiresAt := rel.ExpiresAt.Format(time.RFC3339)
		r.ExpiresAt = &expiresAt
	}
	if rel.Notes != nil {
		r.Notes = rel.Notes
	}
	if def, ok := service.BadgeKeysMap[rel.BadgeKey]; ok {
		r.Title = def.Title
		r.Description = def.Description
		r.Tier = def.Tier
		return r
	}
	if def, err := h.queries.GetBadgeDefinitionByKey(ctx, rel.BadgeKey); err == nil {
		r.Title = def.Title
		r.Description = def.Description
		r.Tier = def.Tier
	}
	return r
}

// GetBadgeCatalog returns all badge definitions that can be scheduled as releases.
// GET /api/admin/badge-catalog
func (h *AdminHandler) GetBadgeCatalog(w http.ResponseWriter, r *http.Request) {
	if h.requireAdmin(w, r) == nil {
		return
	}

	entries := make([]badgeCatalogEntry, 0, len(service.AllBadges))
	for _, badge := range service.AllBadges {
		if !badge.Purchasable {
			continue
		}
		entries = append(entries, badgeCatalogEntry{
			Key:         badge.Key,
			Title:       badge.Title,
			Description: badge.Description,
			Tier:        badge.Tier,
			Purchasable: badge.Purchasable,
		})
	}

	defs, err := h.queries.GetAllBadgeDefinitions(r.Context())
	if err == nil {
		for _, def := range defs {
			if def.IsHardcoded {
				continue
			}
			entries = append(entries, badgeCatalogEntry{
				Key:         def.Key,
				Title:       def.Title,
				Description: def.Description,
				Tier:        def.Tier,
				Purchasable: true,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Tier == entries[j].Tier {
			return entries[i].Title < entries[j].Title
		}
		return entries[i].Tier < entries[j].Tier
	})

	respondJSON(w, http.StatusOK, entries)
}

// ListBadgeReleases lists all badge store release windows.
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

	out := make([]badgeReleaseResponse, 0, len(releases))
	for _, rel := range releases {
		sold, _ := h.queries.CountBadgePurchases(ctx, rel.BadgeKey)
		out = append(out, h.toReleaseResponse(ctx, rel, sold))
	}
	respondJSON(w, http.StatusOK, out)
}

type createBadgeReleaseBody struct {
	BadgeKey   string  `json:"badge_key"`
	Price      int64   `json:"price"`
	Stock      *int    `json:"stock"`
	ReleasedAt string  `json:"released_at"`
	ExpiresAt  *string `json:"expires_at"`
	Notes      *string `json:"notes"`
	Insurance  string  `json:"insurance"`
}

// CreateBadgeRelease creates a new badge release window.
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
	if body.BadgeKey == "" {
		respondError(w, http.StatusBadRequest, "badge_key is required")
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
	switch body.Insurance {
	case "6w", "120w", "lti":
	default:
		respondError(w, http.StatusBadRequest, "insurance must be '6w', '120w', or 'lti'")
		return
	}

	releasedAt := time.Now().UTC()
	if body.ReleasedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, body.ReleasedAt); err == nil {
			releasedAt = parsed.UTC()
		}
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, *body.ExpiresAt); err == nil {
			value := parsed.UTC()
			expiresAt = &value
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

	var body struct {
		createBadgeReleaseBody
		Active bool `json:"active"`
	}
	body.Price = existing.Price
	body.Stock = existing.Stock
	body.ReleasedAt = existing.ReleasedAt.Format("2006-01-02T15:04:05Z")
	body.Active = existing.Active
	body.Notes = existing.Notes
	body.Insurance = existing.Insurance
	if existing.ExpiresAt != nil {
		value := existing.ExpiresAt.Format("2006-01-02T15:04:05Z")
		body.ExpiresAt = &value
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
		if parsed, err2 := time.Parse(time.RFC3339, body.ReleasedAt); err2 == nil {
			releasedAt = parsed.UTC()
		} else if parsed, err2 := time.Parse("2006-01-02T15:04:05Z", body.ReleasedAt); err2 == nil {
			releasedAt = parsed.UTC()
		}
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		if parsed, err2 := time.Parse(time.RFC3339, *body.ExpiresAt); err2 == nil {
			value := parsed.UTC()
			expiresAt = &value
		} else if parsed, err2 := time.Parse("2006-01-02T15:04:05Z", *body.ExpiresAt); err2 == nil {
			value := parsed.UTC()
			expiresAt = &value
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
	for _, def := range defs {
		result = append(result, toBadgeDefResponse(def))
	}
	respondJSON(w, http.StatusOK, result)
}

type createBadgeDefinitionBody struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Icon        string `json:"icon"`
	Insurance   string `json:"insurance"`
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
	validInsurance := map[string]bool{"": true, "6w": true, "120w": true, "lti": true}
	if !validInsurance[body.Insurance] {
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
