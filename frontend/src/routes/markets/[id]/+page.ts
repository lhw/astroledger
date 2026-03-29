import type { PageLoad } from './$types';

// Data is fetched client-side in onMount. Return null so the page
// initialises its loading state and fetches on mount.
export const load: PageLoad = async () => {
	return { market: null, history: [] as import('$lib/types').PricePoint[] };
};
