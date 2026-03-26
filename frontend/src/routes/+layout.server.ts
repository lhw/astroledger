import type { LayoutServerLoad } from './$types';
import { env } from '$env/dynamic/private';

/**
 * Load the current user on every page render (server-side).
 * This eliminates the flash of unauthenticated state on initial load.
 * If the backend is unreachable (e.g., during tests), returns null gracefully
 * and lets the client-side initAuth() handle it via onMount fallback.
 */
export const load: LayoutServerLoad = async ({ cookies }) => {
	const sessionCookie = cookies.get('session');

	const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

	if (!sessionCookie) return { user: null };

	try {
		const res = await fetch(`${BACKEND}/api/me`, {
			headers: { Cookie: `session=${sessionCookie}` }
		});
		if (!res.ok) return { user: null };
		return { user: await res.json() };
	} catch {
		// Backend unavailable (e.g., during Playwright tests) — fall back gracefully.
		return { user: null };
	}
};
