import type { Page } from '@playwright/test';
import type { User, LeaderboardRow, MarketList, ResolutionRequestMarket, Report } from '../../src/lib/types';

// --- Fixtures ---

export const USER_LOGGED_IN: User = {
	id: 1,
	display_name: 'TestPilot',
	email: 'test@example.com',
	balance: 1000,
	is_moderator: false,
	is_admin: false,
	created_at: '2024-01-01T00:00:00Z',
	last_login_at: '2024-01-10T00:00:00Z'
};

export const USER_MODERATOR: User = {
	...USER_LOGGED_IN,
	id: 2,
	display_name: 'ModBot9000',
	is_moderator: true
};

export const MARKETS_RESPONSE: MarketList = {
	total: 2,
	offset: 0,
	markets: [
		{
			id: 1,
			title: 'Will the cargo elevator be fixed before 4.0?',
			description: 'Has been broken for a while.',
			category: 'bug_fixes',
			resolution_criteria: 'Patch notes confirm the fix.',
			resolution_deadline: '2025-12-31T23:59:59Z',
			status: 'active',
			resolution: null,
			created_by: 1,
			creator_name: 'TestPilot',
			resolved_by: null,
			created_at: '2024-06-01T00:00:00Z',
			resolved_at: null,
			liquidity_param: 100,
			yes_shares: 0,
			no_shares: 0,
			resolution_type: 'binary',
			resolution_threshold: null
		},
		{
			id: 2,
			title: 'Will 4.1 ship before end of Q2?',
			description: 'Tracking the release window.',
			category: 'patch_timing',
			resolution_criteria: 'Live build deployed before July 1.',
			resolution_deadline: '2025-06-30T23:59:59Z',
			status: 'active',
			resolution: null,
			created_by: 1,
			creator_name: 'TestPilot',
			resolved_by: null,
			created_at: '2024-06-02T00:00:00Z',
			resolved_at: null,
			liquidity_param: 100,
			yes_shares: 0,
			no_shares: 0,
			resolution_type: 'binary',
			resolution_threshold: null
		}
	]
};

export const LEADERBOARD_RESPONSE: LeaderboardRow[] = [
	{ id: 1, display_name: 'TestPilot', balance: 1200, portfolio_value: 1350 },
	{ id: 2, display_name: 'SpaceBaron', balance: 800, portfolio_value: 1100 }
];

// --- Mock helpers ---

/** Mock GET /api/me — pass null to simulate logged-out (returns 401). */
export async function mockMe(page: Page, user: User | null) {
	await page.route('/api/me', (route) => {
		if (user === null) {
			route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ error: 'Unauthorized' }) });
		} else {
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(user) });
		}
	});
}

/** Mock GET /api/markets with optional filter. */
export async function mockMarkets(page: Page, response: MarketList = MARKETS_RESPONSE) {
	await page.route(/\/api\/markets(\?.*)?$/, (route) => {
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(response) });
	});
}

/** Mock GET /api/leaderboard */
export async function mockLeaderboard(page: Page, rows: LeaderboardRow[] = LEADERBOARD_RESPONSE) {
	await page.route(/\/api\/leaderboard/, (route) => {
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(rows) });
	});
}

/** Mock GET /api/me/positions and /api/me/trades */
export async function mockUserData(page: Page) {
	await page.route('/api/me/positions', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
	);
	await page.route('/api/me/trades*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
	);
}

export const RESOLUTION_REQUEST: ResolutionRequestMarket = {
	id: 10,
	title: 'Will quantum fuel fix before 4.2?',
	description: 'Been broken for years.',
	category: 'bug_fixes',
	resolution_criteria: 'Patch notes confirm fix.',
	resolution_deadline: '2026-12-31T23:59:59Z',
	status: 'resolution_requested',
	resolution: null,
	created_by: 1,
	creator_name: 'TestPilot',
	resolved_by: null,
	created_at: '2026-01-01T00:00:00Z',
	resolved_at: null,
	liquidity_param: 100,
	yes_shares: 10,
	no_shares: 5,
	requested_by: 1,
	requester_name: 'TestPilot',
	request_link: 'https://robertsspaceindustries.com/patch/4.2',
	request_note: 'Confirmed in PTU patch notes.',
	requested_at: '2026-03-20T10:00:00Z'
};

export const PENDING_REPORT: Report = {
	id: 1,
	reporter_id: 1,
	reporter_name: 'TestPilot',
	market_id: 10,
	market_title: 'Will quantum fuel fix before 4.2?',
	reason: 'This market topic duplicates an existing one.',
	status: 'pending',
	created_at: '2026-03-21T09:00:00Z'
};

/** Mock all three mod queue endpoints. */
export async function mockModQueue(
	page: Page,
	opts: {
		pending?: (typeof MARKETS_RESPONSE.markets)[number][];
		resolutionRequests?: ResolutionRequestMarket[];
		reports?: Report[];
	} = {}
) {
	const pending = opts.pending ?? [];
	const resolutionRequests = opts.resolutionRequests ?? [];
	const reports = opts.reports ?? [];

	await page.route('/api/mod/markets', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(pending) })
	);
	await page.route('/api/mod/resolution-requests', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(resolutionRequests) })
	);
	await page.route('/api/mod/reports', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(reports) })
	);
}
