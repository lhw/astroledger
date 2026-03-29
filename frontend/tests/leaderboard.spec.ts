import { test, expect } from '@playwright/test';
import { mockMe, mockLeaderboard, LEADERBOARD_RESPONSE, USER_LOGGED_IN } from './helpers/mock-api';

test.describe('Leaderboard page', () => {
	test('shows page title and leaderboard rows', async ({ page }) => {
		await mockMe(page, null);
		await mockLeaderboard(page);
		await page.goto('/leaderboard');
		await expect(page.getByRole('heading', { name: /Leaderboard/i })).toBeVisible();
		await expect(page.getByText('TestPilot')).toBeVisible();
		await expect(page.getByText('SpaceBaron')).toBeVisible();
	});

	test('shows portfolio and balance values', async ({ page }) => {
		await mockMe(page, null);
		await mockLeaderboard(page);
		await page.goto('/leaderboard');
		// LEADERBOARD_RESPONSE has portfolio_value 1350 for first entry
		await expect(page.getByText('1,350')).toBeVisible();
	});

	test('shows empty state when no data', async ({ page }) => {
		await mockMe(page, null);
		await mockLeaderboard(page, []);
		await page.goto('/leaderboard');
		await expect(page.getByText(/no data yet/i)).toBeVisible();
	});

	test('shows error on API failure', async ({ page }) => {
		await mockMe(page, null);
		await page.route(/\/api\/leaderboard/, (route) =>
			route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"db down"}' })
		);
		await page.goto('/leaderboard');
		await expect(page.getByText(/db down/i)).toBeVisible();
	});

	test('shows gold medal for first place', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockLeaderboard(page);
		await page.goto('/leaderboard');
		await expect(page.getByText('🥇')).toBeVisible();
		await expect(page.getByText('🥈')).toBeVisible();
	});
});
