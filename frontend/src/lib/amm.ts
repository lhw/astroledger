/**
 * Client-side LMSR (Logarithmic Market Scoring Rule) helpers.
 *
 * These mirror the canonical math in backend/internal/service/amm.go.
 * Keep them in sync if the AMM logic ever changes.
 */

/** LMSR cost function: total cost to hold all outstanding shares. */
export function lmsrCost(b: number, qYes: number, qNo: number): number {
	const a = qYes / b,
		c = qNo / b;
	const m = Math.max(a, c);
	return b * (m + Math.log(Math.exp(a - m) + Math.exp(c - m)));
}

/** Cost (in bUEC, rounded up) to buy `shares` YES or NO shares. */
export function buyCost(
	b: number,
	qYes: number,
	qNo: number,
	shares: number,
	yes: boolean
): number {
	const before = lmsrCost(b, qYes, qNo);
	const after = yes ? lmsrCost(b, qYes + shares, qNo) : lmsrCost(b, qYes, qNo + shares);
	return Math.ceil(after - before);
}

/** Revenue (in bUEC, rounded down) received for selling `shares` YES or NO shares. */
export function sellRevenue(
	b: number,
	qYes: number,
	qNo: number,
	shares: number,
	yes: boolean
): number {
	if (shares <= 0) return 0;
	const before = lmsrCost(b, qYes, qNo);
	const after = yes ? lmsrCost(b, qYes - shares, qNo) : lmsrCost(b, qYes, qNo - shares);
	return Math.floor(before - after);
}

/**
 * Instantaneous probability / price of YES shares (0–1 range).
 * This is the LMSR derivative: exp(qYes/b) / (exp(qYes/b) + exp(qNo/b)).
 */
export function yesProb(b: number, qYes: number, qNo: number): number {
	return 1 / (1 + Math.exp((qNo - qYes) / b));
}

/** Binary-search the maximum whole shares buyable within a bUEC budget. */
export function maxAffordable(
	b: number,
	qYes: number,
	qNo: number,
	budget: number,
	yes: boolean
): number {
	if (budget <= 0) return 0;
	let lo = 0,
		hi = budget;
	while (lo < hi) {
		const mid = Math.floor((lo + hi + 1) / 2);
		if (buyCost(b, qYes, qNo, mid, yes) <= budget) lo = mid;
		else hi = mid - 1;
	}
	return lo;
}
