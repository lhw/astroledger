/**
 * Playwright tests for the moderator queue: pending markets, resolution
 * requests, reports, and related actions.
 */
import { test, expect } from '@playwright/test';
import {
	mockMe,
	mockModQueue,
	USER_MODERATOR,
	USER_LOGGED_IN,
	MARKETS_RESPONSE,
	RESOLUTION_REQUEST,
	PENDING_REPORT
} from './helpers/mock-api';

// ─── Helper: standard mod queue setup ────────────────────────────────────────

async function goToModQueue(page: Parameters<typeof mockMe>[0]) {
	await page.goto('/mod');
	// Wait for the page heading — always present regardless of auth state or errors.
	await page.waitForSelector('h1');
}

// ─── Access control ───────────────────────────────────────────────────────────

test.describe('Mod queue — access control', () => {
	test('shows access-denied message to non-moderator', async ({ page }) => {
		await mockMe(page, USER_LOGGED_IN);
		await mockModQueue(page);
		await goToModQueue(page);
		await expect(page.getByText(/must be a moderator/i)).toBeVisible();
	});

	test('renders queue for moderator', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page);
		await goToModQueue(page);
		await expect(page.getByRole('tab', { name: /Pending Review/i })).toBeVisible();
		await expect(page.getByRole('tab', { name: /Resolution Requests/i })).toBeVisible();
		await expect(page.getByRole('tab', { name: /Reports/i })).toBeVisible();
	});
});

// ─── Tab: Pending Review ─────────────────────────────────────────────────────

test.describe('Mod queue — Pending Review tab', () => {
	test('shows empty state when no pending markets', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { pending: [] });
		await goToModQueue(page);
		await expect(page.getByText(/no markets pending/i)).toBeVisible();
	});

	test('shows badge count when pending markets exist', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		const pendingMarket = { ...MARKETS_RESPONSE.markets[0], status: 'pending_review' as const, creator_name: 'TestPilot' };
		await mockModQueue(page, { pending: [pendingMarket] });
		await goToModQueue(page);
		// Badge showing "1" on the Pending Review tab.
		const tab = page.locator('button', { hasText: 'Pending Review' });
		await expect(tab.locator('span', { hasText: '1' })).toBeVisible();
	});

	test('shows pending market title and creator', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		const pendingMarket = {
			...MARKETS_RESPONSE.markets[0],
			title: 'Will cargo elevator be fixed?',
			status: 'pending_review' as const,
			creator_name: 'TestPilot'
		};
		await mockModQueue(page, { pending: [pendingMarket] });
		await goToModQueue(page);
		await expect(page.getByText('Will cargo elevator be fixed?')).toBeVisible();
		await expect(page.getByText('TestPilot')).toBeVisible();
	});

	test('approve button calls approve API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		const pendingMarket = { ...MARKETS_RESPONSE.markets[0], status: 'pending_review' as const, creator_name: 'TestPilot' };
		await mockModQueue(page, { pending: [pendingMarket] });

		let approveCallCount = 0;
		await page.route(`/api/mod/markets/${pendingMarket.id}/approve`, (route) => {
			approveCallCount++;
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"active"}' });
		});

		await goToModQueue(page);
		await page.getByRole('button', { name: /^approve$/i }).first().click();
		// After action, queue reloads — check API was called.
		await page.waitForTimeout(200);
		expect(approveCallCount).toBe(1);
	});

	test('reject button calls reject API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		const pendingMarket = { ...MARKETS_RESPONSE.markets[0], status: 'pending_review' as const, creator_name: 'TestPilot' };
		await mockModQueue(page, { pending: [pendingMarket] });

		let rejectCallCount = 0;
		await page.route(`/api/mod/markets/${pendingMarket.id}/reject`, (route) => {
			rejectCallCount++;
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"cancelled"}' });
		});

		await goToModQueue(page);
		await page.getByRole('button', { name: /^reject$/i }).first().click();
		await page.waitForTimeout(200);
		expect(rejectCallCount).toBe(1);
	});
});

// ─── Tab: Resolution Requests ─────────────────────────────────────────────────

test.describe('Mod queue — Resolution Requests tab', () => {
	test('shows empty state when no requests', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { resolutionRequests: [] });
		await goToModQueue(page);
		await page.getByRole('tab', { name: /Resolution Requests/i }).click();
		await expect(page.getByText(/no resolution requests/i)).toBeVisible();
	});

	test('auto-switches to resolution tab when only that has items', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { pending: [], resolutionRequests: [RESOLUTION_REQUEST] });
		await goToModQueue(page);
		// The page should auto-switch, so the resolution request should be visible.
		await expect(page.getByText(RESOLUTION_REQUEST.title)).toBeVisible();
	});

	test('shows resolution request details', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { resolutionRequests: [RESOLUTION_REQUEST] });
		await goToModQueue(page);
		await page.getByRole('tab', { name: /Resolution Requests/i }).click();

		await expect(page.getByText(RESOLUTION_REQUEST.title)).toBeVisible();
		await expect(page.getByText(new RegExp(`Requested by.*${RESOLUTION_REQUEST.requester_name}`, 'i'))).toBeVisible();
		// Evidence link.
		await expect(page.getByRole('link', { name: RESOLUTION_REQUEST.request_link! })).toBeVisible();
		// Note text.
		await expect(page.getByText(RESOLUTION_REQUEST.request_note!)).toBeVisible();
	});

	test('shows badge count on resolution tab', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { resolutionRequests: [RESOLUTION_REQUEST] });
		await goToModQueue(page);
		const tab = page.locator('button', { hasText: 'Resolution Requests' });
		await expect(tab.locator('span', { hasText: '1' })).toBeVisible();
	});

	test('Resolve YES calls the resolve API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { pending: [], resolutionRequests: [RESOLUTION_REQUEST] });

		let resolveBody: unknown = null;
		await page.route(`/api/mod/markets/${RESOLUTION_REQUEST.id}/resolve`, (route) => {
			resolveBody = JSON.parse(route.request().postData() ?? '{}');
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"resolved"}' });
		});

		await goToModQueue(page);
		await page.getByRole('button', { name: /Resolve YES/i }).click();
		await page.waitForTimeout(200);
		expect(resolveBody).toMatchObject({ winning_outcome_id: 19 });
	});

	test('Resolve NO calls the resolve API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { pending: [], resolutionRequests: [RESOLUTION_REQUEST] });

		let resolveBody: unknown = null;
		await page.route(`/api/mod/markets/${RESOLUTION_REQUEST.id}/resolve`, (route) => {
			resolveBody = JSON.parse(route.request().postData() ?? '{}');
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"resolved"}' });
		});

		await goToModQueue(page);
		await page.getByRole('button', { name: /Resolve NO/i }).click();
		await page.waitForTimeout(200);
		expect(resolveBody).toMatchObject({ winning_outcome_id: 20 });
	});

	test('Deny calls deny-resolution API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { pending: [], resolutionRequests: [RESOLUTION_REQUEST] });

		let denyCalled = false;
		await page.route(`/api/mod/markets/${RESOLUTION_REQUEST.id}/deny-resolution`, (route) => {
			denyCalled = true;
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"active"}' });
		});

		await goToModQueue(page);
		await page.getByRole('button', { name: /deny/i }).click();
		await page.waitForTimeout(200);
		expect(denyCalled).toBe(true);
	});
});

// ─── Tab: Reports ─────────────────────────────────────────────────────────────

test.describe('Mod queue — Reports tab', () => {
	test('shows empty state when no reports', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { reports: [] });
		await goToModQueue(page);
		await page.getByRole('tab', { name: /Reports/i }).click();
		await expect(page.getByText(/no pending reports/i)).toBeVisible();
	});

	test('shows badge count on reports tab', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { reports: [PENDING_REPORT] });
		await goToModQueue(page);
		const tab = page.locator('button', { hasText: 'Reports' });
		await expect(tab.locator('span', { hasText: '1' })).toBeVisible();
	});

	test('shows report details in reports tab', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { reports: [PENDING_REPORT] });
		await goToModQueue(page);
		await page.getByRole('tab', { name: /Reports/i }).click();

		await expect(page.getByText(PENDING_REPORT.reporter_name).last()).toBeVisible();
		await expect(page.getByText(PENDING_REPORT.market_title)).toBeVisible();
		await expect(page.getByText(PENDING_REPORT.reason)).toBeVisible();
	});

	test('Reviewed button calls review API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { reports: [PENDING_REPORT] });

		let reviewCalled = false;
		await page.route(`/api/mod/reports/${PENDING_REPORT.id}/review`, (route) => {
			reviewCalled = true;
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"reviewed"}' });
		});

		await goToModQueue(page);
		await page.getByRole('tab', { name: /Reports/i }).click();
		await page.getByRole('button', { name: /reviewed/i }).click();
		await page.waitForTimeout(200);
		expect(reviewCalled).toBe(true);
	});

	test('Dismiss button calls dismiss API', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		await mockModQueue(page, { reports: [PENDING_REPORT] });

		let dismissCalled = false;
		await page.route(`/api/mod/reports/${PENDING_REPORT.id}/dismiss`, (route) => {
			dismissCalled = true;
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"dismissed"}' });
		});

		await goToModQueue(page);
		await page.getByRole('tab', { name: /Reports/i }).click();
		await page.getByRole('button', { name: /dismiss/i }).click();
		await page.waitForTimeout(200);
		expect(dismissCalled).toBe(true);
	});
});

// ─── Error handling ───────────────────────────────────────────────────────────

test.describe('Mod queue — error handling', () => {
	test('shows server error message without ApiClientError prefix', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		// One endpoint returns an error; mock the rest to avoid proxy failures.
		await page.route('/api/mod/markets/deadline-passed', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
		);
		await page.route('/api/patches', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '{"patches":[]}' })
		);
		await page.route('/api/mod/markets', (route) =>
			route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"database error"}' })
		);
		await page.route('/api/mod/resolution-requests', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
		);
		await page.route('/api/mod/reports', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
		);
		await goToModQueue(page);
		// Should show "database error" not "ApiClientError: database error".
		await expect(page.getByText('database error')).toBeVisible();
		await expect(page.getByText(/ApiClientError/)).toHaveCount(0);
	});

	test('action error shows clean message', async ({ page }) => {
		await mockMe(page, USER_MODERATOR);
		const pendingMarket = { ...MARKETS_RESPONSE.markets[0], status: 'pending_review' as const, creator_name: 'TestPilot' };
		await mockModQueue(page, { pending: [pendingMarket] });

		await page.route(`/api/mod/markets/${pendingMarket.id}/approve`, (route) =>
			route.fulfill({ status: 422, contentType: 'application/json', body: '{"error":"market is not pending_review"}' })
		);

		await goToModQueue(page);
		await page.getByRole('button', { name: /^approve$/i }).first().click();
		await expect(page.getByText('market is not pending_review')).toBeVisible();
		await expect(page.getByText(/ApiClientError/)).toHaveCount(0);
	});
});
