package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

// CommentHandler serves the comment endpoints.
type CommentHandler struct {
	svc *service.CommentService
}

// NewCommentHandler creates a CommentHandler.
func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// List handles GET /api/markets/{id}/comments.
// Public endpoint; shadow-hidden comments are only included for their author.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	marketID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || marketID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	viewerID := int64(0)
	if claims := middleware.GetClaims(r); claims != nil {
		viewerID = claims.UserID
	}

	comments, err := h.svc.ListComments(r.Context(), marketID, viewerID)
	if err != nil {
		slog.Error("list comments", "market_id", marketID, "err", err)
		respondError(w, http.StatusInternalServerError, "could not load comments")
		return
	}

	respondJSON(w, http.StatusOK, comments)
}

// Post handles POST /api/markets/{id}/comments (authentication required).
func (h *CommentHandler) Post(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	marketID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || marketID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid market id")
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	comment, err := h.svc.PostComment(r.Context(), service.CreateCommentInput{
		MarketID: marketID,
		UserID:   claims.UserID,
		Content:  body.Content,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, comment)
}

// Delete handles DELETE /api/comments/{id} (moderator only).
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	commentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || commentID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	if err := h.svc.DeleteComment(r.Context(), commentID); err != nil {
		slog.Error("delete comment", "comment_id", commentID, "err", err)
		respondError(w, http.StatusInternalServerError, "could not delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
