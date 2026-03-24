package service

import "math"

// YESPrice returns the current YES probability in the range (0, 1) given LMSR state.
// b is the liquidity parameter; qYes and qNo are total outstanding shares.
func YESPrice(b, qYes, qNo float64) float64 {
	// Numerically stable: 1 / (1 + exp((qNo - qYes) / b))
	return 1.0 / (1.0 + math.Exp((qNo-qYes)/b))
}

// PriceCents returns the YES price in whole ScollyBucks cents (1-99).
func PriceCents(b, qYes, qNo float64) int64 {
	p := YESPrice(b, qYes, qNo)
	v := int64(math.Round(p * 100))
	if v < 1 {
		v = 1
	}
	if v > 99 {
		v = 99
	}
	return v
}

// lmsr computes the LMSR cost function C(qYes, qNo) = b * log(exp(qYes/b) + exp(qNo/b)).
// Uses the log-sum-exp trick for numerical stability.
func lmsr(b, qYes, qNo float64) float64 {
	a := qYes / b
	c := qNo / b
	m := math.Max(a, c)
	return b * (m + math.Log(math.Exp(a-m)+math.Exp(c-m)))
}

// BuyCost returns the integer ScollyBucks cost to buy deltaShares of the given side.
// sideYes=true means buying YES shares, false means NO.
// Returns cost rounded up to protect the AMM.
func BuyCost(b, qYes, qNo, deltaShares float64, sideYes bool) int64 {
	var before, after float64
	before = lmsr(b, qYes, qNo)
	if sideYes {
		after = lmsr(b, qYes+deltaShares, qNo)
	} else {
		after = lmsr(b, qYes, qNo+deltaShares)
	}
	diff := after - before
	// Round up to protect the market maker.
	return int64(math.Ceil(diff))
}

// SellRevenue returns the integer ScollyBucks received for selling deltaShares.
// Returns revenue rounded down to protect the AMM.
func SellRevenue(b, qYes, qNo, deltaShares float64, sideYes bool) int64 {
	var before, after float64
	before = lmsr(b, qYes, qNo)
	if sideYes {
		after = lmsr(b, qYes-deltaShares, qNo)
	} else {
		after = lmsr(b, qYes, qNo-deltaShares)
	}
	diff := before - after // positive: revenue from selling
	// Round down to protect the market maker.
	return int64(math.Floor(diff))
}

// MaxAffordableShares binary-searches for the max shares a user can buy with budget ScollyBucks.
// Returns 0 if even 1 share is too expensive.
func MaxAffordableShares(b, qYes, qNo float64, budget int64, sideYes bool) float64 {
	if budget <= 0 {
		return 0
	}
	lo, hi := 0.0, float64(budget) // can't buy more shares than budget (1 share costs >= 1)
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if BuyCost(b, qYes, qNo, mid, sideYes) <= budget {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Verify final cost doesn't exceed budget (handle floating-point rounding).
	if BuyCost(b, qYes, qNo, lo, sideYes) > budget {
		return 0
	}
	return lo
}
