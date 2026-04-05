import { test, expect } from '@playwright/test';
import { mockMe, USER_LOGGED_IN } from './helpers/mock-api';
import type { Page } from '@playwright/test';
import type { BadgeCatalogEntry, AdminBadgeRelease, AdminBadgeDefinition } from '../src/lib/types';

// ─── Fixtures ────────────────────────────────────────────────────────────────

const USER_ADMIN = {
	...USER_LOGGED_IN,
	id: 3,
	display_name: 'AdminUser',
	is_admin: true,
	balance: 9999
};

const CATALOG: BadgeCatalogEntry[] = [
	{ key: 'aurora_pilot', title: 'Aurora Pilot', description: 'Started small.', tier: 1, purchasable: true },
	{ key: 'verse_veteran', title: "'Verse Veteran", description: 'A seasoned survivor.', tier: 3, purchasable: true },
	{ key: 'first_trade', title: 'First Trade', description: 'Made their first bet.', tier: 1, purchasable: false }
];

const RELEASE_LTI: AdminBadgeRelease = {
	id: 1,
	badge_key: 'aurora_pilot',
	title: 'Aurora Pilot',
	description: 'Started small.',
	tier: 1,
	price: 500,
	stock: null,
	sold: 3,
	released_at: '2026-01-01T00:00:00Z',
	expires_at: null,
	active: true,
	notes: 'Launch day drop',
	created_at: '2026-01-01T00:00:00Z',
	insurance: 'lti'
};

const RELEASE_6W: AdminBadgeRelease = {
	id: 2,
	badge_key: 'verse_veteran',
	title: "'Verse Veteran",
	description: 'A seasoned survivor.',
	tier: 3,
	price: 1200,
	stock: 50,
	sold: 8,
	released_at: '2026-02-01T00:00:00Z',
	expires_at: '2026-03-01T00:00:00Z',
	active: true,
	notes: null,
	created_at: '2026-02-01T00:00:00Z',
	insurance: '6w'
};

const RELEASE_NO_INS: AdminBadgeRelease = {
	id: 3,
	badge_key: 'aurora_pilot',
	title: 'Aurora Pilot',
	description: 'Started small.',
	tier: 1,
	price: 400,
	stock: 10,
	sold: 5,
	released_at: '2025-06-01T00:00:00Z',
	expires_at: '2025-07-01T00:00:00Z',
	active: false,
	notes: 'Archived release',
	created_at: '2025-06-01T00:00:00Z',
	insurance: ''
};

const BADGE_DEFS: AdminBadgeDefinition[] = [
	{
		id: 1,
		key: 'aurora_pilot',
		title: 'Aurora Pilot',
		description: 'Started small, dreamed big. The Aurora is a classic.',
		tier: 1,
		icon: '',
		is_hardcoded: true,
		purchasable: true,
		insurance: '6w',
		created_at: '2026-01-01T00:00:00Z'
	},
	{
		id: 2,
		key: 'explorer_badge',
		title: 'Explorer',
		description: 'Awarded for exploring the unknown.',
		tier: 3,
		icon: '🔭',
		is_hardcoded: false,
		purchasable: true,
		insurance: '',
		created_at: '2026-01-02T00:00:00Z'
	}
];

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function mockAdminApis(
	page: Page,
	opts: {
		catalog?: BadgeCatalogEntry[];
		releases?: AdminBadgeRelease[];
		defs?: AdminBadgeDefinition[];
	} = {}
) {
	const catalog = opts.catalog ?? CATALOG;
	const releases = opts.releases ?? [RELEASE_LTI, RELEASE_6W];
	const defs = opts.defs ?? BADGE_DEFS;

	await page.route('/api/admin/badge-catalog', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(catalog) })
	);
	await page.route('/api/admin/badge-releases', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(releases) })
	);
	await page.route('/api/admin/badge-definitions', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(defs) })
	);
}

async function goToBadgesTab(page: Page) {
	await page.goto('/admin');
	await page.getByRole('tab', { name: 'Badges' }).click();
	// Wait for the releases list or "No releases" text to be visible
	await page.waitForSelector('.badge-card, [class*="release"], text=No releases yet, text=All Releases', {
		timeout: 5000
	}).catch(() => {/* table may use different selectors */});
}

async function goToBadgeDefsTab(page: Page) {
	await page.goto('/admin');
	await page.getByRole('tab', { name: 'Badge Defs' }).click();
	await expect(page.getByText('New Badge Definition')).toBeVisible();
}

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Admin Panel — Badges', () => {
	test.describe('Access control', () => {
		test('shows access-restricted message for non-admin user', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await page.goto('/admin');
			await expect(page.getByText(/restricted/i)).toBeVisible();
		});

		test('shows access-restricted message for logged-out user', async ({ page }) => {
			await mockMe(page, null);
			await page.goto('/admin');
			await expect(page.getByText(/restricted/i)).toBeVisible();
		});

		test('shows Admin Panel heading for admin user', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await expect(page.getByRole('heading', { name: /admin panel/i })).toBeVisible();
		});

		test('shows Badges tab button for admin user', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await expect(page.getByRole('tab', { name: 'Badges' })).toBeVisible();
		});
	});

	test.describe('Badge Releases list', () => {
		test('shows All Releases section heading after clicking Badges tab', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('All Releases')).toBeVisible();
		});

		test('shows empty state when no releases exist', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText(/no releases yet/i)).toBeVisible();
		});

		test('shows release title in list', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.locator('.divide-y').getByText('Aurora Pilot', { exact: true })).toBeVisible();
		});

		test('shows badge_key in release row', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('aurora_pilot')).toBeVisible();
		});

		test('shows release price in list', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('500 bUEC')).toBeVisible();
		});

		test('shows insurance label LTI in release row', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			// Insurance chip in the release row (not in the form dropdown option)
			await expect(page.locator('.divide-y span').filter({ hasText: 'LTI' })).toBeVisible();
		});

		test('shows insurance label 6W Ins. in release row', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_6W] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			// Insurance chip in the release row renders as '6W Ins.'
			await expect(page.locator('.divide-y span').filter({ hasText: '6W Ins.' })).toBeVisible();
		});

		test('shows status chip for active release', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('active')).toBeVisible();
		});

		test('shows stock info for limited-stock release', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_6W] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			// Component renders '{sold}/{stock} sold' — fixture has sold:8, stock:50
			await expect(page.getByText(/8\/50 sold/)).toBeVisible();
		});

		test('shows multiple releases in list', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI, RELEASE_6W] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.locator('.divide-y').getByText('Aurora Pilot', { exact: true })).toBeVisible();
			await expect(page.locator('.divide-y').getByText("'Verse Veteran", { exact: true })).toBeVisible();
		});

		test('shows notes in release row when set', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('Launch day drop')).toBeVisible();
		});
	});

	test.describe('Badge Definitions', () => {
		test('does not show a Source column', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await goToBadgeDefsTab(page);
			await expect(page.getByRole('columnheader', { name: 'Source' })).toHaveCount(0);
		});

		test('shows a default tier symbol when the icon is empty', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await goToBadgeDefsTab(page);
			const row = page.locator('tr', { hasText: 'Aurora Pilot' });
			await expect(row.locator('.def-icon-chip')).toHaveText('▲');
		});
	});

	test.describe('Create Release form', () => {
		test('shows New Release section heading', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText('New Release')).toBeVisible();
		});

		test('shows insurance tier select dropdown', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.locator('#brc-insurance')).toBeVisible();
		});

		test('insurance dropdown has 6 Weeks, 120 Weeks, LTI options', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			const sel = page.locator('#brc-insurance');
			await expect(sel.locator('option[value=""]')).toHaveCount(0);
			await expect(sel.locator('option[value="6w"]')).toHaveText('6 Weeks');
			await expect(sel.locator('option[value="120w"]')).toHaveText('120 Weeks');
			await expect(sel.locator('option[value="lti"]')).toHaveText('LTI');
		});

		test('shows Create Release button', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByRole('button', { name: /create release/i })).toBeVisible();
		});

		test('shows validation error when submitting without badge selection', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page);
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /create release/i }).click();
			await expect(page.locator('p.text-red-400').filter({ hasText: /select a badge/i })).toBeVisible();
		});

		test('sends insurance field in create request', async ({ page }) => {
			await mockMe(page, USER_ADMIN);

			const createdRelease: AdminBadgeRelease = { ...RELEASE_LTI, id: 99 };
			await page.route('/api/admin/badge-releases', async (route) => {
				if (route.request().method() === 'POST') {
					await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(createdRelease) });
				} else {
					await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([RELEASE_LTI]) });
				}
			});
			await page.route('/api/admin/badge-catalog', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(CATALOG) })
			);

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();

			await page.locator('#brc-badge-select').selectOption('aurora_pilot');

			// Fill in price
			await page.locator('#brc-price').fill('500');

			// Select LTI insurance
			await page.locator('#brc-insurance').selectOption('lti');

			// Submit and capture the request body
			const [request] = await Promise.all([
				page.waitForRequest((req) => req.url().includes('/api/admin/badge-releases') && req.method() === 'POST'),
				page.getByRole('button', { name: /create release/i }).click()
			]);
			const capturedBody = JSON.parse(request.postData() ?? '{}') as Record<string, unknown>;

			expect(capturedBody.insurance).toBe('lti');
			expect(capturedBody.badge_key).toBe('aurora_pilot');
			expect(capturedBody.price).toBe(500);
		});

		test('creating with stock filled does not throw and sends numeric stock', async ({ page }) => {
			await mockMe(page, USER_ADMIN);

			const pageErrors: string[] = [];
			page.on('pageerror', (error) => pageErrors.push(error.message));

			const createdRelease: AdminBadgeRelease = { ...RELEASE_LTI, id: 101, stock: 25 };
			await page.route('/api/admin/badge-releases', async (route) => {
				if (route.request().method() === 'POST') {
					await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(createdRelease) });
				} else {
					await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
				}
			});
			await page.route('/api/admin/badge-catalog', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(CATALOG) })
			);

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.locator('#brc-badge-select').selectOption('aurora_pilot');
			await page.locator('#brc-price').fill('500');
			await page.locator('#brc-stock').fill('25');

			const [request] = await Promise.all([
				page.waitForRequest((req) => req.url().includes('/api/admin/badge-releases') && req.method() === 'POST'),
				page.getByRole('button', { name: /create release/i }).click()
			]);

			const capturedBody = JSON.parse(request.postData() ?? '{}') as Record<string, unknown>;
			expect(capturedBody.stock).toBe(25);
			expect(pageErrors).toEqual([]);
		});

		test('newly created release appears in list', async ({ page }) => {
			await mockMe(page, USER_ADMIN);

			const createdRelease: AdminBadgeRelease = { ...RELEASE_LTI, id: 10 };
			await page.route('/api/admin/badge-releases', async (route) => {
				if (route.request().method() === 'POST') {
					await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(createdRelease) });
				} else {
					await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
				}
			});
			await page.route('/api/admin/badge-catalog', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(CATALOG) })
			);

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByText(/no releases yet/i)).toBeVisible();

			await page.locator('#brc-badge-select').selectOption('aurora_pilot');

			await page.locator('#brc-price').fill('500');
			await page.getByRole('button', { name: /create release/i }).click();

			// The new release should appear in the releases list section
			await expect(page.locator('.divide-y').getByText('Aurora Pilot')).toBeVisible();

			// Empty state should be gone
			await expect(page.getByText(/no releases yet/i)).toHaveCount(0);
		});
	});

	test.describe('Edit Release', () => {
		test('shows Edit button on each release row', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await expect(page.getByRole('button', { name: /edit/i })).toBeVisible();
		});

		test('clicking Edit opens inline edit form', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();
			// Edit form has a Save button
			await expect(page.getByRole('button', { name: /save/i })).toBeVisible();
		});

		test('edit form preserves insurance in PUT request unchanged', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });

			const updated = { ...RELEASE_LTI };
			await page.route(`/api/admin/badge-releases/${RELEASE_LTI.id}`, async (route) => {
				if (route.request().method() === 'PUT') {
					await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(updated) });
				}
			});

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();

			// Edit form has no insurance UI — verify Save sends existing insurance via PUT
			const [request] = await Promise.all([
				page.waitForRequest((req) => req.url().includes(`/api/admin/badge-releases/${RELEASE_LTI.id}`) && req.method() === 'PUT'),
				page.getByRole('button', { name: /save/i }).click()
			]);
			const body = JSON.parse(request.postData() ?? '{}') as Record<string, unknown>;
			expect(body.insurance).toBe('lti');
		});

		test('save with changed price sends correct data in PUT', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });

			const updated = { ...RELEASE_LTI, price: 750 };
			await page.route(`/api/admin/badge-releases/${RELEASE_LTI.id}`, async (route) => {
				if (route.request().method() === 'PUT') {
					await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(updated) });
				}
			});

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();

			// Change price in the edit form (scoped to release row, not create form)
			await page.locator('.divide-y input[type="number"]').first().fill('750');

			const [request] = await Promise.all([
				page.waitForRequest((req) => req.url().includes(`/api/admin/badge-releases/${RELEASE_LTI.id}`) && req.method() === 'PUT'),
				page.getByRole('button', { name: /save/i }).click()
			]);
			const body = JSON.parse(request.postData() ?? '{}') as Record<string, unknown>;
			expect(body.price).toBe(750);
			expect(body.insurance).toBe('lti');
		});

		test('Cancel button closes edit form without saving', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();
			await expect(page.getByRole('button', { name: /save/i })).toBeVisible();
			await page.getByRole('button', { name: /cancel/i }).click();
			await expect(page.getByRole('button', { name: /save/i })).toHaveCount(0);
		});
	});

	test.describe('Archive Release', () => {
		test('shows Archive button when editing a release', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();
			await expect(page.getByRole('button', { name: /archive/i })).toBeVisible();
		});

		test('archive shows confirmation button', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();
			await page.getByRole('button', { name: /archive/i }).click();
			await expect(page.getByRole('button', { name: /confirm/i })).toBeVisible();
		});

		test('confirming archive calls DELETE/archive endpoint', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			await mockAdminApis(page, { releases: [RELEASE_LTI] });

			let archiveCalled = false;
			await page.route(`/api/admin/badge-releases/${RELEASE_LTI.id}/archive`, async (route) => {
				if (route.request().method() === 'POST') {
					archiveCalled = true;
					await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
				}
			});
			// Also intercept the PUT (some backends use PUT with active:false)
			await page.route(`/api/admin/badge-releases/${RELEASE_LTI.id}`, async (route) => {
				if (route.request().method() === 'DELETE') {
					archiveCalled = true;
					await route.fulfill({ status: 204 });
				} else if (route.request().method() === 'PUT') {
					archiveCalled = true;
					const updated = { ...RELEASE_LTI, active: false };
					await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(updated) });
				}
			});

			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			await page.getByRole('button', { name: /edit/i }).first().click();
			await page.getByRole('button', { name: /archive/i }).click();
			await page.getByRole('button', { name: /confirm/i }).click();

			expect(archiveCalled).toBe(true);
		});

		test('archived release shows "archived" status chip', async ({ page }) => {
			await mockMe(page, USER_ADMIN);
			// RELEASE_NO_INS is inactive
			await mockAdminApis(page, { releases: [RELEASE_NO_INS] });
			await page.goto('/admin');
			await page.getByRole('tab', { name: 'Badges' }).click();
			// Status chip uses exact text 'archived' (uppercase in CSS, lowercase in DOM)
			await expect(page.locator('.divide-y span').getByText('archived', { exact: true })).toBeVisible();
		});
	});
});
