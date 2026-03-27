<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { initAuth, currentUser, isLoggedIn, isModerator, isAdmin } from '$lib/stores/auth';
	import { loginWithSCID, logout } from '$lib/api';
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import type { User } from '$lib/types';

	let { children, data } = $props<{ children: unknown; data: { user: User | null } }>();

	// Initialise auth store from server-provided data immediately (no flash).
	// $effect keeps the store in sync when the layout re-runs on navigation.
	$effect(() => {
		if (browser) currentUser.set(data.user);
	});

	onMount(async () => {
		// Client-side refresh: confirms session validity and picks up any
		// balance/role changes since the SSR snapshot. Also acts as fallback
		// when the backend was unreachable during SSR (e.g., in tests).
		await initAuth();
	});
</script>

<svelte:head>
	<title>ScolyMarket — Star Citizen Prediction Markets</title>
</svelte:head>

<div class="min-h-screen flex flex-col">
	<!-- Header — dark charcoal nav, RSI style -->
	<header class="flex items-center justify-between px-6 py-3 bg-surface-900 border-b border-surface-800 sticky top-0 z-10">
		<a href="/" class="flex items-center gap-3 no-underline">
			<span class="text-primary-400 font-bold text-lg tracking-widest uppercase">⚖ ScolyMarket</span>
			<span class="text-surface-500 text-xs hidden sm:inline tracking-wider uppercase">Prediction Markets</span>
		</a>

		<nav class="flex items-center gap-5">
			<a href="/markets" class="text-surface-300 hover:text-primary-400 text-sm tracking-wide transition-colors no-underline uppercase">
				Markets
			</a>
			<a href="/leaderboard" class="text-surface-300 hover:text-primary-400 text-sm tracking-wide transition-colors no-underline uppercase">
				Leaderboard
			</a>
			<a href="/fomo" class="text-surface-300 hover:text-yellow-400 text-sm tracking-wide transition-colors no-underline uppercase">
				FOMO
			</a>
			<a href="/admiral" class="text-surface-300 hover:text-blue-400 text-sm tracking-wide transition-colors no-underline uppercase">
				Rank
			</a>

			{#if $isModerator}
				<a href="/mod" class="text-yellow-400 hover:text-yellow-300 text-sm tracking-wide transition-colors no-underline uppercase">
					Mod Queue
				</a>
			{/if}

			{#if $isAdmin}
				<a href="/admin" class="text-red-400 hover:text-red-300 text-sm tracking-wide transition-colors no-underline uppercase">
					Admin
				</a>
			{/if}

			{#if $isLoggedIn && $currentUser}
				<span class="text-primary-400 text-sm font-semibold">
					{$currentUser.balance.toLocaleString()} bUEC
				</span>
				<a href="/me" class="flex items-center gap-2 text-surface-200 hover:text-primary-400 text-sm transition-colors no-underline">
					<UserAvatar src={$currentUser.avatar_url} name={$currentUser.display_name} size={24} />
					{$currentUser.display_name}
				</a>
				<button onclick={logout} class="border border-surface-600 text-surface-400 hover:border-primary-400 hover:text-primary-400 transition-colors rounded px-3 py-1 text-xs tracking-wider uppercase">
					Logout
				</button>
			{:else if $currentUser === null}
				<button onclick={loginWithSCID} class="btn btn-sm preset-filled-primary-500 tracking-wider uppercase text-xs">
					Login with SCID
				</button>
			{:else}
				<span class="text-surface-500 text-sm">Loading…</span>
			{/if}
		</nav>
	</header>

	<!-- Page content -->
	<main class="flex-1">
		{@render children()}
	</main>

	<!-- Footer -->
	<footer class="text-center text-surface-500 text-xs py-5 border-t border-surface-800 bg-surface-900">
		ScolyMarket — No real money. No real ships delivered.
	</footer>
</div>
