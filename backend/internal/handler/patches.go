package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scolymarket/internal/db"
)

// PatchHandler serves patch detection endpoints.
type PatchHandler struct {
	queries *db.Queries
}

// NewPatchHandler creates a PatchHandler.
func NewPatchHandler(q *db.Queries) *PatchHandler {
	return &PatchHandler{queries: q}
}

// List returns all detected patches newest-first.
// GET /api/patches
func (h *PatchHandler) List(w http.ResponseWriter, r *http.Request) {
	patches, err := h.queries.ListAllPatches(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"patches": patches})
}

// MarkNotified marks a patch as seen by a moderator.
// POST /api/mod/patches/:id/notify
func (h *PatchHandler) MarkNotified(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.queries.MarkPatchNotified(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true})
}
