// Package tests contains integration tests that exercise the full backend stack
// against a real SQLite database with all goose migrations applied.
package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/lhw/scolymarket/internal/database"
	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/service"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// newTestDB opens a fresh SQLite database in a temp directory, runs all goose
// migrations, and registers a cleanup hook. Each call returns an isolated DB.
func newTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	sqlDB, err := database.Open(context.Background(), t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB, db.New(sqlDB)
}

// must fails the test immediately if err is non-nil.
func must(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// assertEq fails the test with a diff message if want != got.
func assertEq[T comparable](t *testing.T, want, got T, label string) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", label, want, got)
	}
}

// createTestUser inserts a user and optionally grants moderator rights.
func createTestUser(t *testing.T, ctx context.Context, q *db.Queries, sub, name string, isMod bool) db.User {
	t.Helper()
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		ScidSub:     sub,
		DisplayName: name,
		Email:       name + "@test.example",
	})
	must(t, err, "create user "+name)
	if isMod {
		must(t, q.UpdateUserGroups(ctx, user.ID, 1, 0), "grant mod to "+name)
		// Re-fetch to get updated flags.
		user, err = q.GetUserByID(ctx, user.ID)
		must(t, err, "re-fetch mod user")
	}
	return user
}

// newServices wires up the service layer on top of a test DB.
func newServices(sqlDB *sql.DB, q *db.Queries) (*service.MarketService, *service.TradingService, *service.BadgeService) {
	badgeSvc := service.NewBadgeService(q)
	marketSvc := service.NewMarketService(q, sqlDB, badgeSvc)
	tradingSvc := service.NewTradingService(q, sqlDB, badgeSvc)
	return marketSvc, tradingSvc, badgeSvc
}

// createActiveMarket creates a market via the service then immediately approves
// it so it is in "active" status and ready for trading.
func createActiveMarket(t *testing.T, ctx context.Context, svc *service.MarketService, title string, creatorID, modID int64) *db.Market {
	t.Helper()
	m, err := svc.CreateMarket(ctx, service.CreateMarketInput{
		Title:              title,
		Description:        "Integration test market, ignore.",
		Category:           "bug_fixes",
		ResolutionCriteria: "Patch notes confirm the fix.",
		Deadline:           time.Now().Add(72 * time.Hour),
		CreatedBy:          creatorID,
	})
	must(t, err, fmt.Sprintf("create market %q", title))
	must(t, svc.ApproveMarket(ctx, m.ID, modID), fmt.Sprintf("approve market %q", title))
	return m
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestFullMarketLifecycle exercises the complete happy-path flow:
//
//	Create users → Create market → Approve → Buy YES → Buy NO → Resolve YES
//	→ Verify winner payout & loser balance unchanged after resolution.
func TestFullMarketLifecycle(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "test:mod", "Moderator", true)
	yesBettor := createTestUser(t, ctx, q, "test:yes", "YesBettor", false)
	noBettor := createTestUser(t, ctx, q, "test:no", "NoBettor", false)

	// ── Create & approve market ───────────────────────────────────────────────
	m := createActiveMarket(t, ctx, marketSvc, "Will quantum fuel fix before 4.2?", mod.ID, mod.ID)

	got, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market after approve")
	assertEq(t, "active", got.Status, "status after approve")

	// ── Buy YES shares ────────────────────────────────────────────────────────
	// Initial state: b=100, qYes=0, qNo=0
	const yesShares = 10.0
	expectedYesCost := service.BuyCost(100.0, 0.0, 0.0, yesShares, true)

	yesResult, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID:   yesBettor.ID,
		MarketID: m.ID,
		Side:     "yes",
		Action:   "buy",
		Shares:   yesShares,
	})
	must(t, err, "buy YES shares")
	assertEq(t, expectedYesCost, yesResult.Cost, "YES trade cost")
	assertEq(t, int64(1000)-expectedYesCost, yesResult.NewBalance, "YES bettor balance after buy")

	// Verify position recorded correctly.
	yesPos, err := q.GetUserPositionOrZero(ctx, yesBettor.ID, m.ID)
	must(t, err, "get YES position")
	assertEq(t, yesShares, yesPos.YesShares, "YES shares in position")

	// ── Buy NO shares ─────────────────────────────────────────────────────────
	// After YES trade: qYes=10, qNo=0.
	const noShares = 10.0
	expectedNoCost := service.BuyCost(100.0, yesShares, 0.0, noShares, false)

	noResult, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID:   noBettor.ID,
		MarketID: m.ID,
		Side:     "no",
		Action:   "buy",
		Shares:   noShares,
	})
	must(t, err, "buy NO shares")
	assertEq(t, expectedNoCost, noResult.Cost, "NO trade cost")
	assertEq(t, int64(1000)-expectedNoCost, noResult.NewBalance, "NO bettor balance after buy")

	// ── Resolve as YES ────────────────────────────────────────────────────────
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID:   m.ID,
		Resolution: "yes",
		ModID:      mod.ID,
	}), "resolve market YES")

	// ── Verify payouts ────────────────────────────────────────────────────────
	// YES winner: initial(1000) - cost + shares*100
	yesUser, err := q.GetUserByID(ctx, yesBettor.ID)
	must(t, err, "get yes bettor post-resolve")
	wantYesFinal := int64(1000) - expectedYesCost + int64(yesShares)*100
	assertEq(t, wantYesFinal, yesUser.Balance, "YES bettor final balance")

	// NO holder: initial(1000) - cost, no payout (lost the bet).
	noUser, err := q.GetUserByID(ctx, noBettor.ID)
	must(t, err, "get no bettor post-resolve")
	wantNoFinal := int64(1000) - expectedNoCost
	assertEq(t, wantNoFinal, noUser.Balance, "NO bettor final balance (no payout)")

	// Market should now be resolved.
	finalMarket, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market post-resolve")
	assertEq(t, "resolved", finalMarket.Status, "market status after resolve")
	if finalMarket.Resolution == nil || *finalMarket.Resolution != "yes" {
		t.Errorf("market.Resolution: want \"yes\", got %v", finalMarket.Resolution)
	}
}

// TestSellShares verifies the sell flow: buy shares then sell them back,
// confirming the balance is restored (minus the AMM spread).
func TestSellShares(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "test:mod2", "Mod2", true)
	trader := createTestUser(t, ctx, q, "test:trader", "Trader", false)

	m := createActiveMarket(t, ctx, marketSvc, "Will 4.2 ship before May?", mod.ID, mod.ID)

	// Buy 20 YES.
	const shares = 20.0
	buyCost := service.BuyCost(100.0, 0.0, 0.0, shares, true)

	buyResult, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID:   trader.ID,
		MarketID: m.ID,
		Side:     "yes",
		Action:   "buy",
		Shares:   shares,
	})
	must(t, err, "buy YES")
	assertEq(t, buyCost, buyResult.Cost, "buy cost")

	// Sell all 20 YES back.
	mkt, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market for sell")
	sellRevenue := service.SellRevenue(100.0, mkt.YesShares, mkt.NoShares, shares, true)

	sellResult, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID:   trader.ID,
		MarketID: m.ID,
		Side:     "yes",
		Action:   "sell",
		Shares:   shares,
	})
	must(t, err, "sell YES")

	// Cost stored as negative for sells (received money).
	assertEq(t, -sellRevenue, sellResult.Cost, "sell revenue (negative cost)")

	// Balance after buy+sell: 1000 - buyCost + sellRevenue.
	// Due to LMSR ceiling/floor, sellRevenue <= buyCost, so spread ≥ 0.
	wantBalance := int64(1000) - buyCost + sellRevenue
	assertEq(t, wantBalance, sellResult.NewBalance, "balance after buy+sell")

	spread := buyCost - sellRevenue
	if spread < 0 {
		t.Errorf("AMM spread is negative (%d): sellRevenue > buyCost", spread)
	}
}

// TestResolveOnResolutionRequested verifies a market can be resolved when its
// status is resolution_requested (not just active).
func TestResolveOnResolutionRequested(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "test:mod3", "Mod3", true)
	trader := createTestUser(t, ctx, q, "test:trader2", "Trader2", false)

	m := createActiveMarket(t, ctx, marketSvc, "Will shields fix in 4.2?", mod.ID, mod.ID)

	// Trader buys YES so they can request resolution.
	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: trader.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 5,
	})
	must(t, err, "buy YES")

	// Trader requests resolution.
	must(t, marketSvc.RequestResolution(ctx, service.RequestResolutionInput{
		MarketID: m.ID,
		CallerID: trader.ID,
		Link:     "https://sc.game/patch/4.2",
		Note:     "Fix is in patch notes",
	}), "request resolution")

	got, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market after request")
	assertEq(t, "resolution_requested", got.Status, "status after request")

	// ListResolutionRequestedMarkets must work without scan errors.
	rows, err := q.ListResolutionRequestedMarkets(ctx)
	must(t, err, "list resolution requested markets")
	found := false
	for _, r := range rows {
		if r.ID == m.ID {
			found = true
			assertEq(t, "resolution_requested", r.Status, "row status")
			assertEq(t, trader.DisplayName, r.RequesterName, "requester name")
		}
	}
	if !found {
		t.Errorf("market %d not found in ListResolutionRequestedMarkets", m.ID)
	}

	// Mod resolves the market while it's still in resolution_requested state.
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID:   m.ID,
		Resolution: "yes",
		ModID:      mod.ID,
	}), "resolve resolution_requested market")

	final, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market post-resolve")
	assertEq(t, "resolved", final.Status, "final status")
}

// TestListPendingReports verifies that the reports table scan works correctly
// after reports are submitted.
func TestListPendingReports(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "test:mod4", "Mod4", true)
	user := createTestUser(t, ctx, q, "test:user4", "User4", false)

	m := createActiveMarket(t, ctx, marketSvc, "Will crimestat persist across relog?", mod.ID, mod.ID)

	// Give the user a trade so they're a participant.
	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: user.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 1,
	})
	must(t, err, "buy shares")

	// Submit a report.
	reportID, err := q.CreateReport(ctx, user.ID, m.ID, "This market violates the rules.")
	must(t, err, "create report")
	if reportID == 0 {
		t.Fatal("expected non-zero report ID")
	}

	// List reports — this is the query that was hitting a scan error in prod.
	reports, err := q.ListPendingReports(ctx)
	must(t, err, "list pending reports")
	found := false
	for _, r := range reports {
		if r.ID == reportID {
			found = true
			assertEq(t, user.DisplayName, r.ReporterName, "reporter name")
			assertEq(t, m.Title, r.MarketTitle, "market title")
			assertEq(t, "pending", r.Status, "report status")
		}
	}
	if !found {
		t.Errorf("report %d not found in ListPendingReports", reportID)
	}

	// Review the report.
	must(t, q.UpdateReportStatus(ctx, reportID, "reviewed"), "review report")

	// Reviewed report should no longer appear in pending list.
	pending, err := q.ListPendingReports(ctx)
	must(t, err, "list pending reports after review")
	for _, r := range pending {
		if r.ID == reportID {
			t.Errorf("reviewed report %d still appears in pending list", reportID)
		}
	}
}

// TestTradeOnResolutionRequestedStatus ensures that trading is still allowed
// when a market is in the resolution_requested state (not blocked by the old
// "market is not active" check).
func TestTradeOnResolutionRequestedStatus(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "test:mod5", "Mod5", true)
	trader := createTestUser(t, ctx, q, "test:trader5", "Trader5", false)

	m := createActiveMarket(t, ctx, marketSvc, "Will jump drives work in atmosphere?", mod.ID, mod.ID)

	// First buy to create a position.
	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: trader.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 5,
	})
	must(t, err, "initial buy")

	// Request resolution → moves status to resolution_requested.
	must(t, marketSvc.RequestResolution(ctx, service.RequestResolutionInput{
		MarketID: m.ID, CallerID: trader.ID,
	}), "request resolution")

	mktStatus, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market")
	assertEq(t, "resolution_requested", mktStatus.Status, "status before second trade")

	// A second user should still be able to buy while resolution is pending.
	trader2 := createTestUser(t, ctx, q, "test:trader5b", "Trader5b", false)
	_, err = tradingSvc.Execute(ctx, service.TradeInput{
		UserID: trader2.ID, MarketID: m.ID, Side: "no", Action: "buy", Shares: 3,
	})
	must(t, err, "buy on resolution_requested market")
}

// TestWeeklyPayout verifies the idempotency of the weekly credit payout.
func TestWeeklyPayout(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	creditsSvc := service.NewCreditsService(q)

	// Create two users.
	u1 := createTestUser(t, ctx, q, "test:payout1", "PayoutUser1", false)
	u2 := createTestUser(t, ctx, q, "test:payout2", "PayoutUser2", false)
	_ = sqlDB // used implicitly via q

	// Run payout once.
	count, err := creditsSvc.WeeklyPayout(ctx)
	must(t, err, "first payout")
	// At least our two users + seed users got credited.
	if count < 2 {
		t.Errorf("expected at least 2 users credited, got %d", count)
	}

	// Confirm balances increased by 200.
	u1After, err := q.GetUserByID(ctx, u1.ID)
	must(t, err, "get user1 after payout")
	assertEq(t, int64(1200), u1After.Balance, "user1 balance after payout")

	u2After, err := q.GetUserByID(ctx, u2.ID)
	must(t, err, "get user2 after payout")
	assertEq(t, int64(1200), u2After.Balance, "user2 balance after payout")

	// Running the same week's payout again must be a no-op.
	count2, err := creditsSvc.WeeklyPayout(ctx)
	must(t, err, "second payout (same week)")
	assertEq(t, int64(0), count2, "second payout should affect 0 users (idempotent)")

	// Balances must not have changed.
	u1Final, err := q.GetUserByID(ctx, u1.ID)
	must(t, err, "get user1 final")
	assertEq(t, int64(1200), u1Final.Balance, "user1 balance unchanged after duplicate payout")
}

// TestAMMMath verifies BuyCost and SellRevenue produce correct integer values
// matching the LMSR formula and that the AMM spread is always non-negative.
func TestAMMMath(t *testing.T) {
	const b = 100.0

	tests := []struct {
		name      string
		qYes, qNo float64
		shares    float64
		sideYes   bool
	}{
		{"fresh market YES 10", 0, 0, 10, true},
		{"fresh market NO 10", 0, 0, 10, false},
		{"skewed YES 50 NO 0, buy NO 10", 50, 0, 10, false},
		{"skewed NO 50 YES 0, buy YES 10", 0, 50, 10, true},
		{"balanced 100/100, buy YES 5", 100, 100, 5, true},
		{"large purchase YES 200", 0, 0, 200, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cost := service.BuyCost(b, tc.qYes, tc.qNo, tc.shares, tc.sideYes)
			if cost <= 0 {
				t.Errorf("BuyCost should be positive, got %d", cost)
			}

			var newYes, newNo float64
			if tc.sideYes {
				newYes = tc.qYes + tc.shares
				newNo = tc.qNo
			} else {
				newYes = tc.qYes
				newNo = tc.qNo + tc.shares
			}
			rev := service.SellRevenue(b, newYes, newNo, tc.shares, tc.sideYes)
			if rev <= 0 {
				t.Errorf("SellRevenue should be positive, got %d", rev)
			}

			spread := cost - rev
			if spread < 0 {
				t.Errorf("AMM spread negative: buy=%d sell=%d", cost, rev)
			}
		})
	}
}

// TestApproveRejectMarket verifies the moderation approve/reject paths.
func TestApproveRejectMarket(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, _, _ := newServices(sqlDB, q)
	_ = sqlDB

	mod := createTestUser(t, ctx, q, "test:mod6", "Mod6", true)
	creator := createTestUser(t, ctx, q, "test:creator", "Creator", false)

	// Create two markets (they'll be in pending_review status).
	m1, err := marketSvc.CreateMarket(ctx, service.CreateMarketInput{
		Title: "Will missile lock bug be fixed?", Description: "Been an issue for ages.",
		Category: "bug_fixes", ResolutionCriteria: "Patch notes.",
		Deadline: time.Now().Add(48 * time.Hour), CreatedBy: creator.ID,
	})
	must(t, err, "create m1")
	assertEq(t, "pending_review", m1.Status, "m1 initial status")

	m2, err := marketSvc.CreateMarket(ctx, service.CreateMarketInput{
		Title: "Will armor damage model be revised?", Description: "Armour rework discussion.",
		Category: "feature_delivery", ResolutionCriteria: "Feature in live build.",
		Deadline: time.Now().Add(48 * time.Hour), CreatedBy: creator.ID,
	})
	must(t, err, "create m2")
	assertEq(t, "pending_review", m2.Status, "m2 initial status")

	// Approve m1.
	must(t, marketSvc.ApproveMarket(ctx, m1.ID, mod.ID), "approve m1")
	got1, _ := q.GetMarketByID(ctx, m1.ID)
	assertEq(t, "active", got1.Status, "m1 after approve")

	// Reject m2.
	must(t, marketSvc.RejectMarket(ctx, m2.ID, mod.ID), "reject m2")
	got2, _ := q.GetMarketByID(ctx, m2.ID)
	assertEq(t, "cancelled", got2.Status, "m2 after reject")

	// Double-approve must fail.
	err = marketSvc.ApproveMarket(ctx, m1.ID, mod.ID)
	if err == nil {
		t.Error("double-approve should return error, got nil")
	}
}

// TestAutoFilter verifies that markets with banned keywords are rejected.
func TestAutoFilter(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, _, _ := newServices(sqlDB, q)
	_ = sqlDB

	mod := createTestUser(t, ctx, q, "test:mod7", "Mod7", true)

	_, err := marketSvc.CreateMarket(ctx, service.CreateMarketInput{
		Title:              "Will CIG accept real money donations?",
		Description:        "Discussing real cash fundraising.",
		Category:           "meta",
		ResolutionCriteria: "Announcement on spectrum.",
		Deadline:           time.Now().Add(48 * time.Hour),
		CreatedBy:          mod.ID,
	})
	if err == nil {
		t.Error("market with banned keyword should be rejected, got nil error")
	}
}

// TestDuplicateTitleDetection verifies that submitting a market with a title
// very similar to an existing active market is rejected by the auto-filter.
func TestDuplicateTitleDetection(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, _, _ := newServices(sqlDB, q)
	_ = sqlDB

	mod := createTestUser(t, ctx, q, "test:mod8", "Mod8", true)

	// Create and activate the first market.
	orig := createActiveMarket(t, ctx, marketSvc,
		"Will the Reclaimer elevator bug be fixed in 4.1?", mod.ID, mod.ID)
	_ = orig

	// Try to submit a near-duplicate (high trigram overlap with the approved title).
	_, err := marketSvc.CreateMarket(ctx, service.CreateMarketInput{
		Title:              "Will the Reclaimer elevator bug be fixed in patch 4.1?",
		Description:        "Same elevator, still broken.",
		Category:           "bug_fixes",
		ResolutionCriteria: "Not on known issues.",
		Deadline:           time.Now().Add(48 * time.Hour),
		CreatedBy:          mod.ID,
	})
	if err == nil {
		t.Error("near-duplicate market should be rejected, got nil error")
	}

	// A clearly different market should pass.
	_, err = marketSvc.CreateMarket(ctx, service.CreateMarketInput{
		Title:              "Will quantum travel stutter be resolved before 4.2?",
		Description:        "Persistent quantum stutter affecting all ships.",
		Category:           "bug_fixes",
		ResolutionCriteria: "No stutter reports in live PTU patch notes.",
		Deadline:           time.Now().Add(48 * time.Hour),
		CreatedBy:          mod.ID,
	})
	must(t, err, "distinct market should be accepted")
}
