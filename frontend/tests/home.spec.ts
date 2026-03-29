import { test, expect } from '@playwright/test';
import { mockMe, mockMarkets, USER_LOGGED_IN } from './helpers/mock-api';

test.describe('Home page', () => {
	test('shows AstroLedger heading', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/');
		await expect(page.getByRole('heading', { name: /AstroLedger/i })).toBeVisible();
	});

	test('shows login button when not authenticated', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/');
		await expect(page.getByRole('button', { name: 'Login with SCID' }).first()).toBeVisible();
	});

	test('shows user balance when authenticated', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockMarkets(page);
		await page.goto('/');
		await expect(page.getByRole('link', { name: 'TestPilot' }).first()).toBeVisible();
		await expect(page.getByText('1,000 bUEC')).toBeVisible();
	});

	test('shows featured markets on home page', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/');
		await expect(page.getByText('Will the cargo elevator be fixed before 4.0?')).toBeVisible();
	});

	test('navigates to markets page from header', async ({ page }) => {
		await mockMe(page, null);
		await mockMarkets(page);
		await page.goto('/');
		await page.getByRole('link', { name: /^markets$/i }).first().click();
		await expect(page).toHaveURL('/markets');
	});
});
