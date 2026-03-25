import type { PageServerLoad } from './$types';
import { env } from '$env/dynamic/private';

/** Pre-fetch the initial active markets list for SSR. */
export const load: PageServerLoad = async ({ cookies }) => {
	const BACKEND = env.BACKEND_URL;
	if (!BACKEND) return { markets: null }; // Not configured — client-side onMount handles it
	const session = cookies.get('session');
	const headers: Record<string, string> = {};
	if (session) headers['Cookie'] = `session=${session}`;

	try {
		const res = await fetch(`${BACKEND}/api/markets?status=active&limit=20`, { headers });
		if (!res.ok) return { markets: null };
		return { markets: await res.json() };
	} catch {
		return { markets: null };
	}
};
