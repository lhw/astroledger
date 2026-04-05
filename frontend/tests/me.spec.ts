import { test, expect } from '@playwright/test';
import { mockMe, USER_LOGGED_IN } from './helpers/mock-api';
import type { Page } from '@playwright/test';

/** Mock all API endpoints that /me calls in parallel. */
async function mockMePageData(
	page: Page,
	opts: {
		positions?: unknown[];
		trades?: unknown[];
		badges?: unknown[];
		botTokens?: unknown[];
	} = {}
) {
	const { positions = [], trades = [], badges = [], botTokens = [] } = opts;
	await page.route('/api/me/positions', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(positions) })
	);
	await page.route('/api/me/trades*', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(trades) })
	);
	await page.route('/api/me/badges', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(badges) })
	);
	await page.route('/api/bot/tokens', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(botTokens) })
	);
}

test.describe('My Profile page', () => {
	test('shows login prompt when not authenticated', async ({ page }) => {
		await mockMe(page, null);
		await page.goto('/me');
		await expect(page.getByRole('heading', { name: /not logged in/i })).toBeVisible();
		await expect(page.getByRole('link', { name: /login/i })).toBeVisible();
	});

	test('shows username and balance when authenticated', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page);
		await page.goto('/me');
		await expect(page.getByRole('heading', { name: USER_LOGGED_IN.display_name })).toBeVisible();
		// Balance appears in the profile section (also in nav, so use first match)
		await expect(page.getByText('1,000').first()).toBeVisible();
	});

	test('shows "Open Positions" section when authenticated', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page);
		await page.goto('/me');
		await expect(page.getByRole('heading', { name: /open positions/i })).toBeVisible();
	});

	test('shows join date', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page);
		await page.goto('/me');
		// USER_LOGGED_IN.created_at = '2024-01-01T00:00:00Z'
		await expect(page.getByText(/joined/i)).toBeVisible();
	});

	test('shows moderator badge for moderators', async ({ page }) => {
		const mod = { ...USER_LOGGED_IN, is_moderator: true };
		await mockMe(page, mod);
		await mockMePageData(page);
		await page.goto('/me');
		await expect(page.getByText('Moderator')).toBeVisible();
	});

	test('shows logout button on the profile card', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page);
		await page.goto('/me');
		// Multiple logout buttons exist (nav + profile card); check at least one is visible
		await expect(page.getByRole('button', { name: /logout/i }).first()).toBeVisible();
	});
});

// ─── Badge Hangar ─────────────────────────────────────────────────────────────

const HANGAR_BADGE_LTI = {
	badge_key: 'aurora_pilot',
	title: 'Aurora Pilot',
	description: 'Started small, dreamed big.',
	tier: 2,
	cost: 750,
	purchasable: true,
	insurance: 'lti',
	awarded_at: '2026-01-01T00:00:00Z'
};

const HANGAR_BADGE_6W = {
	badge_key: 'space_whale',
	title: 'Space Whale',
	description: 'Your wallet is canon-sized.',
	tier: 3,
	cost: 1000,
	purchasable: true,
	insurance: '6w',
	awarded_at: '2026-02-01T00:00:00Z'
};

const HANGAR_BADGE_120W = {
	badge_key: 'verse_veteran',
	title: "'Verse Veteran",
	description: 'A seasoned survivor.',
	tier: 3,
	cost: 1200,
	purchasable: true,
	insurance: '120w',
	awarded_at: '2026-03-01T00:00:00Z'
};

const HANGAR_BADGE_NO_INS = {
	badge_key: 'citizen_backer',
	title: 'Citizen Backer',
	description: 'A true believer.',
	tier: 1,
	cost: 500,
	purchasable: true,
	insurance: '',
	awarded_at: '2026-04-01T00:00:00Z'
};

const HANGAR_BADGE_EARNED = {
	badge_key: 'first_trade',
	title: 'First Trade',
	description: 'Made their first bet.',
	tier: 1,
	cost: 0,
	purchasable: false,
	insurance: undefined,
	awarded_at: '2026-01-15T00:00:00Z'
};

test.describe('Badge Hangar (my profile)', () => {
	test('shows empty state when user has no badges', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page, { badges: [] });
		await page.goto('/me');
		await expect(page.getByText(/no badges yet/i)).toBeVisible();
	});

	test('shows FOMO Store link in empty state', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page, { badges: [] });
		await page.goto('/me');
			// Two FOMO Store links: the header link "FOMO Store →" and the inline one in the empty state.
			// Check the inline link in the empty state text exactly.
			await expect(page.getByRole('link', { name: 'FOMO Store', exact: true })).toBeVisible();
		await mockMePageData(page, { badges: [HANGAR_BADGE_LTI, HANGAR_BADGE_NO_INS] });
		await page.goto('/me');
		await expect(page.locator('.hangar-row')).toHaveCount(2);
	});

	test('shows badge title in each hangar row', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
		await page.goto('/me');
		await expect(page.getByText('Aurora Pilot')).toBeVisible();
	});

	test('shows badge description in each hangar row', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
		await page.goto('/me');
		await expect(page.getByText('Started small, dreamed big.')).toBeVisible();
	});

	test('shows award date for each badge', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
		await page.goto('/me');
		// Locale-formatted date — just check it's non-empty
		await expect(page.locator('.hangar-date').first()).not.toBeEmpty();
	});

	test.describe('Insurance pips', () => {
		test('shows LTI pip for badge with lti insurance', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
			await page.goto('/me');
			await expect(page.locator('.ins-pip.ins-lti')).toBeVisible();
			await expect(page.locator('.ins-pip.ins-lti')).toHaveText('LTI');
		});

		test('shows 6 Weeks pip for badge with 6w insurance', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockMePageData(page, { badges: [HANGAR_BADGE_6W] });
			await page.goto('/me');
			await expect(page.locator('.ins-pip.ins-6w')).toBeVisible();
			await expect(page.locator('.ins-pip.ins-6w')).toHaveText('6 Weeks');
		});

		test('shows 120 Weeks pip for badge with 120w insurance', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockMePageData(page, { badges: [HANGAR_BADGE_120W] });
			await page.goto('/me');
			await expect(page.locator('.ins-pip.ins-120w')).toBeVisible();
			await expect(page.locator('.ins-pip.ins-120w')).toHaveText('120 Weeks');
		});

		test('shows No Ins. pip for purchasable badge with no insurance', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockMePageData(page, { badges: [HANGAR_BADGE_NO_INS] });
			await page.goto('/me');
			await expect(page.locator('.ins-pip.ins-none')).toBeVisible();
			await expect(page.locator('.ins-pip.ins-none')).toHaveText('No Ins.');
		});

		test('shows no insurance pip for earned (non-purchasable) badge', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockMePageData(page, { badges: [HANGAR_BADGE_EARNED] });
			await page.goto('/me');
			await expect(page.locator('.ins-pip')).toHaveCount(0);
		});
	});

	test.describe('Active badge toggle', () => {
		test('shows Set Active button on inactive badge', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: '' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
			await page.goto('/me');
			await expect(page.getByRole('button', { name: 'Set Active' })).toBeVisible();
		});

		test('shows ✓ Active button on active badge', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: 'aurora_pilot' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
			await page.goto('/me');
			await expect(page.getByRole('button', { name: '✓ Active' })).toBeVisible();
		});

		test('active row has .active CSS class', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: 'aurora_pilot' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });
			await page.goto('/me');
			await expect(page.locator('.hangar-row.active')).toHaveCount(1);
		});

		test('clicking Set Active calls PUT /api/me/badge with badge_key', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: '' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });

			const requestPromise = page.waitForRequest('/api/me/badge');
			await page.route('/api/me/badge', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ active_badge_key: 'aurora_pilot' }) })
			);

			await page.goto('/me');
			await page.getByRole('button', { name: 'Set Active' }).click();

			const req = await requestPromise;
			const capturedBody = JSON.parse(req.postData() ?? '{}') as Record<string, unknown>;
			expect(capturedBody.badge_key).toBe('aurora_pilot');
		});

		test('button changes to ✓ Active after clicking Set Active', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: '' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });

			await page.route('/api/me/badge', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ active_badge_key: 'aurora_pilot' }) })
			);

			await page.goto('/me');
			await page.getByRole('button', { name: 'Set Active' }).click();
			await expect(page.getByRole('button', { name: '✓ Active' })).toBeVisible();
		});

		test('clicking ✓ Active for active badge deactivates it', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: 'aurora_pilot' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI] });

			const requestPromise = page.waitForRequest('/api/me/badge');
			await page.route('/api/me/badge', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ active_badge_key: '' }) })
			);

			await page.goto('/me');
			await page.getByRole('button', { name: '\u2713 Active' }).click();

			// pickBadge sends empty string to unset
			const req = await requestPromise;
			const capturedBody = JSON.parse(req.postData() ?? '{}') as Record<string, unknown>;
			expect(capturedBody.badge_key).toBe('');
		});

		test('multiple badges: only active badge shows ✓ Active', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, active_badge_key: 'aurora_pilot' });
			await mockMePageData(page, { badges: [HANGAR_BADGE_LTI, HANGAR_BADGE_NO_INS] });
			await page.goto('/me');
			await expect(page.getByRole('button', { name: '✓ Active' })).toHaveCount(1);
			await expect(page.getByRole('button', { name: 'Set Active' })).toHaveCount(1);
		});
	});
});
