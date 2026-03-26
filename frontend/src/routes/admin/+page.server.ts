import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ cookies }) => {
	const sessionCookie = cookies.get('session');
	if (!sessionCookie) throw redirect(302, '/');

	const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

	let user: { is_admin?: unknown } | null = null;
	try {
		const res = await fetch(`${BACKEND}/api/me`, {
			headers: { Cookie: `session=${sessionCookie}` }
		});
		if (res.ok) user = await res.json();
	} catch {
		// Backend unreachable — treat as unauthenticated.
	}

	if (!user?.is_admin) throw redirect(302, '/');
	return { user };
};
