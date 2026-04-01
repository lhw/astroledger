import { test, expect } from '@playwright/test';
import { mockMe, USER_LOGGED_IN } from './helpers/mock-api';
import type { Page } from '@playwright/test';
import type { StoreBadge } from '../src/lib/types';

// ─── Fixtures ────────────────────────────────────────────────────────────────

const BADGE_NO_INSURANCE: StoreBadge = {
	badge_key: 'citizen_backer',
	title: 'Citizen Backer',
	description: 'A true believer. Been here since before it was cool.',
	tier: 1,
	cost: 500,
	owned: false,
	insurance: '',
	expired: false
};

const BADGE_LTI: StoreBadge = {
	badge_key: 'aurora_pilot',
	title: 'Aurora Pilot',
	description: 'Started small, dreamed big.',
	tier: 2,
	cost: 750,
	owned: false,
	insurance: 'lti',
	expired: false
};

const BADGE_6W: StoreBadge = {
	badge_key: 'space_whale',
	title: 'Space Whale',
	description: 'Your wallet is canon-sized.',
	tier: 3,
	cost: 1000,
	owned: false,
	insurance: '6w',
	expired: false
};

const BADGE_120W: StoreBadge = {
	badge_key: 'verse_veteran',
	title: "'Verse Veteran",
	description: 'A seasoned survivor of the persistent universe.',
	tier: 3,
	cost: 1200,
	owned: false,
	insurance: '120w',
	expired: false
};

const BADGE_LIMITED_STOCK: StoreBadge = {
	badge_key: 'hangar_queen',
	title: 'Hangar Queen',
	description: 'The ships sit in the hangar.',
	tier: 2,
	cost: 800,
	owned: false,
	insurance: '',
	expired: false,
	stock: 10,
	remaining_stock: 3
};

const BADGE_SOLD_OUT: StoreBadge = {
	badge_key: 'star_gazer',
	title: 'Star Gazer',
	description: 'Watched the CitizenCon stream live every year.',
	tier: 2,
	cost: 600,
	owned: false,
	insurance: '',
	expired: false,
	stock: 5,
	remaining_stock: 0
};

const BADGE_TIME_LIMITED: StoreBadge = {
	badge_key: 'roadmap_reader',
	title: 'Roadmap Reader',
	description: 'Holds 47 open tabs of schedule promises.',
	tier: 1,
	cost: 400,
	owned: false,
	insurance: '',
	expired: false,
	available_until: new Date(Date.now() + 5 * 86_400_000).toISOString() // 5 days from now
};

const BADGE_EXPIRED: StoreBadge = {
	badge_key: 'quick_shot',
	title: 'Quick Shot',
	description: 'Too slow.',
	tier: 1,
	cost: 300,
	owned: false,
	insurance: '',
	expired: true,
	available_until: '2020-01-01T00:00:00Z'
};

const BADGE_OWNED: StoreBadge = {
	badge_key: 'professional_bug_finder',
	title: 'Professional Bug Finder',
	description: "It's not a bug, it's a stretch goal.",
	tier: 1,
	cost: 550,
	owned: true,
	insurance: '6w',
	expired: false
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function mockFomoStore(page: Page, badges: StoreBadge[]) {
	await page.route('/api/fomo', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(badges) })
	);
}

async function mockPurchase(page: Page, response: { status: number; body?: string } = { status: 200, body: '{}' }) {
	await page.route('/api/fomo/purchase', (route) =>
		route.fulfill({ status: response.status, contentType: 'application/json', body: response.body ?? '{}' })
	);
}

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('FOMO Store', () => {
	test.describe('Page structure', () => {
		test('renders the page heading', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, []);
			await page.goto('/fomo');
			await expect(page.getByRole('heading', { name: /FOMO Store/i })).toBeVisible();
		});

		test('shows descriptive subtitle', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, []);
			await page.goto('/fomo');
			await expect(page.getByText(/clout/i)).toBeVisible();
		});

		test('shows empty state when no badges available', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, []);
			await page.goto('/fomo');
			// No badge cards rendered
			await expect(page.locator('.badge-card')).toHaveCount(0);
		});

		test('shows badge cards for each available badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE, BADGE_LTI, BADGE_6W]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-card')).toHaveCount(3);
		});
	});

	test.describe('Badge card display', () => {
		test('shows badge title and description', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.getByText('Citizen Backer')).toBeVisible();
			await expect(page.getByText(/true believer/i)).toBeVisible();
		});

		test('shows tier label', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.getByText('Common')).toBeVisible();
		});

		test('shows price in bUEC', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.getByText('500 bUEC')).toBeVisible();
		});

		test('shows Login to Buy button when logged out', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.getByRole('link', { name: /login to buy/i })).toBeVisible();
		});

		test('shows Buy button when logged in', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 1000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await mockPurchase(page);
			await page.goto('/fomo');
			await expect(page.getByRole('button', { name: /^Buy$/i })).toBeVisible();
		});

		test('shows balance chip when logged in', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2500 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			// Balance chip is in the page header (nav also shows balance; narrow to main)
			await expect(page.locator('main, [data-testid="store-balance"], .text-primary-700').getByText(/2,500/).first()).toBeVisible();
		});
	});

	test.describe('Insurance chips', () => {
		test('shows no insurance chip when insurance is empty', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip')).toHaveCount(0);
		});

		test('shows LTI chip for lti insurance', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_LTI]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip.ins-lti')).toBeVisible();
			await expect(page.locator('.badge-ins-chip.ins-lti')).toHaveText('LTI');
		});

		test('shows 6W Ins. chip for 6w insurance', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_6W]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip.ins-6w')).toBeVisible();
			await expect(page.locator('.badge-ins-chip.ins-6w')).toHaveText('6W Ins.');
		});

		test('shows 120W Ins. chip for 120w insurance', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_120W]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip.ins-120w')).toBeVisible();
			await expect(page.locator('.badge-ins-chip.ins-120w')).toHaveText('120W Ins.');
		});

		test('shows all insurance chip variants for mixed store', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_LTI, BADGE_6W, BADGE_120W]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip')).toHaveCount(3);
		});
	});

	test.describe('Scarcity states', () => {
		test('shows limited stock info when stock is set', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_LIMITED_STOCK]);
			await page.goto('/fomo');
			await expect(page.getByText(/3 \/ 10 left/)).toBeVisible();
		});

		test('shows low-stock indicator when remaining <= 25%', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_LIMITED_STOCK]); // 3/10 = 30%... but let's also test a low one
			const lowStock: StoreBadge = { ...BADGE_LIMITED_STOCK, remaining_stock: 2 }; // 2/10 = 20%
			await mockFomoStore(page, [lowStock]);
			await page.goto('/fomo');
			// The 🔥 emoji appears when pct <= 0.1, 📦 otherwise — 2/10 shows 📦 (20%)
			await expect(page.getByText(/2 \/ 10 left/)).toBeVisible();
		});

		test('shows SOLD OUT banner for sold-out badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_SOLD_OUT]);
			await page.goto('/fomo');
			// Banner is inside .scarcity-banner.soldout
			await expect(page.locator('.scarcity-banner.soldout')).toBeVisible();
			await expect(page.locator('.scarcity-banner.soldout')).toHaveText('SOLD OUT');
		});

		test('shows Sold Out tag in footer for sold-out badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_SOLD_OUT]);
			await page.goto('/fomo');
			// Footer tag is .badge-unavailable-tag with text "Sold Out"
			await expect(page.locator('.badge-unavailable-tag')).toBeVisible();
			await expect(page.locator('.badge-unavailable-tag')).toHaveText('Sold Out');
		});

		test('shows EXPIRED banner for expired badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_EXPIRED]);
			await page.goto('/fomo');
			// Banner is inside .scarcity-banner.expired
			await expect(page.locator('.scarcity-banner.expired')).toBeVisible();
			await expect(page.locator('.scarcity-banner.expired')).toHaveText('EXPIRED');
		});

		test('shows Expired tag in footer for expired badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_EXPIRED]);
			await page.goto('/fomo');
			// Footer tag is .badge-unavailable-tag with text "Expired"
			await expect(page.locator('.badge-unavailable-tag')).toBeVisible();
			await expect(page.locator('.badge-unavailable-tag')).toHaveText('Expired');
		});

		test('shows time-until-expiry chip for time-limited badge', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_TIME_LIMITED]);
			await page.goto('/fomo');
			// Should say "Expires in N days" or similar
			await expect(page.getByText(/expire/i)).toBeVisible();
		});

		test('shows Buy button disabled for sold-out badge', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 5000 });
			await mockFomoStore(page, [BADGE_SOLD_OUT]);
			await page.goto('/fomo');
			// No active buy button — just "Sold Out" tag
			await expect(page.getByRole('button', { name: /^Buy$/i })).toHaveCount(0);
		});
	});

	test.describe('Owned badges', () => {
		test('shows Owned tag for already-owned badge', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockFomoStore(page, [BADGE_OWNED]);
			await page.goto('/fomo');
			await expect(page.getByText('Owned')).toBeVisible();
		});

		test('does not show Buy button for owned badge', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockFomoStore(page, [BADGE_OWNED]);
			await page.goto('/fomo');
			await expect(page.getByRole('button', { name: /^Buy$/i })).toHaveCount(0);
		});

		test('shows owned chip style on owned card', async ({ page }) => {
			await mockMe(page, USER_LOGGED_IN);
			await mockFomoStore(page, [BADGE_OWNED]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-card.owned')).toHaveCount(1);
		});
	});

	test.describe('Purchase flow', () => {
		test('Buy button sends purchase request with badge_key and insurance', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_LTI]);

			const requestPromise = page.waitForRequest('/api/fomo/purchase');
			await page.route('/api/fomo/purchase', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
			);

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();

			const req = await requestPromise;
			const capturedBody = JSON.parse(req.postData() ?? '{}') as Record<string, unknown>;
			expect(capturedBody.badge_key).toBe('aurora_pilot');
			expect(capturedBody.insurance).toBe('lti');
		});

		test('Buy button sends empty insurance when badge has none', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);

			const requestPromise = page.waitForRequest('/api/fomo/purchase');
			await page.route('/api/fomo/purchase', (route) =>
				route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
			);

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();

			const req = await requestPromise;
			const capturedBody = JSON.parse(req.postData() ?? '{}') as Record<string, unknown>;
			expect(capturedBody.badge_key).toBe('citizen_backer');
			expect(capturedBody.insurance).toBe('');
		});

		test('shows success message after purchase', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await mockPurchase(page, { status: 200, body: '{}' });

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();
			await expect(page.getByText(/added to your profile/i)).toBeVisible();
		});

		test('marks badge as Owned after successful purchase', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await mockPurchase(page, { status: 200, body: '{}' });

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();
			await expect(page.getByText('Owned')).toBeVisible();
		});

		test('deducts cost from displayed balance after purchase', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 1000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]); // costs 500
			await mockPurchase(page, { status: 200, body: '{}' });

			await page.goto('/fomo');
			// Balance chip is in the fomo page header (text-primary-700 span), separate from nav
			const balanceChip = page.locator('.text-primary-700').first();
			await expect(balanceChip).toHaveText('1,000 bUEC');
			await page.getByRole('button', { name: /^Buy$/i }).click();
			await expect(balanceChip).toHaveText('500 bUEC');
		});

		test('shows error message on purchase failure', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await mockPurchase(page, { status: 400, body: JSON.stringify({ error: 'insufficient balance' }) });

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();
			await expect(page.getByText(/insufficient balance/i)).toBeVisible();
		});

		test('does not show Buy button when balance is insufficient', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 100 }); // BADGE costs 500
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			// canBuy() returns false because balance < cost; button exists but is disabled
			const btn = page.getByRole('button', { name: /^Buy$/i });
			await expect(btn).toBeDisabled();
		});

		test('during purchase, button shows Buying…', async ({ page }) => {
			await mockMe(page, { ...USER_LOGGED_IN, balance: 2000 });
			await mockFomoStore(page, [BADGE_NO_INSURANCE]);

			// Use a deferred promise so we can observe the in-flight loading state
			let resolveRoute!: () => void;
			const routeReady = new Promise<void>((res) => { resolveRoute = res; });
			await page.route('/api/fomo/purchase', async (route) => {
				// Wait briefly to let the UI update before resolving
				await routeReady;
				await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
			});

			await page.goto('/fomo');
			await page.getByRole('button', { name: /^Buy$/i }).click();
			// After click, before route resolves, button should say Buying…
			await expect(page.getByRole('button', { name: /Buying…/i })).toBeVisible();
			// Now let the route finish
			resolveRoute();
		});
	});

	test.describe('Multiple badges in store', () => {
		test('renders each badge independently with correct insurance', async ({ page }) => {
			await mockMe(page, null);
			await mockFomoStore(page, [BADGE_LTI, BADGE_6W, BADGE_120W, BADGE_NO_INSURANCE]);
			await page.goto('/fomo');
			await expect(page.locator('.badge-ins-chip.ins-lti')).toBeVisible();
			await expect(page.locator('.badge-ins-chip.ins-6w')).toBeVisible();
			await expect(page.locator('.badge-ins-chip.ins-120w')).toBeVisible();
			await expect(page.locator('.badge-ins-chip')).toHaveCount(3); // NO_INSURANCE has no chip
		});

		test('shows correct card count', async ({ page }) => {
			const allBadges = [BADGE_LTI, BADGE_6W, BADGE_120W, BADGE_NO_INSURANCE, BADGE_OWNED, BADGE_SOLD_OUT];
			await mockMe(page, USER_LOGGED_IN);
			await mockFomoStore(page, allBadges);
			await page.goto('/fomo');
			await expect(page.locator('.badge-card')).toHaveCount(allBadges.length);
		});
	});
});
