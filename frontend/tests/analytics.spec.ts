/**
 * GoatCounter analytics integration tests.
 *
 * Tests are split into two groups:
 *
 *   1. Frontend-only (always run): intercept the browser request to verify
 *      the analytics POST fires on navigation. No backend or GC needed.
 *
 *   2. GC integration (skip-guarded): verify the GoatCounter API itself is
 *      reachable and accepts direct hit POSTs. Requires: task dev.
 *
 * All skip-guarded tests never break normal CI runs.
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

	test('navigating to the home page sends a hit via the analytics proxy', async ({ page }) => {
		// Intercept the analytics POST from the browser. This verifies the frontend
		// fires afterNavigate → POST /api/analytics/hit without needing the backend
		// or GoatCounter to be running.
		let capturedHit: { path?: string; title?: string } | null = null;
		await page.route('**/api/analytics/hit', async (route) => {
			const body = route.request().postDataJSON() as { path?: string; title?: string };
			capturedHit = body;
			await route.fulfill({ status: 204 });
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');

		// afterNavigate should have fired and called /api/analytics/hit
		expect(capturedHit).not.toBeNull();
		expect(capturedHit!.path).toBe('/');
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
