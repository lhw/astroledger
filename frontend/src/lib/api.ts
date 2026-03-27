/**
 * Centralized API client. All backend calls go through here.
 * Uses relative URLs so the Vite dev proxy forwards to localhost:8080.
 */

import type {
	User,
	PublicUser,
	LeaderboardRow,
	Market,
	MarketWithPrice,
	MarketList,
	MarketCategory,
	TradeWithMarket,
	TradeWithTrader,
	Position,
	TradeResult,
	CreateMarketBody,
	ApiError,
	ResolutionRequestMarket,
	Badge,
	StoreBadge,
	AdmiralRanksResponse,
	Report,
	PricePoint,
	Comment,
	DetectedPatch,
	AnalyticsStats,
	UserSearchResult,
	BadgeCatalogEntry,
	AdminBadgeRelease
} from './types';

class ApiClientError extends Error {
	constructor(
		public readonly status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiClientError';
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, {
		credentials: 'include',
		headers: { 'Content-Type': 'application/json', ...init?.headers },
		...init
	});

	if (!res.ok) {
		let message = `HTTP ${res.status}`;
		try {
			const err = (await res.json()) as ApiError;
			message = err.error ?? message;
		} catch {
			// ignore parse error
		}
		throw new ApiClientError(res.status, message);
	}

	return res.json() as Promise<T>;
}

// --- Auth ---

/** Redirects browser to backend OIDC login. */
export function loginWithSCID() {
	window.location.href = '/auth/login';
}

/** Logs out and reloads. */
export async function logout() {
	await fetch('/auth/logout', { method: 'POST', credentials: 'include' });
	window.location.reload();
}

// --- Users ---

export async function getMe(): Promise<User | null> {
	try {
		return await request<User>('/api/me');
	} catch (err) {
		if (err instanceof ApiClientError && err.status === 401) return null;
		throw err;
	}
}

export async function getUser(id: number): Promise<PublicUser> {
	return request<PublicUser>(`/api/users/${id}`);
}

export async function getLeaderboard(limit = 25): Promise<LeaderboardRow[]> {
	return request<LeaderboardRow[]>(`/api/leaderboard?limit=${limit}`);
}

export async function getMyPositions(): Promise<Position[]> {
	return request<Position[]>('/api/me/positions');
}

export async function getMyTrades(offset = 0): Promise<TradeWithMarket[]> {
	return request<TradeWithMarket[]>(`/api/me/trades?offset=${offset}`);
}

// --- Markets ---

export async function listMarkets(
	status = 'active',
	category: MarketCategory | '' = '',
	offset = 0
): Promise<MarketList> {
	const params = new URLSearchParams({ status, offset: String(offset) });
	if (category) params.set('category', category);
	return request<MarketList>(`/api/markets?${params}`);
}

export async function getMarket(id: number): Promise<MarketWithPrice> {
	return request<MarketWithPrice>(`/api/markets/${id}`);
}

export async function createMarket(body: CreateMarketBody): Promise<Market> {
	return request<Market>('/api/markets', {
		method: 'POST',
		body: JSON.stringify(body)
	});
}

export async function getMarketPriceHistory(id: number): Promise<PricePoint[]> {
	return request<PricePoint[]>(`/api/markets/${id}/history`);
}

export async function getMarketTrades(id: number, offset = 0): Promise<TradeWithTrader[]> {
	return request<TradeWithTrader[]>(`/api/markets/${id}/trades?offset=${offset}`);
}

// --- Trading ---

export async function executeTrade(
	market_id: number,
	outcome_id: number,
	action: 'buy' | 'sell',
	shares: number
): Promise<TradeResult> {
	return request<TradeResult>('/api/trades', {
		method: 'POST',
		body: JSON.stringify({ market_id, outcome_id, action, shares })
	});
}

// --- Moderation ---

export async function listPendingMarkets(): Promise<(Market & { creator_name: string })[]> {
	return request('/api/mod/markets');
}

export async function approveMarket(id: number): Promise<void> {
	await request(`/api/mod/markets/${id}/approve`, { method: 'POST' });
}

export async function rejectMarket(id: number): Promise<void> {
	await request(`/api/mod/markets/${id}/reject`, { method: 'POST' });
}

export async function resolveMarket(id: number, winningOutcomeId: number, evidenceLink?: string): Promise<void> {
	await request(`/api/mod/markets/${id}/resolve`, {
		method: 'POST',
		body: JSON.stringify({ winning_outcome_id: winningOutcomeId, evidence_link: evidenceLink || undefined })
	});
}

export async function requestResolution(
	marketId: number,
	link?: string,
	note?: string
): Promise<void> {
	await request(`/api/markets/${marketId}/request-resolution`, {
		method: 'POST',
		body: JSON.stringify({ link: link ?? '', note: note ?? '' })
	});
}

export async function listResolutionRequestedMarkets(): Promise<ResolutionRequestMarket[]> {
	return request('/api/mod/resolution-requests');
}

export async function denyResolutionRequest(marketId: number): Promise<void> {
	await request(`/api/mod/markets/${marketId}/deny-resolution`, { method: 'POST' });
}

// --- Badges ---

export async function getMyBadges(): Promise<Badge[]> {
	return request<Badge[]>('/api/me/badges');
}

export async function getUserBadges(id: number): Promise<Badge[]> {
	return request<Badge[]>(`/api/users/${id}/badges`);
}

export async function getStoreBadges(): Promise<StoreBadge[]> {
	return request<StoreBadge[]>('/api/fomo');
}

export async function purchaseBadge(badge_key: string): Promise<{ status: string }> {
	return request<{ status: string }>('/api/fomo/purchase', {
		method: 'POST',
		body: JSON.stringify({ badge_key })
	});
}

export async function getAdmiralRanks(): Promise<AdmiralRanksResponse> {
	return request<AdmiralRanksResponse>('/api/admiral');
}

export async function setActiveBadge(badge_key: string): Promise<{ active_badge_key: string }> {
	return request<{ active_badge_key: string }>('/api/me/badge', {
		method: 'PUT',
		body: JSON.stringify({ badge_key })
	});
}

// --- Reports ---

export async function submitReport(marketId: number, reason: string): Promise<{ id: number }> {
	return request<{ id: number }>('/api/reports', {
		method: 'POST',
		body: JSON.stringify({ market_id: marketId, reason })
	});
}

export async function listPendingReports(): Promise<Report[]> {
	return request<Report[]>('/api/mod/reports');
}

export async function reviewReport(id: number): Promise<void> {
	await request(`/api/mod/reports/${id}/review`, { method: 'POST' });
}

export async function dismissReport(id: number): Promise<void> {
	await request(`/api/mod/reports/${id}/dismiss`, { method: 'POST' });
}

// --- Comments ---

export async function getComments(marketId: number): Promise<Comment[]> {
	return request<Comment[]>(`/api/markets/${marketId}/comments`);
}

export async function postComment(marketId: number, content: string): Promise<Comment> {
	return request<Comment>(`/api/markets/${marketId}/comments`, {
		method: 'POST',
		body: JSON.stringify({ content })
	});
}

export async function deleteComment(commentId: number): Promise<void> {
	await request(`/api/comments/${commentId}`, { method: 'DELETE' });
}

// --- Admin ---

export interface WeeklyPayoutResult {
	users_paid: number;
	credits_per_user: number;
	message: string;
}

export async function adminTriggerWeeklyPayout(): Promise<WeeklyPayoutResult> {
	return request<WeeklyPayoutResult>('/api/admin/weekly-payout', { method: 'POST' });
}

export async function adminAdjustBalance(
	userId: number,
	amount: number,
	reason: string
): Promise<{ new_balance: number }> {
	return request<{ new_balance: number }>(`/api/admin/users/${userId}/balance`, {
		method: 'POST',
		body: JSON.stringify({ amount, reason })
	});
}

export async function adminGetAnalytics(period: '7d' | '30d'): Promise<AnalyticsStats> {
	return request<AnalyticsStats>(`/api/admin/analytics?period=${period}`);
}

export async function adminSearchUsers(q: string): Promise<UserSearchResult[]> {
	return request<UserSearchResult[]>(`/api/admin/users/search?q=${encodeURIComponent(q)}`);
}

// --- Patches ---

export async function getPatches(): Promise<DetectedPatch[]> {
	const res = await request<{ patches: DetectedPatch[] }>('/api/patches');
	return res.patches;
}

export async function markPatchNotified(id: number): Promise<void> {
	await request(`/api/mod/patches/${id}/notify`, { method: 'POST' });
}

// --- Admin: Badge Releases ---

export async function adminGetBadgeCatalog(): Promise<BadgeCatalogEntry[]> {
	return request<BadgeCatalogEntry[]>('/api/admin/badge-catalog');
}

export async function adminListBadgeReleases(): Promise<AdminBadgeRelease[]> {
	return request<AdminBadgeRelease[]>('/api/admin/badge-releases');
}

export async function adminCreateBadgeRelease(body: {
	badge_key: string;
	price: number;
	stock: number | null;
	released_at: string;
	expires_at: string | null;
	notes: string | null;
}): Promise<AdminBadgeRelease> {
	return request<AdminBadgeRelease>('/api/admin/badge-releases', {
		method: 'POST',
		body: JSON.stringify(body)
	});
}

export async function adminUpdateBadgeRelease(
	id: number,
	body: Partial<{
		price: number;
		stock: number | null;
		released_at: string;
		expires_at: string | null;
		active: boolean;
		notes: string | null;
	}>
): Promise<AdminBadgeRelease> {
	return request<AdminBadgeRelease>(`/api/admin/badge-releases/${id}`, {
		method: 'PUT',
		body: JSON.stringify(body)
	});
}

export async function adminArchiveBadgeRelease(id: number): Promise<void> {
	await request(`/api/admin/badge-releases/${id}`, { method: 'DELETE' });
}

export { ApiClientError };
