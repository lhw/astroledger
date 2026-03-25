/** User profile returned by GET /api/me */
export interface User {
	id: number;
	display_name: string;
	email: string;
	balance: number;
	is_moderator: boolean;
	is_admin: boolean;
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
export type MarketStatus = 'pending_review' | 'active' | 'resolved' | 'cancelled';

/** Market resolution */
export type Resolution = 'yes' | 'no' | null;

/** Market category */
export type MarketCategory =
	| 'bug_fixes'
	| 'feature_delivery'
	| 'patch_timing'
	| 'community_events'
	| 'meta';

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
	created_at: string;
	resolved_at: string | null;
	liquidity_param: number;
	yes_shares: number;
	no_shares: number;
}

/** Market with current prices */
export interface MarketWithPrice {
	market: Market;
	yes_price: number; // 1-99 cents
	no_price: number;  // 1-99 cents
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
}

/** API error response */
export interface ApiError {
	error: string;
}
