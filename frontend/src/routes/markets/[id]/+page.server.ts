import type { PageServerLoad } from './$types';
import { env } from '$env/dynamic/private';

/** Pre-fetch market detail and price history for SSR. */
export const load: PageServerLoad = async ({ params, cookies }) => {
	const BACKEND = env.BACKEND_URL;
	if (!BACKEND) return { market: null, history: [] }; // Not configured — client-side onMount handles it
	const session = cookies.get('session');
	const headers: Record<string, string> = {};
	if (session) headers['Cookie'] = `session=${session}`;

	try {
		const [marketRes, historyRes] = await Promise.all([
			fetch(`${BACKEND}/api/markets/${params.id}`, { headers }),
			fetch(`${BACKEND}/api/markets/${params.id}/history`, { headers })
		]);
		const market = marketRes.ok ? await marketRes.json() : null;
		const history = historyRes.ok ? await historyRes.json() : [];
		return { market, history };
	} catch {
		return { market: null, history: [] };
	}
};
