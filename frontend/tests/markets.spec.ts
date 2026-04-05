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
		await expect(page.getByText('1\u20132 of 2')).toBeVisible();
	});

	test('shows create market button when logged in', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMarkets(page);
		await page.goto('/markets');
		await expect(page.getByRole('link', { name: /submit market/i })).toBeVisible();
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
					market: {
						...MARKETS_RESPONSE.markets[0],
						outcomes: [
							{ id: 1, market_id: 1, label: 'YES', shares: 0, sort_order: 0, price: 42 },
							{ id: 2, market_id: 1, label: 'NO', shares: 0, sort_order: 1, price: 58 }
						]
					},
					outcomes: [
						{ id: 1, market_id: 1, label: 'YES', shares: 0, sort_order: 0, price: 42 },
						{ id: 2, market_id: 1, label: 'NO', shares: 0, sort_order: 1, price: 58 }
					],
					my_positions: [],
					total_volume: 0,
					trader_count: 0,
					trade_count: 0
				})
			});
		});
		await page.route('/api/markets/1/history', (route) => {
			route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
		});
		await page.route('/api/markets/1/comments', (route) => {
			route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
		});
	});

	test('shows market title', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByRole('heading', { name: 'Will the cargo elevator be fixed before 4.0?' })).toBeVisible();
	});

	test('shows yes and no prices', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByText('42%').first()).toBeVisible();
		await expect(page.getByText('58%').first()).toBeVisible();
	});

	test('shows buy widget when logged in', async ({ page }) => {
		await page.goto('/markets/1');
		await expect(page.getByRole('button', { name: 'Buy', exact: true })).toBeVisible();
	});

	test('opens and submits the report market form', async ({ page }) => {
		await page.route('/api/reports', async (route) => {
			await route.fulfill({ status: 200, contentType: 'application/json', body: '{"id":1}' });
		});

		await page.goto('/markets/1');
		await page.getByRole('button', { name: /report this market/i }).click();
		await page.getByLabel('Reason').fill('Duplicate market and stale resolution criteria.');

		const requestPromise = page.waitForRequest('**/api/reports');
		await page.getByRole('button', { name: /submit report/i }).click();
		const request = await requestPromise;
		const payload = request.postDataJSON() as { reason?: string; market_id?: number };

		expect(payload.reason).toBe('Duplicate market and stale resolution criteria.');
		expect(payload.market_id).toBe(1);
		await expect(page.getByRole('button', { name: /report this market/i })).toBeVisible();
	});

	test('posts a new comment into the discussion', async ({ page }) => {
		const comments = [] as Array<{ id: number; content: string; author_name: string; hidden: boolean; created_at: string }>;

		await page.unroute('/api/markets/1/comments');
		await page.route('/api/markets/1/comments', async (route) => {
			if (route.request().method() === 'POST') {
				const payload = route.request().postDataJSON() as { content?: string };
				const created = {
					id: comments.length + 1,
					market_id: 1,
					user_id: 1,
					author_name: 'TestPilot',
					author_top_badge: '',
					content: payload.content ?? '',
					hidden: false,
					created_at: '2026-04-05T12:00:00Z'
				};
				comments.push(created);
				await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(created) });
				return;
			}

			await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(comments) });
		});

		await page.goto('/markets/1');
		await page.getByPlaceholder(/add a comment/i).fill('Looks fixable in 4.0 if the patch notes stop lying.');
		await page.getByRole('button', { name: 'Post' }).click();

		await expect(page.getByText('Looks fixable in 4.0 if the patch notes stop lying.')).toBeVisible();
		await expect(page.getByText(/Discussion \(1\)/)).toBeVisible();
	});
});
