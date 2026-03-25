import { test, expect } from '@playwright/test';
import { mockMe, mockMarkets, MARKETS_RESPONSE, USER_LOGGED_IN } from './helpers/mock-api';

test.describe('Markets list page', () => {
	test('shows all active markets', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/markets');
		await expect(page.getByText('Will the cargo elevator be fixed before 4.0?')).toBeVisible();
		await expect(page.getByText('Will 4.1 ship before end of Q2?')).toBeVisible();
	});

	test('shows market count', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/markets');
		await expect(page.getByText('2')).toBeVisible();
	});

	test('shows create market button when logged in', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMarkets(page);
		await page.goto('/markets');
		await expect(page.getByRole('link', { name: /create/i })).toBeVisible();
	});

	test('market link navigates to detail page', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		// Mock the market detail endpoint too
		await page.route('/api/markets/1', (route) => {
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					market: MARKETS_RESPONSE.markets[0],
					yes_price: 50,
					no_price: 50
				})
			});
		});
		await page.goto('/markets');
		await page.getByText('Will the cargo elevator be fixed before 4.0?').click();
		await expect(page).toHaveURL('/markets/1');
	});

	test('shows empty state when no markets', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page, { total: 0, offset: 0, markets: [] });
		await page.goto('/markets');
		await expect(page.getByText(/no markets/i)).toBeVisible();
	});
});

test.describe('Market detail page', () => {
	test.beforeEach(async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await page.route('/api/markets/1', (route) => {
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					market: MARKETS_RESPONSE.markets[0],
					yes_price: 42,
					no_price: 58
				})
			});
		});
	});

	test('shows market title', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByRole('heading', { name: 'Will the cargo elevator be fixed before 4.0?' })).toBeVisible();
	});

	test('shows yes and no prices', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByText('42¢')).toBeVisible();
		await expect(page.getByText('58¢')).toBeVisible();
	});

	test('shows buy widget when logged in', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByRole('button', { name: /buy/i })).toBeVisible();
	});
});
