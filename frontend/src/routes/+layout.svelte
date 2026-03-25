<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { initAuth, currentUser, isLoggedIn, isModerator } from '$lib/stores/auth';
	import { loginWithSCID, logout } from '$lib/api';

	let { children } = $props();

	onMount(async () => {
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

			{#if $isModerator}
				<a href="/mod" class="text-yellow-400 hover:text-yellow-300 text-sm tracking-wide transition-colors no-underline uppercase">
					Mod Queue
				</a>
			{/if}

			{#if $isLoggedIn && $currentUser}
				<span class="text-primary-400 text-sm font-semibold">
					{$currentUser.balance.toLocaleString()} bUEC
				</span>
				<a href="/me" class="text-surface-200 hover:text-primary-400 text-sm transition-colors no-underline">
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
	<footer class="text-center text-surface-500 text-xs py-5 border-t border-surface-200 bg-white">
		ScolyMarket — No real money. No real ships delivered.
	</footer>
</div>
