package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/lhw/scolymarket/internal/db"
)

// BadgeDefinition describes a badge and when it should be awarded.
type BadgeDefinition struct {
	Key         string
	Title       string
	Description string
	// Tier controls visual weight: 1=common, 2=uncommon, 3=rare, 4=epic, 5=legendary.
	Tier int
	// Cost is the bUEC price to purchase from the FOMO Store. 0 = not purchasable (earned only).
	Cost int64
	// Purchasable marks badges that can be bought rather than only earned.
	Purchasable bool
	// Stock is the maximum number of users that can own this badge. nil = unlimited.
	Stock *int
	// AvailableUntil, if set, is the deadline after which the badge can no longer be purchased.
	AvailableUntil *time.Time
	// SpendThreshold, if > 0, marks this as an Admiral Rank badge: it is awarded
	// automatically when the user's cumulative FOMO store spend reaches this amount.
	SpendThreshold int64
}

// intPtr is a convenience helper for badge stock values.
func intPtr(n int) *int { return &n }

// timePtr is a convenience helper for badge expiry values.
func timePtr(t time.Time) *time.Time { return &t }

// AllBadges lists every badge the system can award or sell.
var AllBadges = []BadgeDefinition{
	// ── Earned badges (Cost=0, not in store) ─────────────────────────────────
	// Trade-count milestones
	{
		Key:         "first_blood",
		Title:       "First Blood",
		Description: "Made your first ever trade. The journey begins.",
		Tier:        1,
	},
	{
		Key:         "quick_shot",
		Title:       "Quick Shot",
		Description: "10 trades in. Getting the hang of this.",
		Tier:        1,
	},
	{
		Key:         "market_maven",
		Title:       "Market Maven",
		Description: "50 trades. Your mouse button is suffering.",
		Tier:        2,
	},
	{
		Key:         "seasoned_trader",
		Title:       "Seasoned Trader",
		Description: "100 trades. At this point it's a lifestyle.",
		Tier:        2,
	},
	{
		Key:         "market_obsessed",
		Title:       "Market Obsessed",
		Description: "250 trades. We're not judging. We're in awe.",
		Tier:        3,
	},
	{
		Key:         "galaxy_brained",
		Title:       "Galaxy Brained",
		Description: "500 trades. When does this become a job?",
		Tier:        4,
	},
	// Prediction milestones
	{
		Key:         "bug_prophet",
		Title:       "Bug Prophet",
		Description: "Correctly predicted 5 markets. The 'verse bends to your will.",
		Tier:        3,
	},
	{
		Key:         "skeptic",
		Title:       "Skeptic",
		Description: "Correctly predicted 10 markets. You've seen this before.",
		Tier:        2,
	},
	{
		Key:         "oracle",
		Title:       "Oracle",
		Description: "25 correct predictions. Are you actually clairvoyant?",
		Tier:        4,
	},
	// Participation breadth
	{
		Key:         "eternal_optimist",
		Title:       "Eternal Optimist",
		Description: "Active in 10 or more markets. Hope springs eternal.",
		Tier:        2,
	},
	{
		Key:         "doomsayer",
		Title:       "Doomsayer",
		Description: "Predicted against the crowd in 10 or more markets.",
		Tier:        2,
	},
	{
		Key:         "portfolio_manager",
		Title:       "Portfolio Manager",
		Description: "Positions in 25 markets. Hedged everywhere, profitable nowhere.",
		Tier:        2,
	},
	{
		Key:         "universe_citizen",
		Title:       "Universe Citizen",
		Description: "Positions in 50 markets. You have no free time and we respect that.",
		Tier:        3,
	},
	// Market creation milestones
	{
		Key:         "market_founder",
		Title:       "Market Founder",
		Description: "Posted a market that made it live. It was probably about bugs.",
		Tier:        1,
	},
	{
		Key:         "serial_founder",
		Title:       "Serial Founder",
		Description: "5 live markets. You're practically running the 'verse.",
		Tier:        2,
	},
	// ── FOMO Store badges (Purchasable=true) ─────────────────────────────────
	// ── General availability (unlimited stock, always purchasable) ────────────
	// ── General availability (unlimited stock, always purchasable) ────────────
	{
		Key:         "citizen_backer",
		Title:       "Citizen Backer",
		Description: "A true believer. Been here since before it was cool.",
		Tier:        1, Cost: 50, Purchasable: true,
	},
	{
		Key:         "professional_bug_finder",
		Title:       "Professional Bug Finder",
		Description: "It's not a bug, it's a stretch goal.",
		Tier:        1, Cost: 150, Purchasable: true,
	},
	{
		Key:         "aurora_pilot",
		Title:       "Aurora Pilot",
		Description: "Started small, dreamed big. The Aurora is a classic.",
		Tier:        1, Cost: 100, Purchasable: true,
	},
	{
		Key:         "roadmap_reader",
		Title:       "Roadmap Reader",
		Description: "Holds 47 open tabs of schedule promises. Refreshes hourly.",
		Tier:        1, Cost: 200, Purchasable: true,
	},
	{
		Key:         "warp_speed",
		Title:       "Warp Speed",
		Description: "Will jump to conclusions faster than quantum drive.",
		Tier:        1, Cost: 125, Purchasable: true,
	},
	{
		Key:         "mostly_backer",
		Title:       "Mostly Backer",
		Description: "In since 2012. Still waiting. Mostly fine about it.",
		Tier:        2, Cost: 300, Purchasable: true,
	},
	{
		Key:         "hangar_queen",
		Title:       "Hangar Queen",
		Description: "The ships sit in the hangar. The pilots sit at the keyboard. Aspirational.",
		Tier:        2, Cost: 250, Purchasable: true,
	},
	{
		Key:         "tech_preview_survivor",
		Title:       "Tech Preview Survivor",
		Description: "I've seen things you wouldn't believe. Then they wiped the servers.",
		Tier:        2, Cost: 400, Purchasable: true,
	},
	{
		Key:         "star_gazer",
		Title:       "Star Gazer",
		Description: "Watched the CitizenCon stream live every year. No regrets.",
		Tier:        2, Cost: 350, Purchasable: true,
	},
	{
		Key:         "alpha_tester",
		Title:       "Alpha Tester",
		Description: "Tested things that clearly weren't ready. Knew it. Did it anyway.",
		Tier:        2, Cost: 450, Purchasable: true,
	},
	{
		Key:         "space_whale",
		Title:       "Space Whale",
		Description: "Your wallet is canon-sized. The CIG shareholders send their regards.",
		Tier:        2, Cost: 500, Purchasable: true,
	},
	{
		Key:         "bugged_not_broken",
		Title:       "Bugged, Not Broken",
		Description: "It's a feature. Definitely a feature. A very intentional feature.",
		Tier:        2, Cost: 250, Purchasable: true,
		Stock: intPtr(50),
	},
	{
		Key:         "verse_veteran",
		Title:       "'Verse Veteran",
		Description: "A seasoned survivor of the persistent universe. Emphasis on survivor.",
		Tier:        2, Cost: 500, Purchasable: true,
		AvailableUntil: timePtr(time.Date(2026, time.April, 30, 23, 59, 59, 0, time.UTC)),
	},
	{
		Key:         "persistent_citizen",
		Title:       "Persistent Universe Citizen",
		Description: "Was there for server meshing. Both times. Survived both wipes.",
		Tier:        3, Cost: 600, Purchasable: true,
	},
	{
		Key:         "org_leader",
		Title:       "Org Leader",
		Description: "Commands fleets. Herds cats. Usually the same thing.",
		Tier:        3, Cost: 1000, Purchasable: true,
		Stock: intPtr(25),
	},
	{
		Key:         "900i_enjoyer",
		Title:       "900i Enjoyer",
		Description: "Luxury is a lifestyle, not a budget.",
		Tier:        3, Cost: 1500, Purchasable: true,
		AvailableUntil: timePtr(time.Date(2026, time.April, 5, 23, 59, 59, 0, time.UTC)),
	},
	// ── Hull Limited (scarce — fixed global stock) ────────────────────────────
	{
		Key:         "system_colonist",
		Title:       "System Colonist",
		Description: "Reserved a plot in a star system that may never ship. We admire the commitment.",
		Tier:        3, Cost: 2000, Purchasable: true,
		Stock: intPtr(15),
	},
	{
		Key:         "idris_captain",
		Title:       "Idris Captain",
		Description: "Full sovereignty. Even in a game about bugs.",
		Tier:        4, Cost: 3000, Purchasable: true,
		Stock: intPtr(10),
	},
	{
		Key:         "backer_royalty",
		Title:       "Backer Royalty",
		Description: "The original true believers. You were there before it was ironic.",
		Tier:        4, Cost: 3500, Purchasable: true,
		Stock: intPtr(10),
	},
	{
		Key:         "fleet_commander_badge",
		Title:       "Fleet Commander",
		Description: "Controls more ships than actual crew. The spreadsheet is enormous.",
		Tier:        4, Cost: 4500, Purchasable: true,
		Stock: intPtr(8),
	},
	{
		Key:         "golden_ticket",
		Title:       "Golden Ticket",
		Description: "The most prestigious flex in the 'verse. Pure gold.",
		Tier:        5, Cost: 5000, Purchasable: true,
		Stock:          intPtr(3),
		AvailableUntil: timePtr(time.Date(2026, time.April, 30, 23, 59, 59, 0, time.UTC)),
	},
	{
		Key:         "unobtainium",
		Title:       "Unobtainium Tier",
		Description: "You spent HOW much on fake badges? We are genuinely speechless.",
		Tier:        5, Cost: 7500, Purchasable: true,
		Stock: intPtr(5),
	},
	// ── Rotating (limited-time window — existing owners keep theirs) ──────────
	{
		Key:         "alpha_optimist",
		Title:       "Alpha Optimist",
		Description: "Believed the patch notes. Every time. Bless your heart.",
		Tier:        2, Cost: 750, Purchasable: true,
		AvailableUntil: timePtr(time.Date(2026, time.April, 30, 23, 59, 59, 0, time.UTC)),
	},
	{
		Key:         "q4_enjoyer",
		Title:       "Q4 Enjoyer",
		Description: "Adjusted expectations quarterly since 2015 and feels great about it.",
		Tier:        2, Cost: 600, Purchasable: true,
		AvailableUntil: timePtr(time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC)),
	},
	{
		Key:         "citizencon_pilgrim",
		Title:       "CitizenCon Pilgrim",
		Description: "Was there. Watched the stream. Followed every thread. No regrets.",
		Tier:        3, Cost: 1200, Purchasable: true,
		AvailableUntil: timePtr(time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC)),
	},
	// ── Admiral Rank badges (SpendThreshold > 0, auto-awarded by lifetime FOMO spend) ─
	{
		Key:            "ensign",
		Title:          "Ensign",
		Description:    "You spent real fake money on fake badges. Welcome to the fleet.",
		Tier:           1,
		SpendThreshold: 500,
	},
	{
		Key:            "lieutenant",
		Title:          "Lieutenant",
		Description:    "A promising career in poor financial decisions.",
		Tier:           2,
		SpendThreshold: 5000,
	},
	{
		Key:            "commander",
		Title:          "Commander",
		Description:    "A distinguished patron of pointless digital flexing.",
		Tier:           3,
		SpendThreshold: 25000,
	},
	{
		Key:            "captain",
		Title:          "Captain",
		Description:    "You command respect. And an alarming spending habit.",
		Tier:           4,
		SpendThreshold: 100000,
	},
	{
		Key:            "coin_admiral",
		Title:          "Coin Admiral",
		Description:    "One million fake credits. Spent. On badges. We have no words.",
		Tier:           5,
		SpendThreshold: 1000000,
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

// AdmiralRankBadgesCache caches the admiral rank badges (those with SpendThreshold > 0).
var AdmiralRankBadgesCache = func() []BadgeDefinition {
	var out []BadgeDefinition
	for _, b := range AllBadges {
		if b.SpendThreshold > 0 {
			out = append(out, b)
		}
	}
	return out
}()

// AdmiralRankBadges returns the admiral rank badges in threshold order.
func AdmiralRankBadges() []BadgeDefinition {
	return AdmiralRankBadgesCache
}

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
	s.checkTradeMilestones(ctx, userID)
	s.checkPredictionMilestones(ctx, userID)
	s.checkMarketBreadth(ctx, userID)
	s.checkMarketCreation(ctx, userID)
	s.checkDoomsayer(ctx, userID)
}

// ComputeLifetimeSpend returns the total bUEC the user has spent in the FOMO store.
// It uses the purchase_price recorded on each user_badge row (set when buying via a release).
// Badges acquired before the release system (purchase_price=0) fall back to the static
// Cost field on the BadgeDefinition so old spends still count toward admiral ranks.
func (s *BadgeService) ComputeLifetimeSpend(ctx context.Context, userID int64) (int64, error) {
	prices, err := s.queries.GetUserBadgePurchasePrices(ctx, userID)
	if err != nil {
		return 0, err
	}
	var total int64
	for key, price := range prices {
		if price > 0 {
			total += price
		} else if def, ok := BadgeKeysMap[key]; ok && def.Purchasable {
			// Fallback for badges purchased before the release system was introduced.
			total += def.Cost
		}
	}
	return total, nil
}

// CheckAndAwardAdmiralRanks awards any admiral rank badges unlocked by the user's
// current lifetime FOMO spend. Call this after every successful FOMO purchase.
func (s *BadgeService) CheckAndAwardAdmiralRanks(ctx context.Context, userID int64) {
	spend, err := s.ComputeLifetimeSpend(ctx, userID)
	if err != nil {
		slog.Warn("admiral rank check failed", "user_id", userID, "err", err)
		return
	}
	for _, b := range AllBadges {
		if b.SpendThreshold > 0 && spend >= b.SpendThreshold {
			s.award(ctx, userID, b.Key)
		}
	}
}

func (s *BadgeService) award(ctx context.Context, userID int64, key string) {
	if err := s.queries.AwardBadgeIfNew(ctx, userID, key); err != nil {
		slog.Warn("badge award failed", "badge", key, "user_id", userID, "err", err)
		return
	}
	slog.Info("badge awarded", "badge", key, "user_id", userID)
}

func (s *BadgeService) checkTradeMilestones(ctx context.Context, userID int64) {
	count, err := s.queries.CountUserTrades(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "group", "trade_milestones", "err", err)
		return
	}
	if count >= 1 {
		s.award(ctx, userID, "first_blood")
	}
	if count >= 10 {
		s.award(ctx, userID, "quick_shot")
	}
	if count >= 50 {
		s.award(ctx, userID, "market_maven")
	}
	if count >= 100 {
		s.award(ctx, userID, "seasoned_trader")
	}
	if count >= 250 {
		s.award(ctx, userID, "market_obsessed")
	}
	if count >= 500 {
		s.award(ctx, userID, "galaxy_brained")
	}
}

func (s *BadgeService) checkPredictionMilestones(ctx context.Context, userID int64) {
	count, err := s.queries.CountCorrectPredictions(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "group", "prediction_milestones", "err", err)
		return
	}
	if count >= 5 {
		s.award(ctx, userID, "bug_prophet")
	}
	if count >= 10 {
		s.award(ctx, userID, "skeptic")
	}
	if count >= 25 {
		s.award(ctx, userID, "oracle")
	}
}

func (s *BadgeService) checkMarketBreadth(ctx context.Context, userID int64) {
	count, err := s.queries.CountMarketsWithYES(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "group", "market_breadth", "err", err)
		return
	}
	if count >= 10 {
		s.award(ctx, userID, "eternal_optimist")
	}
	if count >= 25 {
		s.award(ctx, userID, "portfolio_manager")
	}
	if count >= 50 {
		s.award(ctx, userID, "universe_citizen")
	}
}

func (s *BadgeService) checkMarketCreation(ctx context.Context, userID int64) {
	count, err := s.queries.CountLiveMarketsCreatedBy(ctx, userID)
	if err != nil {
		slog.Warn("badge check failed", "group", "market_creation", "err", err)
		return
	}
	if count >= 1 {
		s.award(ctx, userID, "market_founder")
	}
	if count >= 5 {
		s.award(ctx, userID, "serial_founder")
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
