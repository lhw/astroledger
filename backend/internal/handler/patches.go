package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/astroledger/internal/db"
	"github.com/lhw/astroledger/internal/middleware"
)

// PatchHandler serves patch detection endpoints.
type PatchHandler struct {
	queries *db.Queries
}

// NewPatchHandler creates a PatchHandler.
func NewPatchHandler(q *db.Queries) *PatchHandler {
	return &PatchHandler{queries: q}
}

// List returns all detected patches newest-first, each enriched with any active
// markets whose resolution criteria references that patch version.
// GET /api/patches
func (h *PatchHandler) List(w http.ResponseWriter, r *http.Request) {
	patches, err := h.queries.ListAllPatches(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	type patchWithMarkets struct {
		db.DetectedPatch
		RelatedMarkets []db.PatchMarket `json:"related_markets"`
	}

	result := make([]patchWithMarkets, 0, len(patches))
	for _, p := range patches {
		markets, err := h.queries.ListActiveMarketsForPatch(r.Context(), p.PatchVersion)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if markets == nil {
			markets = []db.PatchMarket{}
		}
		result = append(result, patchWithMarkets{DetectedPatch: p, RelatedMarkets: markets})
	}

	respondJSON(w, http.StatusOK, map[string]any{"patches": result})
}

// MarkNotified marks a patch as seen by a moderator and queues any related
// active markets for resolution.
// POST /api/mod/patches/:id/notify
func (h *PatchHandler) MarkNotified(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Fetch the patch so we can look up related markets.
	patch, err := h.queries.GetPatchByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "patch not found")
		return
	}

	if err := h.queries.MarkPatchNotified(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	note := "marked patch notification as seen"
	if err := h.queries.LogModAudit(r.Context(), db.LogModAuditParams{
		ActionType: "patch_mark_seen",
		TargetType: "patch",
		TargetID:   id,
		ModUserID:  claims.UserID,
		Note:       &note,
	}); err != nil {
		slog.Warn("mod audit log failed", "action", "patch_mark_seen", "patch_id", id, "mod_id", claims.UserID, "err", err)
	}

	// Queue related active markets for resolution.
	related, err := h.queries.ListActiveMarketsForPatch(r.Context(), patch.PatchVersion)
	if err != nil {
		slog.Warn("patch mark-seen: failed to list related markets", "patch_id", id, "err", err)
	}
	queued := 0
	for _, m := range related {
		if err := h.queries.UpdateMarketStatus(r.Context(), db.UpdateMarketStatusParams{
			Status: "resolution_requested",
			ID:     m.ID,
		}); err != nil {
			slog.Warn("patch mark-seen: failed to queue market", "market_id", m.ID, "err", err)
			continue
		}
		patchNote := "Auto-queued: patch " + patch.PatchVersion + " marked as seen"
		if err := h.queries.UpsertResolutionRequestDetails(r.Context(), m.ID, claims.UserID, nil, &patchNote); err != nil {
			slog.Warn("patch mark-seen: failed to store resolution details", "market_id", m.ID, "err", err)
		}
		queued++
	}
	if queued > 0 {
		slog.Info("patch mark-seen: queued markets for resolution", "patch_id", id, "patch_version", patch.PatchVersion, "queued", queued)
	}

	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "queued_markets": queued})
}
