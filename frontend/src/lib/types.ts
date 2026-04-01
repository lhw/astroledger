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
	active_badge_key: string;
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

/** Market category */
export type MarketCategory =
	| 'bug_fixes'
	| 'feature_delivery'
	| 'patch_timing'
	| 'community_events'
	| 'meta';

/** Market resolution type */
export type MarketResolutionType = 'binary' | 'date' | 'numeric';

/** One outcome of a market (e.g. YES / NO / specific patch version) */
export interface MarketOutcome {
	id: number;
	market_id: number;
	label: string;
	shares: number;
	sort_order: number;
	/** Current probability-based price (1–99 cents per share) */
	price: number;
}

/** A user's position in a single outcome */
export interface UserOutcomePosition {
	outcome_id: number;
	label: string;
	shares: number;
}

/** Market as returned by the API */
export interface Market {
	id: number;
	title: string;
	description: string;
	category: MarketCategory;
	resolution_criteria: string;
	resolution_deadline: string;
	status: MarketStatus;
	resolved_outcome_id: number | null;
	created_by: number;
	creator_name: string;
	resolved_by: number | null;
	resolver_name: string | null;
	created_at: string;
	resolved_at: string | null;
	liquidity_param: number;
	/** Market subtype: binary yes/no, date prediction, or numeric/price prediction */
	resolution_type: MarketResolutionType;
	/** For date markets: ISO date string. For numeric: stringified number (e.g. "200" for $200). */
	resolution_threshold: string | null;
	/** Mod-provided evidence link stored when a market is resolved. */
	resolution_evidence: string | null;
	/** Comment count (included in list view only). */
	comment_count?: number;
	/** Matching auto-filter rules (shown in mod queue). */
	auto_filter_matches?: string[];
	/** All outcomes for this market */
	outcomes: MarketOutcome[];
}

/** Market with current prices */
export interface MarketWithPrice {
	market: Market;
	/** outcomes with current per-outcome prices (same as market.outcomes, convenience alias) */
	outcomes: MarketOutcome[];
	my_positions: UserOutcomePosition[];
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
	resolved_outcome_id: number | null;
	created_by: number;
	creator_name: string;
	resolved_by: number | null;
	created_at: string;
	resolved_at: string | null;
	liquidity_param: number;
	outcomes: MarketOutcome[];
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

/** A trending market enriched with recent (24h) trading activity */
export interface TrendingMarket extends Market {
	creator_name: string;
	recent_trade_count: number;
	recent_volume: number;
}

/** Trade record */
export interface Trade {
	id: number;
	user_id: number;
	market_id: number;
	outcome_id: number;
	outcome_label: string;
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
	/** Custom outcomes for multi-outcome markets. If omitted, defaults to YES/NO. */
	outcomes?: string[];
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
	author_top_badge?: string;
	content: string;
	/**
	 * True only when this is the viewer's own shadow-hidden comment.
	 * Other viewers never receive hidden comments in their response at all.
	 */
	hidden: boolean;
	created_at: string;
}

/** Badge awarded to a user */
export interface Badge {
	badge_key: string;
	awarded_at: string;
	title: string;
	description: string;
	tier: number;
	cost: number;
	purchasable: boolean;
	/** Insurance tier: '6w', '120w', 'lti', or '' — only present on /me/badges */
	insurance?: string;
}

/** A badge available in the FOMO Store */
export interface StoreBadge {
	badge_key: string;
	title: string;
	description: string;
	tier: number;
	cost: number;
	owned: boolean;
	/** Total stock cap; omitted if unlimited. */
	stock?: number;
	/** Remaining units; omitted if unlimited. */
	remaining_stock?: number;
	/** ISO date after which this badge can no longer be purchased. */
	available_until?: string;
	/** True if available_until has passed. */
	expired: boolean;
	/** Cosmetic insurance tier assigned to this release: '6w', '120w', 'lti', or ''. */
	insurance: string;
}

/** A rank in the Admiral Rank progression */
export interface AdmiralRank {
	badge_key: string;
	title: string;
	description: string;
	tier: number;
	spend_threshold: number;
	owned: boolean;
}

/** Response from GET /api/admiral */
export interface AdmiralRanksResponse {
	ranks: AdmiralRank[];
	lifetime_spend: number;
}

/** A pending moderation report */
export interface Report {
	id: number;
	reporter_id: number;
	reporter_name: string;
	market_id: number;
	market_title: string;
	category: MarketCategory;
	reason: string;
	status: string;
	created_at: string;
}

/** Price history data point for charting */
export interface PricePoint {
	price_at_trade: number; // 0.0 – 1.0 probability
	outcome_label: string;
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

// ── Analytics (GoatCounter proxy) ────────────────────────────────────────

export interface AnalyticsDayStat {
	date: string;
	views: number;
	unique: number;
}

export interface AnalyticsPage {
	path: string;
	title: string;
	views: number;
	unique: number;
}

export interface AnalyticsRef {
	name: string;
	views: number;
}

export interface AnalyticsStat {
	name: string;
	count: number;
}

export interface AnalyticsStats {
	configured: boolean;
	period: string;
	total_views: number;
	total_unique: number;
	daily: AnalyticsDayStat[];
	top_pages: AnalyticsPage[];
	top_refs: AnalyticsRef[];
	browsers: AnalyticsStat[];
	systems: AnalyticsStat[];
	locations: AnalyticsStat[];
	languages: AnalyticsStat[];
}

export interface UserSearchResult {
	id: number;
	display_name: string;
	rsi_handle: string | null;
	balance: number;
}

/** A badge definition from the catalog (admin use) */
export interface BadgeCatalogEntry {
	key: string;
	title: string;
	description: string;
	tier: number;
	purchasable: boolean;
}

/** A badge definition row (admin-created or hardcoded) from badge_definitions table */
export interface AdminBadgeDefinition {
	id: number;
	key: string;
	title: string;
	description: string;
	tier: number;
	icon: string;
	is_hardcoded: boolean;
	purchasable: boolean;
	insurance: string;
	created_at: string;
}

/** A badge release created by an admin */
export interface AdminBadgeRelease {
	id: number;
	badge_key: string;
	title: string;
	description: string;
	tier: number;
	price: number;
	stock: number | null;
	released_at: string;
	expires_at: string | null;
	active: boolean;
	notes: string | null;
	insurance: string;
	created_at: string;
	/** Number of users who have purchased this badge so far. */
	sold: number;
}

/** User position enriched with cost basis and resolved market info */
export interface Position {
	user_id: number;
	market_id: number;
	outcome_id: number;
	outcome_label: string;
	shares: number;
	market_title: string;
	market_status: MarketStatus;
	liquidity_param: number;
	outcomes: MarketOutcome[];
	/** Net bUEC paid for this position (buys minus sells). */
	cost_basis: number;
	/** Winning outcome ID once market resolves; null while active. */
	resolved_outcome_id: number | null;
}

/** A bot API token entry (secret value is never returned in list calls). */
export interface BotApiToken {
	id: number;
	name: string;
	token_prefix: string;
	can_read: boolean;
	can_trade: boolean;
	created_at: string;
	last_used_at: string | null;
}

/** Response when creating a bot token (includes one-time full token). */
export interface BotApiTokenCreateResponse extends BotApiToken {
	token: string;
}
