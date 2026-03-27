<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { initAuth, currentUser, isLoggedIn, isModerator, isAdmin } from '$lib/stores/auth';
	import { loginWithSCID, logout } from '$lib/api';
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import type { User } from '$lib/types';

	let { children, data } = $props<{ children: unknown; data: { user: User | null } }>();
	let mobileMenuOpen = $state(false);

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

	$effect(() => {
		if (!browser) return;

		const closeOnResize = () => {
			if (window.innerWidth >= 768) mobileMenuOpen = false;
		};

		window.addEventListener('resize', closeOnResize);
		return () => window.removeEventListener('resize', closeOnResize);
	});
</script>

<svelte:head>
	<title>ScolyMarket — Star Citizen Prediction Markets</title>
</svelte:head>

<div class="min-h-screen flex flex-col">
	<!-- Header — dark charcoal nav, RSI style -->
	<header class="relative flex items-center justify-between px-4 sm:px-6 py-3 bg-surface-900 border-b border-surface-800 sticky top-0 z-20">
		<a href="/" class="flex items-center gap-3 no-underline min-w-0" onclick={() => (mobileMenuOpen = false)}>
			<span class="text-primary-400 font-bold text-lg tracking-widest uppercase truncate">⚖ ScolyMarket</span>
			<span class="text-surface-500 text-xs hidden sm:inline tracking-wider uppercase">Prediction Markets</span>
		</a>

		<button
			type="button"
			onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
			class="md:hidden inline-flex items-center justify-center rounded border border-surface-700 px-3 py-2 text-surface-200 hover:text-primary-400 hover:border-primary-400 transition-colors"
			aria-expanded={mobileMenuOpen}
			aria-label="Toggle navigation menu"
		>
			{#if mobileMenuOpen}
				<span class="text-base leading-none">✕</span>
			{:else}
				<span class="text-base leading-none">☰</span>
			{/if}
		</button>

		<nav class="hidden md:flex items-center gap-5">
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

		{#if mobileMenuOpen}
			<div class="absolute left-0 right-0 top-full mx-3 mt-2 rounded-xl border border-surface-700 bg-surface-900/95 backdrop-blur p-3 shadow-xl md:hidden">
				<nav class="flex flex-col gap-1">
					<a href="/markets" class="text-surface-300 hover:text-primary-400 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
						Markets
					</a>
					<a href="/leaderboard" class="text-surface-300 hover:text-primary-400 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
						Leaderboard
					</a>
					<a href="/fomo" class="text-surface-300 hover:text-yellow-400 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
						FOMO
					</a>
					<a href="/admiral" class="text-surface-300 hover:text-blue-400 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
						Rank
					</a>

					{#if $isModerator}
						<a href="/mod" class="text-yellow-400 hover:text-yellow-300 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
							Mod Queue
						</a>
					{/if}

					{#if $isAdmin}
						<a href="/admin" class="text-red-400 hover:text-red-300 text-sm tracking-wide transition-colors no-underline uppercase px-2 py-2 rounded hover:bg-surface-800" onclick={() => (mobileMenuOpen = false)}>
							Admin
						</a>
					{/if}

					<div class="my-2 border-t border-surface-800"></div>

					{#if $isLoggedIn && $currentUser}
						<div class="flex items-center justify-between gap-2 px-2 py-2">
							<span class="text-primary-400 text-sm font-semibold">{$currentUser.balance.toLocaleString()} bUEC</span>
							<a href="/me" class="flex items-center gap-2 text-surface-200 hover:text-primary-400 text-sm transition-colors no-underline" onclick={() => (mobileMenuOpen = false)}>
								<UserAvatar src={$currentUser.avatar_url} name={$currentUser.display_name} size={24} />
								<span class="max-w-[10rem] truncate">{$currentUser.display_name}</span>
							</a>
						</div>
						<button onclick={logout} class="w-full border border-surface-600 text-surface-400 hover:border-primary-400 hover:text-primary-400 transition-colors rounded px-3 py-2 text-xs tracking-wider uppercase">
							Logout
						</button>
					{:else if $currentUser === null}
						<button onclick={loginWithSCID} class="w-full btn btn-sm preset-filled-primary-500 tracking-wider uppercase text-xs">
							Login with SCID
						</button>
					{:else}
						<span class="text-surface-500 text-sm px-2 py-2">Loading…</span>
					{/if}
				</nav>
			</div>
		{/if}
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
