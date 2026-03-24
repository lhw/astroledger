import { writable, derived } from 'svelte/store';
import type { User } from '$lib/types';
import { getMe } from '$lib/api';

/** The currently authenticated user, or null if not logged in, or undefined if loading. */
export const currentUser = writable<User | null | undefined>(undefined);

/** True if auth state has been resolved (even if logged out). */
export const authReady = derived(currentUser, ($user) => $user !== undefined);

/** True if the user is logged in. */
export const isLoggedIn = derived(currentUser, ($user) => $user !== null && $user !== undefined);

/** True if the current user is a moderator or admin. */
export const isModerator = derived(
	currentUser,
	($user) => Boolean($user?.is_moderator || $user?.is_admin)
);

/**
 * Fetch the current user from the API and populate the auth store.
 * Call this once on app initialisation.
 */
export async function initAuth(): Promise<void> {
	try {
		const user = await getMe();
		currentUser.set(user);
	} catch {
		currentUser.set(null);
	}
}
