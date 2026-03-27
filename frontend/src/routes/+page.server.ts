import type { PageServerLoad } from './$types';
import { env } from '$env/dynamic/private';

/** Pre-fetch featured active markets and trending markets for the home page hero. */
export const load: PageServerLoad = async () => {
	const BACKEND = env.BACKEND_URL;
	if (!BACKEND) return { featured: null, trending: null };
	try {
		const [featuredRes, trendingRes] = await Promise.all([
			fetch(`${BACKEND}/api/markets?status=active&limit=20`),
			fetch(`${BACKEND}/api/markets/trending`)
		]);
		const featured = featuredRes.ok ? await featuredRes.json() : null;
		const trending = trendingRes.ok ? await trendingRes.json() : null;
		return { featured, trending };
	} catch {
		return { featured: null, trending: null };
	}
};
