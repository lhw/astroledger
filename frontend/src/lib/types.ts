/** User profile returned by GET /api/me */
export interface User {
	id: number;
	display_name: string;
	email: string;
	balance: number;
	is_moderator: boolean;
	is_admin: boolean;
	is_rsi_verified: boolean;
	rsi_handle: string | null;
	rsi_verified_at: string | null;
	rsi_enlisted: string | null;
	rsi_citizen_record: string | null;
	avatar_url: string | null;
	created_at: string;
	last_login_at: string;
}

/** Public user profile returned by GET /api/users/:id */
export interface PublicUser {
	id: number;
	display_name: string;
	balance: number;
	created_at: string;
}

/** Leaderboard row */
export interface LeaderboardRow {
	id: number;
	display_name: string;
	balance: number;
	portfolio_value: number;
}

/** Market status values */
export type MarketStatus = 'pending_review' | 'active' | 'resolved' | 'cancelled' | 'resolution_requested' | 'deadline_passed';

/** Market resolution */
export type Resolution = 'yes' | 'no' | null;

/** Market category */
export type MarketCategory =
	| 'bug_fixes'
	| 'feature_delivery'
	| 'patch_timing'
	| 'community_events'
	| 'meta';

/** Market resolution type */
export type MarketResolutionType = 'binary' | 'date' | 'numeric';

/** Market as returned by the API */
export interface Market {
	id: number;
	title: string;
	description: string;
	category: MarketCategory;
	resolution_criteria: string;
	resolution_deadline: string;
	status: MarketStatus;
	resolution: Resolution;
	created_by: number;
	creator_name: string;
	resolved_by: number | null;
	resolver_name: string | null;
	created_at: string;
	resolved_at: string | null;
	liquidity_param: number;
	yes_shares: number;
	no_shares: number;
	/** Market subtype: binary yes/no, date prediction, or numeric/price prediction */
	resolution_type: MarketResolutionType;
	/** For date markets: ISO date string. For numeric: stringified number (e.g. "200" for $200). */
	resolution_threshold: string | null;
	/** Mod-provided evidence link stored when a market is resolved. */
	resolution_evidence: string | null;
	/** Comment count (included in list view only). */
	comment_count?: number;
}

/** Market with current prices */
export interface MarketWithPrice {
	market: Market;
	yes_price: number; // 1-99 cents
	no_price: number;  // 1-99 cents
	my_position?: { yes_shares: number; no_shares: number };
	/** Total bUEC traded in this market across all time */
	total_volume: number;
	/** Number of unique users who have traded */
	trader_count: number;
	/** Total number of individual trades */
	trade_count: number;
}

/** Resolution request row returned by GET /api/mod/resolution-requests */
export interface ResolutionRequestMarket {
	id: number;
	title: string;
	description: string;
	category: MarketCategory;
	resolution_criteria: string;
	resolution_deadline: string;
	status: MarketStatus;
	resolution: Resolution;
	created_by: number;
	creator_name: string;
	resolved_by: number | null;
	created_at: string;
	resolved_at: string | null;
	liquidity_param: number;
	yes_shares: number;
	no_shares: number;
	// Resolution-request metadata
	requested_by: number;
	requester_name: string;
	request_link: string | null;
	request_note: string | null;
	requested_at: string;
}

/** Paginated market list response */
export interface MarketList {
	total: number;
	markets: (Market & { creator_name: string })[];
	offset: number;
}

/** Trade record */
export interface Trade {
	id: number;
	user_id: number;
	market_id: number;
	side: 'yes' | 'no';
	action: 'buy' | 'sell';
	shares: number;
	cost: number;
	price_at_trade: number;
	created_at: string;
}

/** Trade with market title (from user's trade history) */
export interface TradeWithMarket extends Trade {
	market_title: string;
}

/** Trade with trader name (from market's trade list) */
export interface TradeWithTrader extends Trade {
	trader_name: string;
}

/** User position in a market */
export interface Position {
	user_id: number;
	market_id: number;
	yes_shares: number;
	no_shares: number;
	market_title: string;
	market_status: MarketStatus;
	pool_yes: number;
	pool_no: number;
	liquidity_param: number;
}

/** Trade execution result */
export interface TradeResult {
	TradeID: number;
	Cost: number;
	Shares: number;
	PriceAtTrade: number;
	NewBalance: number;
}

/** Create market request body */
export interface CreateMarketBody {
	title: string;
	description: string;
	category: MarketCategory;
	resolution_criteria: string;
	deadline: string; // RFC3339
	resolution_type?: MarketResolutionType;
	resolution_threshold?: string; // ISO date or numeric value string
}

/** API error response */
export interface ApiError {
	error: string;
}

/** A user comment on a market. */
export interface Comment {
	id: number;
	market_id: number;
	user_id: number;
	author_name: string;
	content: string;
	/**
	 * True only when this is the viewer's own shadow-hidden comment.
	 * Other viewers never receive hidden comments in their response at all.
	 */
	hidden: boolean;
	is_own_hidden: boolean;
	created_at: string;
}

/** Badge awarded to a user */
export interface Badge {
	badge_key: string;
	awarded_at: string;
	title: string;
	description: string;
}

/** A pending moderation report */
export interface Report {
	id: number;
	reporter_id: number;
	reporter_name: string;
	market_id: number;
	market_title: string;
	reason: string;
	status: string;
	created_at: string;
}

/** Price history data point for charting */
export interface PricePoint {
	price_at_trade: number; // 0.0 – 1.0 YES probability
	side: string;           // 'yes' | 'no'
	created_at: string;
}

/** A detected LIVE patch from the RSI Spectrum forum */
export interface DetectedPatch {
	id: number;
	title: string;
	patch_version: string;
	thread_url: string;
	first_seen_at: string;
	notified: number; // 0 = unseen, 1 = seen by mod
}
