import { test, expect } from '@playwright/test';
import { mockMe, mockUserData, USER_LOGGED_IN } from './helpers/mock-api';
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
