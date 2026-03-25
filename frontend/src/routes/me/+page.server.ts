import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

/**
 * Redirect unauthenticated visitors away from the profile page.
 * We only redirect if there is no session cookie at all — if the cookie
 * exists but the backend is temporarily unavailable, we keep the page and
 * let the client-side auth store handle the "not logged in" UI.
 */
export const load: PageServerLoad = async ({ cookies }) => {
	if (!cookies.get('session')) {
		throw redirect(302, '/');
	}
	return {};
};
