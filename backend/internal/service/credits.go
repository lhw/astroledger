package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lhw/scolymarket/internal/db"
)

// CreditsService manages ScollyBucks™ balances.
type CreditsService struct {
	queries *db.Queries
}

// NewCreditsService creates a CreditsService.
func NewCreditsService(queries *db.Queries) *CreditsService {
	return &CreditsService{queries: queries}
}

// WeeklyPayout gives every user 200 bUEC. It is idempotent: calling it multiple
// times in the same ISO week does nothing after the first successful run.
// Returns the number of users paid and any error.
func (s *CreditsService) WeeklyPayout(ctx context.Context) (int64, error) {
	weekKey := isoWeekKey(time.Now())

	already, err := s.queries.WeeklyPayoutAlreadyRan(ctx, weekKey)
	if err != nil {
		return 0, fmt.Errorf("check payout log: %w", err)
	}
	if already {
		slog.Debug("weekly payout already ran", "week", weekKey)
		return 0, nil
	}

	count, err := s.queries.RunWeeklyPayout(ctx, weekKey)
	if err != nil {
		return 0, fmt.Errorf("run weekly payout: %w", err)
	}
	slog.Info("weekly payout complete", "week", weekKey, "users_paid", count)
	return count, nil
}

// isoWeekKey returns a string like "2026-12" (year-weeknumber) for the given time.
func isoWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%04d-%02d", year, week)
}
