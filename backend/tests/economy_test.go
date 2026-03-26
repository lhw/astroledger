package tests

// Economy integration tests — full API+DB flow.
//
// Verifies that after trading and resolution the total bUEC in the system only
// increases by at most the per-market LMSR subsidy:
//
//	b × ln(2) × payoutPerShare  ≈  6,931 bUEC   (b=100, payout=100)

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/service"
)

const (
	// payoutPerShareE matches the hardcoded constant in ResolveMarket.
	payoutPerShareE = 100
	// defaultLiqB is the liquidity param hardcoded in CreateMarket.
	defaultLiqB = 100.0
	// lmsrMaxLoss is the theoretical max platform subsidy per market.
	lmsrMaxLoss = defaultLiqB * math.Ln2 * payoutPerShareE // ≈ 6,931 bUEC
)

// topUpBalance adds extra bUEC to a user so they can make larger trades.
func topUpBalance(t *testing.T, ctx context.Context, q *db.Queries, userID, extra int64) {
	t.Helper()
	must(t, q.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: extra, ID: userID}), "top-up balance")
}

// sumBalances returns the total bUEC held across the given user IDs.
func sumBalances(t *testing.T, ctx context.Context, q *db.Queries, ids ...int64) int64 {
	t.Helper()
	total := int64(0)
	for _, id := range ids {
		u, err := q.GetUserByID(ctx, id)
		must(t, err, fmt.Sprintf("get user %d", id))
		total += u.Balance
	}
	return total
}

// assertBoundedGrowth fails if total bUEC grew beyond the LMSR per-market bound.
func assertBoundedGrowth(t *testing.T, before, after int64, numMarkets int, label string) {
	t.Helper()
	growth := after - before
	maxGrowth := int64(math.Ceil(lmsrMaxLoss)) * int64(numMarkets)
	t.Logf("[%s] total bUEC before=%d  after=%d  growth=%d  LMSR bound=%d",
		label, before, after, growth, maxGrowth)
	if growth > maxGrowth {
		t.Errorf(
			"[%s] economy bUEC grew by %d but LMSR bound is %d per market (%d markets).\n"+
				"Excess: %d bUEC created from nothing.\n"+
				"FIX: multiply BuyCost/SellRevenue output by %d.",
			label, growth, maxGrowth, numMarkets, growth-maxGrowth, payoutPerShareE,
		)
	}
}

// TestEconomy_SingleUser_AllYes_ResolvesYes: one user bets YES, wins.
func TestEconomy_SingleUser_AllYes_ResolvesYes(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod1", "Mod1", true)
	bettor := createTestUser(t, ctx, q, "eco:bettor1", "Bettor1", false)

	before := sumBalances(t, ctx, q, mod.ID, bettor.ID)
	m := createActiveMarket(t, ctx, marketSvc, "eco: single YES bettor", mod.ID, mod.ID)

	const shares = 10.0
	cost := service.BuyCost(defaultLiqB, 0, 0, shares, true)
	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: bettor.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: shares,
	})
	must(t, err, "buy YES")
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve YES")

	payout := shares * payoutPerShareE
	t.Logf("YES bettor: cost=%d, payout=%.0f, net=%.0f", cost, payout, payout-float64(cost))

	after := sumBalances(t, ctx, q, mod.ID, bettor.ID)
	assertBoundedGrowth(t, before, after, 1, "single_user_all_yes_resolves_yes")
}

// TestEconomy_SingleUser_AllYes_ResolvesNo: user bets YES, loses.
func TestEconomy_SingleUser_AllYes_ResolvesNo(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod2", "Mod2", true)
	bettor := createTestUser(t, ctx, q, "eco:bettor2", "Bettor2", false)

	before := sumBalances(t, ctx, q, mod.ID, bettor.ID)
	m := createActiveMarket(t, ctx, marketSvc, "eco: YES bettor loses", mod.ID, mod.ID)

	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: bettor.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 5,
	})
	must(t, err, "buy YES")
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "no", ModID: mod.ID,
	}), "resolve NO")

	after := sumBalances(t, ctx, q, mod.ID, bettor.ID)
	if after > before {
		t.Errorf("economy grew by %d when YES bettor lost: expected no new bUEC", after-before)
	}
	t.Logf("Economy shrunk by %d bUEC (platform net gain from losing bet)", before-after)
}

// TestEconomy_ThreeUsers_MixedBets_ResolveYes: alice(YES)+carol(YES)+bob(NO).
func TestEconomy_ThreeUsers_MixedBets_ResolveYes(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod3", "Mod3", true)
	alice := createTestUser(t, ctx, q, "eco:alice", "Alice", false)
	bob := createTestUser(t, ctx, q, "eco:bob", "Bob", false)
	carol := createTestUser(t, ctx, q, "eco:carol", "Carol", false)

	users := []int64{mod.ID, alice.ID, bob.ID, carol.ID}
	before := sumBalances(t, ctx, q, users...)
	m := createActiveMarket(t, ctx, marketSvc, "eco: 3 users mixed", mod.ID, mod.ID)

	buys := []struct {
		userID int64
		side   string
		shares float64
	}{
		{alice.ID, "yes", 5},
		{bob.ID, "no", 4},
		{carol.ID, "yes", 3},
	}

	qYes, qNo := 0.0, 0.0
	totalCollected := int64(0)
	for _, buy := range buys {
		sideYes := buy.side == "yes"
		cost := service.BuyCost(defaultLiqB, qYes, qNo, buy.shares, sideYes)
		res, err := tradingSvc.Execute(ctx, service.TradeInput{
			UserID: buy.userID, MarketID: m.ID, Side: buy.side,
			Action: "buy", Shares: buy.shares,
		})
		must(t, err, fmt.Sprintf("buy %s for user %d", buy.side, buy.userID))
		if res.Cost != cost {
			t.Errorf("trade cost mismatch: service=%d, manual=%d", res.Cost, cost)
		}
		totalCollected += cost
		if sideYes {
			qYes += buy.shares
		} else {
			qNo += buy.shares
		}
	}

	mkt, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market state")
	if math.Abs(mkt.YesShares-qYes) > 0.001 {
		t.Errorf("AMM yes_shares: DB=%.4f, expected=%.4f", mkt.YesShares, qYes)
	}
	if math.Abs(mkt.NoShares-qNo) > 0.001 {
		t.Errorf("AMM no_shares: DB=%.4f, expected=%.4f", mkt.NoShares, qNo)
	}

	totalYesPayout := int64(buys[0].shares+buys[2].shares) * payoutPerShareE

	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve YES")

	after := sumBalances(t, ctx, q, users...)
	assertBoundedGrowth(t, before, after, 1, "three_users_mixed_resolve_yes")
	t.Logf("Collected: %d bUEC, YES payout: %d bUEC, growth: %d", totalCollected, totalYesPayout, after-before)
}

// TestEconomy_ThreeUsers_MixedBets_ResolveNo: same setup, resolves NO.
func TestEconomy_ThreeUsers_MixedBets_ResolveNo(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod4", "Mod4", true)
	alice := createTestUser(t, ctx, q, "eco:alice4", "Alice4", false)
	bob := createTestUser(t, ctx, q, "eco:bob4", "Bob4", false)
	carol := createTestUser(t, ctx, q, "eco:carol4", "Carol4", false)

	users := []int64{mod.ID, alice.ID, bob.ID, carol.ID}
	before := sumBalances(t, ctx, q, users...)
	m := createActiveMarket(t, ctx, marketSvc, "eco: 3 users resolve NO", mod.ID, mod.ID)

	for _, buy := range []struct {
		id     int64
		side   string
		shares float64
	}{
		{alice.ID, "yes", 5},
		{bob.ID, "no", 4},
		{carol.ID, "yes", 3},
	} {
		_, err := tradingSvc.Execute(ctx, service.TradeInput{
			UserID: buy.id, MarketID: m.ID, Side: buy.side,
			Action: "buy", Shares: buy.shares,
		})
		must(t, err, fmt.Sprintf("buy %s", buy.side))
	}
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "no", ModID: mod.ID,
	}), "resolve NO")
	after := sumBalances(t, ctx, q, users...)
	assertBoundedGrowth(t, before, after, 1, "three_users_mixed_resolve_no")
}

// TestEconomy_HeavySingleBettor_TenKShares is the exact user-reported bug:
// "buy 10,000 YES shares at ~1 bUEC each, resolve YES -> million bUEC".
// With the fix, 10k shares cost ~993,069 bUEC, payout 1,000,000, profit ~6,931.
func TestEconomy_HeavySingleBettor_TenKShares(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod5", "Mod5", true)
	whale := createTestUser(t, ctx, q, "eco:whale", "Whale", false)
	topUpBalance(t, ctx, q, whale.ID, 999_000) // whale has 1,000,000 bUEC

	// Take snapshot AFTER market creation so the creator bonus (+50) is in the
	// baseline and doesn't pollute the LMSR trading-bound measurement.
	m := createActiveMarket(t, ctx, marketSvc, "eco: whale 10k shares", mod.ID, mod.ID)
	users := []int64{mod.ID, whale.ID}
	before := sumBalances(t, ctx, q, users...)

	const shares = 10_000.0
	cost := service.BuyCost(defaultLiqB, 0, 0, shares, true)
	payout := int64(shares) * payoutPerShareE

	t.Logf("=== Whale buys %.0f YES shares ===", shares)
	t.Logf("  BuyCost:                %d bUEC (%.2f bUEC/share)", cost, float64(cost)/shares)
	t.Logf("  Payout if resolves YES: %d bUEC", payout)
	t.Logf("  Net profit if wins:     %d bUEC (%.3fx)", payout-cost, float64(payout)/float64(cost))

	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: whale.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: shares,
	})
	must(t, err, "whale buys 10k YES")
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve YES")

	after := sumBalances(t, ctx, q, users...)
	assertBoundedGrowth(t, before, after, 1, "whale_10k_shares")
	maxGrowth := int64(math.Ceil(lmsrMaxLoss))
	t.Logf("Economy grew by %d bUEC (LMSR bound: %d bUEC)", after-before, maxGrowth)
}

// TestEconomy_SellBeforeResolution: buy+sell roundtrip, economy must not grow.
func TestEconomy_SellBeforeResolution(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod6", "Mod6", true)
	trader := createTestUser(t, ctx, q, "eco:trader6", "Trader6", false)
	m := createActiveMarket(t, ctx, marketSvc, "eco: buy & sell", mod.ID, mod.ID)

	users := []int64{mod.ID, trader.ID}
	before := sumBalances(t, ctx, q, users...)

	const shares = 5.0
	buyRes, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: trader.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: shares,
	})
	must(t, err, "buy")

	mkt, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get market after buy")
	sellRev := service.SellRevenue(defaultLiqB, mkt.YesShares, mkt.NoShares, shares, true)

	sellRes, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: trader.ID, MarketID: m.ID, Side: "yes", Action: "sell", Shares: shares,
	})
	must(t, err, "sell")

	spread := buyRes.Cost - (-sellRes.Cost)
	t.Logf("buy=%d, sell=%d (manual=%d), spread=%d", buyRes.Cost, -sellRes.Cost, sellRev, spread)
	if spread < 0 {
		t.Errorf("negative spread: user profited %d bUEC from buy+sell roundtrip", -spread)
	}
	after := sumBalances(t, ctx, q, users...)
	if after > before {
		t.Errorf("economy grew by %d bUEC from a buy+sell with no resolution", after-before)
	}
}

// TestEconomy_MultipleMarkets_MultipleUsers: 2 markets, 4 users, mixed bets.
func TestEconomy_MultipleMarkets_MultipleUsers(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod7", "Mod7", true)
	users := make([]db.User, 4)
	for i := range users {
		users[i] = createTestUser(t, ctx, q, fmt.Sprintf("eco:u7%d", i), fmt.Sprintf("User7%d", i), false)
	}
	allIDs := make([]int64, len(users)+1)
	allIDs[0] = mod.ID
	for i, u := range users {
		allIDs[i+1] = u.ID
	}

	marketA := createActiveMarket(t, ctx, marketSvc, "eco: market A 4.2 ships", mod.ID, mod.ID)
	marketB := createActiveMarket(t, ctx, marketSvc, "eco: market B quantum fix", mod.ID, mod.ID)
	before := sumBalances(t, ctx, q, allIDs...)

	for _, tr := range []struct {
		marketID int64
		userID   int64
		side     string
		shares   float64
	}{
		{marketA.ID, users[0].ID, "yes", 3},
		{marketA.ID, users[1].ID, "no", 2},
		{marketA.ID, users[2].ID, "yes", 2},
		{marketB.ID, users[1].ID, "yes", 2},
		{marketB.ID, users[2].ID, "no", 3},
		{marketB.ID, users[3].ID, "no", 1},
		{marketA.ID, users[3].ID, "yes", 1},
	} {
		_, err := tradingSvc.Execute(ctx, service.TradeInput{
			UserID: tr.userID, MarketID: tr.marketID, Side: tr.side,
			Action: "buy", Shares: tr.shares,
		})
		must(t, err, fmt.Sprintf("trade market=%d user=%d %s %.0f", tr.marketID, tr.userID, tr.side, tr.shares))
	}
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: marketA.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve A YES")
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: marketB.ID, Resolution: "no", ModID: mod.ID,
	}), "resolve B NO")
	after := sumBalances(t, ctx, q, allIDs...)
	assertBoundedGrowth(t, before, after, 2, "two_markets_four_users")
}

// TestEconomy_PriceMovesAfterTrade: YES price increases with each YES buy.
func TestEconomy_PriceMovesAfterTrade(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod8", "Mod8", true)
	trader := createTestUser(t, ctx, q, "eco:trader8", "Trader8", false)
	topUpBalance(t, ctx, q, trader.ID, 5_000)
	m := createActiveMarket(t, ctx, marketSvc, "eco: price impact", mod.ID, mod.ID)

	prevPrice := service.YESPrice(defaultLiqB, 0, 0)
	qYes := 0.0
	for i, lot := range []float64{2, 2, 2, 2, 2} {
		_, err := tradingSvc.Execute(ctx, service.TradeInput{
			UserID: trader.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: lot,
		})
		must(t, err, fmt.Sprintf("buy lot %d", i))
		qYes += lot
		mkt, err := q.GetMarketByID(ctx, m.ID)
		must(t, err, "get market")
		newPrice := service.YESPrice(mkt.LiquidityParam, mkt.YesShares, mkt.NoShares)
		t.Logf("lot %d (qYes=%.0f): YES price = %.2f%% (was %.2f%%)", i+1, qYes, newPrice*100, prevPrice*100)
		if newPrice <= prevPrice {
			t.Errorf("lot %d: price did not increase after buying YES (%.4f -> %.4f)", i+1, prevPrice, newPrice)
		}
		prevPrice = newPrice
	}
}

// TestEconomy_PositionAccumulation: separate buys accumulate; payout = total_shares*100.
func TestEconomy_PositionAccumulation(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod9", "Mod9", true)
	bettor := createTestUser(t, ctx, q, "eco:bettor9", "Bettor9", false)
	topUpBalance(t, ctx, q, bettor.ID, 2000)
	m := createActiveMarket(t, ctx, marketSvc, "eco: position accumulate", mod.ID, mod.ID)

	totalShares := 0.0
	for _, lot := range []float64{2, 3, 2} {
		_, err := tradingSvc.Execute(ctx, service.TradeInput{
			UserID: bettor.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: lot,
		})
		must(t, err, fmt.Sprintf("buy %.0f shares", lot))
		totalShares += lot
	}
	pos, err := q.GetUserPositionOrZero(ctx, bettor.ID, m.ID)
	must(t, err, "get position")
	if math.Abs(pos.YesShares-totalShares) > 0.001 {
		t.Errorf("position YES shares: got %.4f, want %.4f", pos.YesShares, totalShares)
	}
	bettorBefore, err := q.GetUserByID(ctx, bettor.ID)
	must(t, err, "get bettor before resolve")
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve YES")
	bettorAfter, err := q.GetUserByID(ctx, bettor.ID)
	must(t, err, "get bettor after resolve")
	expectedPayout := int64(totalShares) * payoutPerShareE
	actualPayout := bettorAfter.Balance - bettorBefore.Balance
	if actualPayout != expectedPayout {
		t.Errorf("payout: got %d bUEC, want %d bUEC (%.0f shares x %d)", actualPayout, expectedPayout, totalShares, payoutPerShareE)
	}
	t.Logf("Accumulated %.0f YES shares -> %d bUEC payout.", totalShares, actualPayout)
}

// TestEconomy_WinnerPayoutEquality: equal share-count holders get equal payouts.
func TestEconomy_WinnerPayoutEquality(t *testing.T) {
	ctx := context.Background()
	sqlDB, q := newTestDB(t)
	marketSvc, tradingSvc, _ := newServices(sqlDB, q)

	mod := createTestUser(t, ctx, q, "eco:mod10", "Mod10", true)
	early := createTestUser(t, ctx, q, "eco:early10", "Early10", false)
	late := createTestUser(t, ctx, q, "eco:late10", "Late10", false)
	topUpBalance(t, ctx, q, early.ID, 2000)
	topUpBalance(t, ctx, q, late.ID, 2000)
	m := createActiveMarket(t, ctx, marketSvc, "eco: payout equality", mod.ID, mod.ID)

	_, err := tradingSvc.Execute(ctx, service.TradeInput{
		UserID: early.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 5,
	})
	must(t, err, "early buy")
	mkt, err := q.GetMarketByID(ctx, m.ID)
	must(t, err, "get mkt")
	lateCostPre := service.BuyCost(mkt.LiquidityParam, mkt.YesShares, mkt.NoShares, 5, true)
	_, err = tradingSvc.Execute(ctx, service.TradeInput{
		UserID: late.ID, MarketID: m.ID, Side: "yes", Action: "buy", Shares: 5,
	})
	must(t, err, "late buy")
	earlyBefore, _ := q.GetUserByID(ctx, early.ID)
	lateBefore, _ := q.GetUserByID(ctx, late.ID)
	must(t, marketSvc.ResolveMarket(ctx, service.ResolveInput{
		MarketID: m.ID, Resolution: "yes", ModID: mod.ID,
	}), "resolve")
	earlyAfter, _ := q.GetUserByID(ctx, early.ID)
	lateAfter, _ := q.GetUserByID(ctx, late.ID)
	earlyPayout := earlyAfter.Balance - earlyBefore.Balance
	latePayout := lateAfter.Balance - lateBefore.Balance
	if earlyPayout != latePayout {
		t.Errorf("payout should be equal for equal share counts: early=%d, late=%d", earlyPayout, latePayout)
	}
	t.Logf("Both held 5 YES shares -> %d bUEC each; late buyer paid %d (more expensive)", earlyPayout, lateCostPre)
}

// TestEconomy_PerShareCostMatchesProbability: per-share cost in bUEC ~ probability%.
func TestEconomy_PerShareCostMatchesProbability(t *testing.T) {
	cases := []struct{ qYes, qNo float64 }{
		{0, 0},  // 50%
		{30, 0}, // ~75%
		{0, 30}, // ~25%
	}
	for _, tc := range cases {
		prob := service.YESPrice(defaultLiqB, tc.qYes, tc.qNo)
		cost := service.BuyCost(defaultLiqB, tc.qYes, tc.qNo, 1, true)
		expected := prob * payoutPerShareE
		if float64(cost) < expected-20 || float64(cost) > expected+20 {
			t.Errorf("at qYes=%.0f qNo=%.0f (p=%.0f%%): BuyCost=%d bUEC, want ~%.0f bUEC",
				tc.qYes, tc.qNo, prob*100, cost, expected,
			)
		}
		t.Logf("p=%.0f%% -> BuyCost=%d bUEC (expected ~%.0f)", prob*100, cost, expected)
	}
}
