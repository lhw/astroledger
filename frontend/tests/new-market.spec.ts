import { test, expect } from '@playwright/test';
import { mockMe, USER_LOGGED_IN } from './helpers/mock-api';

test.describe('Create Market page', () => {
	test('shows login prompt when not authenticated', async ({ page }) => {
		await mockMe(page, null);
		await page.goto('/markets/new');
		await expect(page.getByText(/you must be logged in/i)).toBeVisible();
		await expect(page.getByRole('link', { name: /login/i })).toBeVisible();
	});

	test('shows form fields when authenticated', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await page.goto('/markets/new');
		await expect(page.getByRole('heading', { name: /create a market/i })).toBeVisible();
		await expect(page.getByPlaceholder(/will \[bug\]/i)).toBeVisible();
		await expect(page.getByText(/category/i)).toBeVisible();
		await expect(page.getByText(/resolution criteria/i, { exact: false })).toBeVisible();
	});

	test('submit button is disabled when title is empty', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await page.goto('/markets/new');
		const submit = page.getByRole('button', { name: /submit for review/i });
		await expect(submit).toBeDisabled();
	});

	test('submit button enables after filling required fields', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await page.goto('/markets/new');

		await page.getByPlaceholder(/will \[bug\]/i).fill('Will cargo be fixed?');
		// Fill the deadline date input
		const tomorrow = new Date();
		tomorrow.setDate(tomorrow.getDate() + 2);
		const dateStr = tomorrow.toISOString().slice(0, 10);
		await page.locator('input[type="date"]').first().fill(dateStr);

		const submit = page.getByRole('button', { name: /submit for review/i });
		await expect(submit).toBeEnabled();
	});

	test('shows API error when submission fails', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await page.route('/api/markets', (route) => {
			if (route.request().method() === 'POST') {
				route.fulfill({
					status: 422,
					contentType: 'application/json',
					body: '{"error":"title too similar to existing market"}'
				});
			} else {
				route.continue();
			}
		});

		await page.goto('/markets/new');
		await page.getByPlaceholder(/will \[bug\]/i).fill('Will cargo be fixed?');
		const tomorrow = new Date();
		tomorrow.setDate(tomorrow.getDate() + 2);
		await page.locator('input[type="date"]').first().fill(tomorrow.toISOString().slice(0, 10));

		await page.getByRole('button', { name: /submit for review/i }).click();
		await expect(page.getByText(/title too similar/i)).toBeVisible();
	});
});
