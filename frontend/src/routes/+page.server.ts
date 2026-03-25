import type { PageServerLoad } from './$types';
import { env } from '$env/dynamic/private';

/** Pre-fetch featured active markets for the home page hero. */
export const load: PageServerLoad = async () => {
	const BACKEND = env.BACKEND_URL;
	if (!BACKEND) return { featured: null }; // Not configured — client-side onMount handles it
	try {
		const res = await fetch(`${BACKEND}/api/markets?status=active&limit=20`);
		if (!res.ok) return { featured: null };
		return { featured: await res.json() };
	} catch {
		return { featured: null };
	}
};
