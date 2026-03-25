package service

import (
	"context"
	"log/slog"

	"github.com/lhw/scolymarket/internal/db"
)

// BadgeDefinition describes a badge and when it should be awarded.
type BadgeDefinition struct {
	Key         string
	Title       string
	Description string
}

// AllBadges lists every badge the system can award.
var AllBadges = []BadgeDefinition{
	{
		Key:         "first_blood",
		Title:       "First Blood",
		Description: "Made your first ever trade.",
	},
	{
		Key:         "market_maven",
		Title:       "Market Maven",
		Description: "Executed 50 or more trades.",
	},
	{
		Key:         "bug_prophet",
		Title:       "Bug Prophet",
		Description: "Correctly predicted the outcome of 5 or more markets.",
	},
	{
		Key:         "eternal_optimist",
		Title:       "Eternal Optimist",
		Description: "Bought YES shares in 10 or more markets.",
	},
	{
		Key:         "doomsayer",
		Title:       "Doomsayer",
		Description: "Bought NO shares in 10 or more markets.",
	},
}

// BadgeKeysMap provides O(1) lookup by key.
var BadgeKeysMap = func() map[string]BadgeDefinition {
	m := make(map[string]BadgeDefinition, len(AllBadges))
	for _, b := range AllBadges {
		m[b.Key] = b
	}
	return m
}()

// BadgeService evaluates and awards badges after state-changing events.
type BadgeService struct {
	queries *db.Queries
}

// NewBadgeService returns a BadgeService.
func NewBadgeService(queries *db.Queries) *BadgeService {
	return &BadgeService{queries: queries}
}

// CheckAndAward evaluates all badge conditions for userID and awards any that are
// newly met. Errors inside badge checks are logged and swallowed — badges should
// never block the primary operation (trade / resolution).
func (s *BadgeService) CheckAndAward(ctx context.Context, userID int64) {
	s.checkFirstBlood(ctx, userID)
	s.checkMarketMaven(ctx, userID)
	s.checkBugProphet(ctx, userID)
	s.checkEternalOptimist(ctx, userID)
	s.checkDoomsayer(ctx, userID)
}

func (s *BadgeService) award(ctx context.Context, userID int64, key string) {
	if err := s.queries.AwardBadgeIfNew(ctx, userID, key); err != nil {
		slog.Warn("badge award failed", "badge", key, "user_id", userID, "err", err)
		return
	}
	slog.Info("badge awarded", "badge", key, "user_id", userID)
}

func (s *BadgeService) checkFirstBlood(ctx context.Context, userID int64) {
	count, err := s.queries.CountUserTrades(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "badge", "first_blood", "err", err)
		return
	}
	if count >= 1 {
		s.award(ctx, userID, "first_blood")
	}
}

func (s *BadgeService) checkMarketMaven(ctx context.Context, userID int64) {
	count, err := s.queries.CountUserTrades(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "badge", "market_maven", "err", err)
		return
	}
	if count >= 50 {
		s.award(ctx, userID, "market_maven")
	}
}

func (s *BadgeService) checkBugProphet(ctx context.Context, userID int64) {
	count, err := s.queries.CountCorrectPredictions(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "badge", "bug_prophet", "err", err)
		return
	}
	if count >= 5 {
		s.award(ctx, userID, "bug_prophet")
	}
}

func (s *BadgeService) checkEternalOptimist(ctx context.Context, userID int64) {
	count, err := s.queries.CountMarketsWithYES(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "badge", "eternal_optimist", "err", err)
		return
	}
	if count >= 10 {
		s.award(ctx, userID, "eternal_optimist")
	}
}

func (s *BadgeService) checkDoomsayer(ctx context.Context, userID int64) {
	count, err := s.queries.CountMarketsWithNO(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "badge", "doomsayer", "err", err)
		return
	}
	if count >= 10 {
		s.award(ctx, userID, "doomsayer")
	}
}
