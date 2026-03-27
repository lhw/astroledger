/**
 * Client-side LMSR (Logarithmic Market Scoring Rule) helpers.
 *
 * These mirror the canonical math in backend/internal/service/amm.go.
 * Keep them in sync if the AMM logic ever changes.
 *
 * The multi-outcome LMSR cost function:
 *   C(q) = b * ln( Σ exp(q_i / b) )
 *
 * For binary markets, shares = [qYes, qNo] and outcomeIdx ∈ {0, 1}.
 */

/** Multi-outcome LMSR cost function. */
export function lmsrCostN(b: number, shares: number[]): number {
	const maxExp = Math.max(...shares.map((s) => s / b));
	const sum = shares.reduce((acc, s) => acc + Math.exp(s / b - maxExp), 0);
	return b * (maxExp + Math.log(sum));
}

/** Cost (in bUEC, rounded up) to buy `delta` shares of outcome at `outcomeIdx`. */
export function buyCost(b: number, shares: number[], outcomeIdx: number, delta: number): number {
	const before = lmsrCostN(b, shares);
	const after_shares = shares.map((s, i) => (i === outcomeIdx ? s + delta : s));
	const after = lmsrCostN(b, after_shares);
	return Math.ceil((after - before) * 100);
}

/** Revenue (in bUEC, rounded down) to sell `delta` shares of outcome at `outcomeIdx`. */
export function sellRevenue(
	b: number,
	shares: number[],
	outcomeIdx: number,
	delta: number
): number {
	if (delta <= 0) return 0;
	const before = lmsrCostN(b, shares);
	const after_shares = shares.map((s, i) => (i === outcomeIdx ? s - delta : s));
	const after = lmsrCostN(b, after_shares);
	return Math.floor((before - after) * 100);
}

/** Probability (0–1) for outcome at `outcomeIdx`. */
export function outcomeProb(b: number, shares: number[], outcomeIdx: number): number {
	const maxExp = Math.max(...shares.map((s) => s / b));
	const denom = shares.reduce((acc, s) => acc + Math.exp(s / b - maxExp), 0);
	return Math.exp(shares[outcomeIdx] / b - maxExp) / denom;
}

/** Price in cents (1–99) for outcome at `outcomeIdx`. */
export function outcomePriceCents(b: number, shares: number[], outcomeIdx: number): number {
	const p = outcomeProb(b, shares, outcomeIdx);
	return Math.max(1, Math.min(99, Math.round(p * 100)));
}

/** Binary-search the maximum whole shares buyable within a bUEC budget. */
export function maxAffordable(
	b: number,
	shares: number[],
	outcomeIdx: number,
	budget: number
): number {
	if (budget <= 0) return 0;
	let lo = 0,
		hi = budget;
	while (lo < hi) {
		const mid = Math.floor((lo + hi + 1) / 2);
		if (buyCost(b, shares, outcomeIdx, mid) <= budget) lo = mid;
		else hi = mid - 1;
	}
	return lo;
}

// ── Legacy binary helpers (for existing references) ─────────────────────

/** @deprecated Use lmsrCostN instead. */
export function lmsrCost(b: number, qYes: number, qNo: number): number {
	return lmsrCostN(b, [qYes, qNo]);
}

/** @deprecated Use buyCost with shares array. */
export function buyCostBinary(
	b: number,
	qYes: number,
	qNo: number,
	shares: number,
	yes: boolean
): number {
	return buyCost(b, [qYes, qNo], yes ? 0 : 1, shares);
}

/** @deprecated Use sellRevenue with shares array. */
export function sellRevenueBinary(
	b: number,
	qYes: number,
	qNo: number,
	shares: number,
	yes: boolean
): number {
	return sellRevenue(b, [qYes, qNo], yes ? 0 : 1, shares);
}

/** @deprecated Use outcomeProb with shares array. */
export function yesProb(b: number, qYes: number, qNo: number): number {
	return outcomeProb(b, [qYes, qNo], 0);
}

/** @deprecated Use maxAffordable with shares array. */
export function maxAffordableBinary(
	b: number,
	qYes: number,
	qNo: number,
	budget: number,
	yes: boolean
): number {
	return maxAffordable(b, [qYes, qNo], yes ? 0 : 1, budget);
}
