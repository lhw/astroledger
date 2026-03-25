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
	<!-- Header -->
	<header class="flex items-center justify-between px-4 py-2 bg-surface-900 border-b border-surface-700 sticky top-0 z-10">
		<a href="/" class="flex items-center gap-2 no-underline">
			<span class="text-primary-500 font-bold text-xl tracking-tight">⚖ ScolyMarket</span>
			<span class="text-surface-400 text-xs hidden sm:inline">Prediction Markets</span>
		</a>

		<nav class="flex items-center gap-4">
			<a href="/markets" class="text-surface-200 hover:text-primary-400 text-sm transition-colors no-underline">
				Markets
			</a>
			<a href="/leaderboard" class="text-surface-200 hover:text-primary-400 text-sm transition-colors no-underline">
				Leaderboard
			</a>

			{#if $isModerator}
				<a href="/mod" class="text-warning-400 hover:text-warning-300 text-sm transition-colors no-underline">
					Mod Queue
				</a>
			{/if}

			{#if $isLoggedIn && $currentUser}
				<span class="text-primary-500 text-sm font-semibold">
					{$currentUser.balance.toLocaleString()} bUEC
				</span>
				<a href="/me" class="btn btn-sm preset-tonal-surface no-underline">
					{$currentUser.display_name}
				</a>
				<button onclick={logout} class="btn btn-sm preset-outlined">Logout</button>
			{:else if $currentUser === null}
				<button onclick={loginWithSCID} class="btn btn-sm preset-filled-primary-500">
					Login with SCID
				</button>
			{:else}
				<span class="text-surface-400 text-sm">Loading…</span>
			{/if}
		</nav>
	</header>

	<!-- Page content -->
	<main class="flex-1">
		{@render children()}
	</main>

	<!-- Footer -->
	<footer class="text-center text-surface-500 text-xs py-4 border-t border-surface-700">
		ScolyMarket — No real money. No real ships delivered.
	</footer>
</div>
