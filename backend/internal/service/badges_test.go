package service

import (
	"context"
	"testing"

	"github.com/lhw/astroledger/internal/database"
	"github.com/lhw/astroledger/internal/db"
)

func newBadgeServiceTestDB(t *testing.T) (*db.Queries, int64) {
	t.Helper()
	sqlDB, err := database.Open(context.Background(), t.TempDir()+"/badges-test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	queries := db.New(sqlDB)
	created, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ScidSub:     "test:badge-user",
		DisplayName: "Badge Pilot",
		Email:       "badge-pilot@test.example",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return queries, created.ID
}

func TestComputeLifetimeSpendUsesRecordedPricesAndFallbackCosts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	queries, userID := newBadgeServiceTestDB(t)
	service := NewBadgeService(queries)

	if err := queries.AwardBadgePurchased(ctx, userID, "roadmap_reader", 250, "6w"); err != nil {
		t.Fatalf("AwardBadgePurchased explicit price: %v", err)
	}
	if err := queries.AwardBadgePurchased(ctx, userID, "aurora_pilot", 0, "lti"); err != nil {
		t.Fatalf("AwardBadgePurchased fallback price: %v", err)
	}
	if err := queries.AwardBadgeIfNew(ctx, userID, "first_blood"); err != nil {
		t.Fatalf("AwardBadgeIfNew earned badge: %v", err)
	}

	spend, err := service.ComputeLifetimeSpend(ctx, userID)
	if err != nil {
		t.Fatalf("ComputeLifetimeSpend: %v", err)
	}

	if want := int64(350); spend != want {
		t.Fatalf("ComputeLifetimeSpend = %d, want %d", spend, want)
	}
}

func TestCheckAndAwardAdmiralRanksAwardsUnlockedRank(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	queries, userID := newBadgeServiceTestDB(t)
	service := NewBadgeService(queries)

	if err := queries.AwardBadgePurchased(ctx, userID, "space_whale", 600, "120w"); err != nil {
		t.Fatalf("AwardBadgePurchased: %v", err)
	}

	service.CheckAndAwardAdmiralRanks(ctx, userID)

	badges, err := queries.GetUserBadges(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserBadges: %v", err)
	}

	hasEnsign := false
	hasLieutenant := false
	for _, badge := range badges {
		if badge.BadgeKey == "ensign" {
			hasEnsign = true
		}
		if badge.BadgeKey == "lieutenant" {
			hasLieutenant = true
		}
	}

	if !hasEnsign {
		t.Fatal("expected ensign badge to be awarded")
	}
	if hasLieutenant {
		t.Fatal("did not expect lieutenant badge to be awarded at 600 spend")
	}
}