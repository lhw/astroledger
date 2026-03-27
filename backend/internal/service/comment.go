package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lhw/scolymarket/internal/db"
)

// Comment is the service-layer comment type returned to API callers.
type Comment struct {
	ID         int64  `json:"id"`
	MarketID   int64  `json:"market_id"`
	UserID     int64  `json:"user_id"`
	AuthorName string `json:"author_name"`
	// AuthorTopBadge is the badge_key of the user's highest-tier badge, or empty.
	AuthorTopBadge string `json:"author_top_badge,omitempty"`
	Content        string `json:"content"`
	// Hidden is true when this comment was shadow-hidden by the abuse detector
	// AND the viewer is the author (the author can always see their own comment).
	// Other viewers never receive hidden=true — they simply don't see the comment.
	Hidden    bool      `json:"hidden"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateCommentInput is the input for PostComment.
type CreateCommentInput struct {
	MarketID int64
	UserID   int64
	Content  string
}

// CommentService handles posting and retrieving market comments.
type CommentService struct {
	q   *db.Queries
	mod *ModerationClient
}

// NewCommentService creates a CommentService.
func NewCommentService(q *db.Queries, mod *ModerationClient) *CommentService {
	return &CommentService{q: q, mod: mod}
}

// PostComment validates, moderates, and stores a new comment.
// It never fails due to a moderation error — if the Perspective API is
// unavailable, the comment passes through unfiltered (and unscored).
func (s *CommentService) PostComment(ctx context.Context, inp CreateCommentInput) (*Comment, error) {
	content := strings.TrimSpace(inp.Content)
	if len([]rune(content)) < 1 || len(content) > 1000 {
		return nil, fmt.Errorf("comment must be between 1 and 1000 characters")
	}

	// Validate that the market exists and accepts comments.
	market, err := s.q.GetMarketByID(ctx, inp.MarketID)
	if err != nil {
		return nil, fmt.Errorf("market not found")
	}
	switch market.Status {
	case "active", "resolved", "resolution_requested":
		// OK
	default:
		return nil, fmt.Errorf("comments are not open for this market")
	}

	// Run abuse detection. A failure is non-fatal — log and allow through.
	modResult, modErr := s.mod.Moderate(ctx, content)
	if modErr != nil {
		slog.Warn("comment moderation error (passing comment through)", "err", modErr)
		modResult = &ModerationResult{}
	}

	hidden := int64(0)
	if modResult.Hide {
		hidden = 1
		slog.Info("comment shadow-hidden by abuse detector", "market_id", inp.MarketID, "user_id", inp.UserID)
	}

	// Encode moderation scores for storage.
	var toxScore *float64
	var flagsJSON *string
	if len(modResult.Scores) > 0 {
		if tox, ok := modResult.Scores["TOXICITY"]; ok {
			toxScore = &tox
		}
		b, _ := json.Marshal(modResult.Scores)
		s := string(b)
		flagsJSON = &s
	}

	row, err := s.q.CreateComment(ctx, db.CreateCommentParams{
		MarketID:        inp.MarketID,
		UserID:          inp.UserID,
		Content:         content,
		Hidden:          hidden,
		ToxicityScore:   toxScore,
		ModerationFlags: flagsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	return &Comment{
		ID:         row.ID,
		MarketID:   row.MarketID,
		UserID:     row.UserID,
		AuthorName: "", // populated by caller from session claims if needed
		Content:    row.Content,
		Hidden:     modResult.Hide,
		CreatedAt:  row.CreatedAt,
	}, nil
}

// ListComments returns visible comments for a market plus any shadow-hidden
// comments belonging to the viewer.
// Pass viewerID=0 for anonymous requests.
func (s *CommentService) ListComments(ctx context.Context, marketID, viewerID int64) ([]*Comment, error) {
	rows, err := s.q.GetCommentsByMarket(ctx, db.GetCommentsByMarketParams{
		MarketID: marketID,
		UserID:   viewerID,
	})
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	// Collect unique author IDs so we can batch-fetch their top badge.
	authorIDs := make([]int64, 0, len(rows))
	seen := map[int64]bool{}
	for _, r := range rows {
		if !seen[r.UserID] {
			seen[r.UserID] = true
			authorIDs = append(authorIDs, r.UserID)
		}
	}

	topBadge, err := s.q.GetUsersActiveBadges(ctx, authorIDs)
	if err != nil {
		topBadge = map[int64]string{} // non-fatal: render without badges
	}

	out := make([]*Comment, 0, len(rows))
	for _, r := range rows {
		dbHidden := r.Hidden != 0
		isOwnHidden := dbHidden && r.UserID == viewerID

		out = append(out, &Comment{
			ID:             r.ID,
			MarketID:       r.MarketID,
			UserID:         r.UserID,
			AuthorName:     r.AuthorName,
			AuthorTopBadge: topBadge[r.UserID],
			Content:        r.Content,
			// Only expose hidden=true to the comment's own author.
			Hidden:    isOwnHidden,
			CreatedAt: r.CreatedAt,
		})
	}

	return out, nil
}

// DeleteComment hard-deletes a comment (moderator action).
func (s *CommentService) DeleteComment(ctx context.Context, commentID, modUserID int64) error {
	if err := s.q.DeleteComment(ctx, commentID); err != nil {
		return fmt.Errorf("delete comment %d: %w", commentID, err)
	}
	note := "deleted comment"
	if err := s.q.LogModAudit(ctx, db.LogModAuditParams{
		ActionType: "delete_comment",
		TargetType: "comment",
		TargetID:   commentID,
		ModUserID:  modUserID,
		Note:       &note,
	}); err != nil {
		slog.Warn("mod audit log failed", "action", "delete_comment", "comment_id", commentID, "mod_id", modUserID, "err", err)
	}
	return nil
}
