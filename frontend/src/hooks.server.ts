import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

// Internal SvelteKit / Vite paths we never want to count as page views.
const SKIP_PREFIXES = ['/_app/', '/__data', '/favicon', '/robots.txt', '/sitemap'];

/** Returns true only for a real browser navigation (initial HTML load). */
function isPageRequest(request: Request): boolean {
	const { pathname } = new URL(request.url);
	if (SKIP_PREFIXES.some((p) => pathname.startsWith(p))) return false;
	const accept = request.headers.get('accept') ?? '';
	return accept.includes('text/html');
}

/**
 * Record a page view with GoatCounter via its server-side count API.
 * Fire-and-forget — never blocks the response, never throws.
 * Passes real IP, User-Agent, and Referer so GoatCounter can deduplicate
 * and detect bots just like it would with client-side JS.
 */
function recordHit(request: Request): void {
	const gcURL = env.GOATCOUNTER_URL;
	const gcKey = env.GOATCOUNTER_API_KEY;
	if (!gcURL || !gcKey) return;
	if (!isPageRequest(request)) return;

	const url = new URL(request.url);
	const path = url.pathname + url.search;
	const ref = request.headers.get('referer') ?? '';
	const ua = request.headers.get('user-agent') ?? '';
	// Respect X-Forwarded-For set by Caddy / reverse proxy for real client IP.
	const ip =
		request.headers.get('x-forwarded-for')?.split(',')[0].trim() ??
		request.headers.get('x-real-ip') ??
		'';
	const language = request.headers.get('accept-language') ?? '';

	fetch(`${gcURL}/api/v0/count`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			Authorization: `Bearer ${gcKey}`,
		},
		body: JSON.stringify({
			no_sessions: false,
			hits: [{ path, ref, user_agent: ua, ip, language }],
		}),
	})
		.then((res) => {
			if (!res.ok) {
				res.text().then((body) => {
					console.error(`[analytics] GoatCounter returned ${res.status}: ${body}`);
				});
			}
		})
		.catch((err) => {
			console.error(`[analytics] GoatCounter unreachable (${gcURL}):`, err.message ?? err);
		});
}

export const handle: Handle = async ({ event, resolve }) => {
	recordHit(event.request);
	return resolve(event);
};
