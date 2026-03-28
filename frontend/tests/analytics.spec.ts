/**
 * GoatCounter analytics integration tests.
 *
 * These tests verify that page hits from SvelteKit's hooks.server.ts actually
 * reach GoatCounter. They require the full dev stack (task dev) to be running:
 *   • Frontend at :5173
 *   • Backend at :8080
 *   • GoatCounter at :8081
 *
 * Run with the full stack: GOATCOUNTER_API_KEY=<key> npx playwright test analytics
 *
 * All tests skip gracefully when GoatCounter / backend is unreachable, so they
 * never break normal CI runs.
 */

import { test, expect } from '@playwright/test';

const GC_URL = process.env.GOATCOUNTER_URL ?? 'http://localhost:8081';
const GC_KEY = process.env.GOATCOUNTER_API_KEY ?? '';

// ── Helpers ────────────────────────────────────────────────────────────────

/** Check reachability of a URL; returns false on network error. */
async function isReachable(url: string, apiKey?: string): Promise<boolean> {
	try {
		const headers: Record<string, string> = {};
		if (apiKey) headers['Authorization'] = `Bearer ${apiKey}`;
		const res = await fetch(url, { headers, signal: AbortSignal.timeout(2000) });
		// 401 means the server is up but our key is wrong — still reachable
		return res.status < 500;
	} catch {
		return false;
	}
}

/**
 * Fetch the current total page-view count from GoatCounter's stats API.
 * Returns null when unreachable or key is missing.
 */
async function getGCHitCount(): Promise<number | null> {
	if (!GC_KEY) return null;
	try {
		const today = new Date().toISOString().split('T')[0];
		const yesterday = new Date(Date.now() - 86400 * 1000).toISOString().split('T')[0];
		const url = `${GC_URL}/api/v0/stats/hits?start=${yesterday}&end=${today}&limit=200`;
		const res = await fetch(url, {
			headers: { Authorization: `Bearer ${GC_KEY}` },
			signal: AbortSignal.timeout(3000),
		});
		if (!res.ok) return null;
		const data = (await res.json()) as { hits?: { count: number }[] };
		return (data.hits ?? []).reduce((sum, h) => sum + (h.count ?? 0), 0);
	} catch {
		return null;
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

test.describe('GoatCounter integration', () => {
	test.beforeAll(async () => {
		if (!GC_KEY) {
			console.log('[analytics] GOATCOUNTER_API_KEY not set — skipping backend tests');
		}
	});

	test('GoatCounter API is reachable and accepts the API key', async ({ request }) => {
		if (!GC_KEY) {
			test.skip(true, 'GOATCOUNTER_API_KEY not set');
			return;
		}
		const up = await isReachable(`${GC_URL}/api/v0/me`, GC_KEY);
		if (!up) {
			test.skip(true, `GoatCounter not running at ${GC_URL}`);
			return;
		}

		// /api/v0/me returns 200 with a valid Bearer token
		const res = await request.get(`${GC_URL}/api/v0/me`, {
			headers: { Authorization: `Bearer ${GC_KEY}` },
		});
		expect(res.status()).toBe(200);
	});

	test('direct POST to /api/v0/count is accepted (200 or 202)', async ({ request }) => {
		if (!GC_KEY) {
			test.skip(true, 'GOATCOUNTER_API_KEY not set');
			return;
		}
		const up = await isReachable(`${GC_URL}/api/v0/me`, GC_KEY);
		if (!up) {
			test.skip(true, `GoatCounter not running at ${GC_URL}`);
			return;
		}

		const res = await request.post(`${GC_URL}/api/v0/count`, {
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${GC_KEY}`,
			},
			data: {
				no_sessions: false,
				hits: [
					{
						path: '/playwright-direct-hit-test',
						user_agent:
							'Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0',
						ip: '203.0.113.1',
					},
				],
			},
		});
		// GoatCounter returns 200 on success
		expect([200, 202]).toContain(res.status());
	});

	test('navigating to the home page sends a hit via hooks.server.ts', async ({
		page,
		request,
	}) => {
		if (!GC_KEY) {
			test.skip(true, 'GOATCOUNTER_API_KEY not set');
			return;
		}
		const up = await isReachable(`${GC_URL}/api/v0/me`, GC_KEY);
		if (!up) {
			test.skip(true, `GoatCounter not running at ${GC_URL}`);
			return;
		}

		// Record current hit count before navigation
		const before = await getGCHitCount();
		if (before === null) {
			test.skip(true, 'Could not read GoatCounter stats');
			return;
		}

		// Navigate to the home page — this triggers hooks.server.ts → GoatCounter POST
		await page.goto('/');
		await page.waitForLoadState('networkidle');

		// GoatCounter processes hits asynchronously; give it up to 5 s
		let after: number | null = null;
		for (let i = 0; i < 10; i++) {
			await page.waitForTimeout(500);
			after = await getGCHitCount();
			if (after !== null && after > before) break;
		}

		expect(after).not.toBeNull();
		expect(after!).toBeGreaterThan(before);
	});

	test('no analytics script tags or external analytics requests in browser', async ({ page }) => {
		// This test always runs — no GoatCounter dependency.
		const externalRequests: string[] = [];
		page.on('request', (req) => {
			const url = req.url();
			if (/goatcounter|gc\.zgo\.at|plausible|matomo|google-analytics|googletagmanager/i.test(url)) {
				externalRequests.push(url);
			}
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');

		// The browser must NEVER contact an analytics domain directly.
		// All tracking is server-side (hooks.server.ts → GoatCounter API).
		expect(externalRequests).toHaveLength(0);
	});
});
