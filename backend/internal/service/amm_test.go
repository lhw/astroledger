package service

// AMM economy unit tests.
//
// These tests verify the core correctness property of the LMSR AMM:
//
//	cost_to_buy_n_shares  ≈  n × (probability%) bUEC
//	payout_if_win         =  n × 100 bUEC
//
// That means at p=50% the cost of 1 share should be ~50 bUEC, not ~1 bUEC.
// The maximum net gain the platform can "gift" to users per market is bounded by
// b × ln(2) × payoutPerShare ≈ 6,931 bUEC (with b=100).
//
// Any test below that FAILS indicates the scale factor of 100 is missing from
// BuyCost / SellRevenue.

import (
	"fmt"
	"math"
	"testing"
)

const (
	// payoutPerShare is the fixed payout each winning share receives at resolution.
	payoutPerShare = 100
)

// ─── Pure AMM math ────────────────────────────────────────────────────────────

// TestAMMPerShareCost_AtFiftyPercent verifies that buying a single YES share
// from the starting state (equal odds, p=50%) costs roughly 50 bUEC — matching
// the displayed probability percentage that users see in the UI.
//
// With the missing ×100 scale factor this returns 1 bUEC due to integer ceiling.
func TestAMMPerShareCost_AtFiftyPercent(t *testing.T) {
	b := 100.0
	// Initial state: no shares outstanding, 50 / 50 probability.
	costBatch := BuyCostBinary(b, 0, 0, 10, true) // 10 shares to smooth integer ceiling
	perShare := float64(costBatch) / 10.0

	// At p=50%, the marginal price per share is 50 bUEC when correctly scaled.
	// Allow ±10 bUEC for small-batch price impact.
	if perShare < 40 || perShare > 60 {
		t.Errorf(
			"per-share cost at 50%% probability = %.1f bUEC; want 40–60 bUEC.\n\n"+
				"DIAGNOSIS: BuyCost is returning %.1f bUEC/share, which matches the\n"+
				"raw LMSR differential (~0.5) with only integer ceiling applied.\n"+
				"The cost function is missing the ×%d scale factor needed so that\n"+
				"prices in bUEC match the probability percentage shown in the UI.\n\n"+
				"FIX: change `return int64(math.Ceil(diff))` to\n"+
				"         `return int64(math.Ceil(diff * %d))`\n"+
				"     in BuyCost (and mirror it in SellRevenue).",
			perShare, perShare, payoutPerShare, payoutPerShare,
		)
	}
}

// TestAMMPerShareCost_AtTenPercent verifies that at ~10% probability (achieved
// by a large NO imbalance) the per-share YES cost is ~10 bUEC.
func TestAMMPerShareCost_AtTenPercent(t *testing.T) {
	b := 100.0
	// Push the market toward ~10% by pre-loading no-shares.
	// p_yes = 1/(1+exp((qNo-qYes)/b)) ≈ 0.10 when qNo ≈ b*ln(9) ≈ 220
	qNo := b * math.Log(9) // ~219.7, giving p_yes ≈ 10%
	prob := YESPrice(b, 0, qNo)
	if math.Abs(prob-0.10) > 0.01 {
		t.Fatalf("test setup: wanted p≈0.10, got %.3f", prob)
	}

	costBatch := BuyCostBinary(b, 0, qNo, 10, true)
	perShare := float64(costBatch) / 10.0

	// At p=10%, fair price per share ≈ 10 bUEC.
	if perShare < 5 || perShare > 20 {
		t.Errorf(
			"per-share cost at %.0f%% probability = %.1f bUEC; want 5–20 bUEC.\n"+
				"At p≈10%%, each winning share pays %d bUEC so the fair price is ~10 bUEC.",
			prob*100, perShare, payoutPerShare,
		)
	}
}

// TestAMMExpectedValue verifies that for a small number of shares the cost is
// close to the probability-weighted expected payout (within 50% tolerance for
// price impact on small batches).
//
// cost ≈ p × n × payoutPerShare
//
// With the missing scale the cost is 100× too low, producing a ratio of ~0.01.
func TestAMMExpectedValue(t *testing.T) {
	cases := []struct {
		name         string
		b, qYes, qNo float64
		shares       float64
	}{
		{"50pct_start", 100, 0, 0, 5},
		{"75pct_yes_edge", 100, 100, 0, 5}, // qYes pushed up
		{"25pct_no_edge", 100, 0, 100, 5},  // qNo pushed up
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p := YESPrice(tc.b, tc.qYes, tc.qNo)
			cost := float64(BuyCostBinary(tc.b, tc.qYes, tc.qNo, tc.shares, true))
			ev := p * tc.shares * payoutPerShare

			// cost / ev should be in the range [0.5, 2.0].
			// Far outside this means the scale is wrong.
			ratio := cost / ev
			if ratio < 0.5 || ratio > 2.0 {
				t.Errorf(
					"%s: p=%.0f%%, %g shares → cost=%g bUEC, EV=%.1f bUEC, ratio=%.3f.\n"+
						"Expected ratio near 1.0; got %.3f, indicating a ~%.0f× scale mismatch.\n"+
						"FIX: multiply BuyCost diff by %d before returning.",
					tc.name, p*100, tc.shares, cost, ev, ratio, ratio,
					math.Abs(math.Log10(ratio)+2), payoutPerShare,
				)
			}
		})
	}
}

// TestAMMBoundedMarketMakerLoss is the core LMSR safety guarantee:
// no matter how many shares a single user buys, the platform's maximum net
// payout above collected costs is bounded by b × ln(2) × payoutPerShare.
//
// With b=100 and payoutPerShare=100 that bound is ≈ 6,931 bUEC per market.
func TestAMMBoundedMarketMakerLoss(t *testing.T) {
	b := 100.0
	maxAllowedLoss := int64(math.Ceil(b * math.Log(2) * payoutPerShare)) // ≈ 6,931

	// Try increasingly aggressive single-user scenarios.
	for _, shares := range []float64{100, 1_000, 10_000, 100_000} {
		shares := shares
		t.Run(fmt.Sprintf("%.0f_shares", shares), func(t *testing.T) {
			cost := BuyCostBinary(b, 0, 0, shares, true)
			payout := int64(shares) * payoutPerShare
			platLoss := payout - cost

			t.Logf(
				"%.0f YES shares from start: cost=%d bUEC, resolve-YES payout=%d bUEC, platform loss=%d bUEC (max allowed %d)",
				shares, cost, payout, platLoss, maxAllowedLoss,
			)

			if platLoss > maxAllowedLoss {
				t.Errorf(
					"LMSR bounded-loss violated for %.0f shares!\n"+
						"  cost collected : %d bUEC\n"+
						"  payout if wins : %d bUEC\n"+
						"  platform loss  : %d bUEC  (>  max allowed %d bUEC)\n\n"+
						"ROOT CAUSE: BuyCost does not scale its output by ×%d.\n"+
						"  The LMSR diff for %.0f shares is ~%.0f,\n"+
						"  but it needs to be ~%.0f (×%d) so that costs match payout scale.\n\n"+
						"FIX: `return int64(math.Ceil(diff * %d))` in BuyCost\n"+
						"     `return int64(math.Floor(diff * %d))` in SellRevenue",
					shares,
					cost, payout, platLoss, maxAllowedLoss,
					payoutPerShare,
					shares, float64(cost), float64(cost)*float64(payoutPerShare),
					payoutPerShare, payoutPerShare, payoutPerShare,
				)
			}
		})
	}
}

// TestAMMSellRevenueLEBuyCost verifies the AMM spread is non-negative:
// you should never receive more from selling than you paid to buy.
func TestAMMSellRevenueLEBuyCost(t *testing.T) {
	b := 100.0

	for _, shares := range []float64{1, 5, 20, 100} {
		shares := shares
		t.Run(fmt.Sprintf("%.0f_shares", shares), func(t *testing.T) {
			buy := BuyCostBinary(b, 0, 0, shares, true)
			// After buying, sell from the new state.
			sell := SellRevenueBinary(b, shares, 0, shares, true)

			if sell > buy {
				t.Errorf(
					"AMM spread violation for %.0f shares: buy=%d, sell=%d (sell > buy by %d).",
					shares, buy, sell, sell-buy,
				)
			}
			t.Logf("%.0f shares: buy=%d, sell=%d, spread=%d", shares, buy, sell, buy-sell)
		})
	}
}

// TestAMMPriceMonotonicity confirms that buying more shares always costs more.
func TestAMMPriceMonotonicity(t *testing.T) {
	b := 100.0
	prev := int64(-1)
	for _, n := range []float64{1, 5, 10, 50, 100, 500, 1000} {
		cost := BuyCostBinary(b, 0, 0, n, true)
		if cost <= prev {
			t.Errorf("cost for %.0f shares (%d) ≤ cost for fewer shares (%d): not monotone", n, cost, prev)
		}
		prev = cost
	}
}

// TestAMMHeavySingleBettor_BugRepro reproduces the exact user-reported bug:
// "I buy 10,000 shares on YES (~1 bUEC/ea) and if the market is resolved I now
// have above a million in bUEC."
//
// At b=100, fresh market (p=50%): BuyCostBinary(100, 0, 0, 10000, true) returns
// 9931 bUEC (in the broken implementation). Payout = 10000 × 100 = 1,000,000.
// Net profit = 990,069 bUEC — ~100× the investment with zero real risk.
func TestAMMHeavySingleBettor_BugRepro(t *testing.T) {
	b := 100.0
	shares := 10_000.0

	cost := BuyCostBinary(b, 0, 0, shares, true)
	payout := int64(shares) * payoutPerShare
	netProfit := payout - cost
	maxAllowedPlatformLoss := int64(math.Ceil(b * math.Log(2) * payoutPerShare)) // ≈ 6,931

	t.Logf("=== Heavy Single Bettor Scenario ===")
	t.Logf("  Shares bought (YES, from start): %.0f", shares)
	t.Logf("  Cost paid:                       %d bUEC (%.2f bUEC/share)", cost, float64(cost)/shares)
	t.Logf("  Payout if resolves YES:          %d bUEC", payout)
	t.Logf("  Net profit to user:              %d bUEC (%.1f×)", netProfit, float64(payout)/float64(cost))
	t.Logf("  Max allowed platform loss:       %d bUEC", maxAllowedPlatformLoss)

	if netProfit > maxAllowedPlatformLoss {
		t.Errorf(
			"BUG CONFIRMED — user earns %d bUEC profit on a %d bUEC bet (%.0f×× return).\n"+
				"The platform is over-paying by %d bUEC beyond the LMSR safety bound.\n\n"+
				"EXPECTED: cost ≈ %d bUEC (%.0f×× payout at p≈100%% after price impact).\n"+
				"ACTUAL cost: %d bUEC — this is the raw LMSR diff (~%.0f) without ×%d scaling.\n\n"+
				"FIX in backend/internal/service/amm.go:\n"+
				"  BuyCost:    `return int64(math.Ceil(diff  * %d))`\n"+
				"  SellRevenue:`return int64(math.Floor(diff * %d))`\n\n"+
				"FIX in frontend/src/lib/amm.ts:\n"+
				"  buyCost:    `return Math.ceil((after - before)  * %d)`\n"+
				"  sellRevenue:`return Math.floor((before - after) * %d)`",
			netProfit, cost, float64(payout)/float64(cost),
			netProfit-maxAllowedPlatformLoss,
			payout-maxAllowedPlatformLoss,
			float64(payout)/float64(payout-maxAllowedPlatformLoss),
			cost, float64(cost),
			payoutPerShare, payoutPerShare, payoutPerShare, payoutPerShare, payoutPerShare,
		)
	}
}

// TestAMMMaxAffordableSharesRoundTrip verifies that MaxAffordableShares returns
// a share count whose actual cost is within budget.
func TestAMMMaxAffordableSharesRoundTrip(t *testing.T) {
	b := 100.0

	for _, budget := range []int64{100, 500, 1000, 5000} {
		budget := budget
		t.Run(fmt.Sprintf("budget_%d", budget), func(t *testing.T) {
			maxS := MaxAffordableShares(b, []float64{0, 0}, 0, budget)
			if maxS <= 0 {
				t.Fatalf("budget %d: MaxAffordableShares returned %.4f (≤0)", budget, maxS)
			}

			actualCost := BuyCostBinary(b, 0, 0, maxS, true)
			if actualCost > budget {
				t.Errorf(
					"budget %d: MaxAffordableShares=%.4f but BuyCost=%d > budget.",
					budget, maxS, actualCost,
				)
			}

			// One more share should exceed budget.
			oneMorer := BuyCostBinary(b, 0, 0, maxS+1, true)
			if oneMorer <= budget {
				// Not a hard failure — integer floor means this can sometimes still fit.
				t.Logf("budget %d: MaxAffordableShares+1 still fits (%d ≤ %d) — minor floor artefact", budget, oneMorer, budget)
			}

			t.Logf("budget %d → max %.0f shares @ cost %d bUEC", budget, maxS, actualCost)
		})
	}
}
