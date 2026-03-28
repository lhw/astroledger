package service

import "math"

// OutcomeProb returns the normalized probability of one outcome given all
// outcomes' share pools. Uses the LMSR formula: p_i = exp(q_i/b) / Σ exp(q_j/b).
// allShares must include the share count for the target outcome at position idx.
func OutcomeProb(b float64, allShares []float64, idx int) float64 {
	// Log-sum-exp trick for numerical stability.
	if len(allShares) == 0 {
		return 0
	}
	maxVal := allShares[0] / b
	for _, q := range allShares[1:] {
		if v := q / b; v > maxVal {
			maxVal = v
		}
	}
	sum := 0.0
	for _, q := range allShares {
		sum += math.Exp(q/b - maxVal)
	}
	return math.Exp(allShares[idx]/b-maxVal) / sum
}

// OutcomeProbCents returns the probability in whole bUEC-per-share cents (1–99).
func OutcomeProbCents(b float64, allShares []float64, idx int) int64 {
	p := OutcomeProb(b, allShares, idx)
	v := int64(math.Round(p * 100))
	if v < 1 {
		v = 1
	}
	if v > 99 {
		v = 99
	}
	return v
}

// lmsrMulti computes the multi-outcome LMSR cost function using independent
// pools: C = b * log(Σ exp(q_i/b)).
func lmsrMulti(b float64, shares []float64) float64 {
	// Log-sum-exp.
	maxVal := shares[0] / b
	for _, q := range shares[1:] {
		if v := q / b; v > maxVal {
			maxVal = v
		}
	}
	sum := 0.0
	for _, q := range shares {
		sum += math.Exp(q/b - maxVal)
	}
	return b * (maxVal + math.Log(sum))
}

// BuyCost returns the integer bUEC cost to buy deltaShares of outcome outcomeIdx.
// Rounds up to protect the AMM.
func BuyCost(b float64, shares []float64, outcomeIdx int, deltaShares float64) int64 {
	before := lmsrMulti(b, shares)
	after := make([]float64, len(shares))
	copy(after, shares)
	after[outcomeIdx] += deltaShares
	diff := lmsrMulti(b, after) - before
	// Scale by payoutPerShare (100) so per-share cost in bUEC ≈ probability%.
	return int64(math.Ceil(diff * 100))
}

// SellRevenue returns the integer bUEC received for selling deltaShares.
// Rounds down to protect the AMM.
func SellRevenue(b float64, shares []float64, outcomeIdx int, deltaShares float64) int64 {
	before := lmsrMulti(b, shares)
	after := make([]float64, len(shares))
	copy(after, shares)
	after[outcomeIdx] -= deltaShares
	diff := before - lmsrMulti(b, after)
	return int64(math.Floor(diff * 100))
}

// MaxAffordableShares binary-searches for the max whole shares buyable with budget bUEC.
func MaxAffordableShares(b float64, shares []float64, outcomeIdx int, budget int64) float64 {
	if budget <= 0 {
		return 0
	}
	lo, hi := 0.0, float64(budget)
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if BuyCost(b, shares, outcomeIdx, mid) <= budget {
			lo = mid
		} else {
			hi = mid
		}
	}
	if BuyCost(b, shares, outcomeIdx, lo) > budget {
		return 0
	}
	return lo
}
