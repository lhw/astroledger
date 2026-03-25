<script lang="ts">
	import { onMount } from 'svelte';
	import { listMarkets } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import { loginWithSCID } from '$lib/api';
	import type { MarketList } from '$lib/types';

	let featured: MarketList | null = null;
	let loading = true;

	onMount(async () => {
		try {
			featured = await listMarkets('active', '', 0);
		} catch {
			featured = null;
		} finally {
			loading = false;
		}
	});

	const CATEGORY_LABELS: Record<string, string> = {
		bug_fixes: 'Bug Fix',
		feature_delivery: 'Feature / Patch',
		patch_timing: 'Patch Timing',
		community_events: 'Community Event',
		meta: 'Meta'
	};
</script>

<svelte:head>
	<title>ScolyMarket — Star Citizen Prediction Markets</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-8">
	<!-- Hero -->
	<section class="text-center mb-12">
		<h1 class="text-5xl font-bold text-primary-500 mb-4">⚖ ScolyMarket</h1>
		<p class="text-xl text-surface-200 mb-2">
			Prediction markets for Star Citizen events and development milestones.
		</p>
		<p class="text-surface-400 mb-8">
			Bet bUEC on real predictions. No real money involved.
		</p>

		{#if !$isLoggedIn}
			<button onclick={loginWithSCID} class="btn preset-filled-primary-500 text-lg px-8 py-3">
				Login with SCID to Start Betting
			</button>
		{:else}
			<a href="/markets/new" class="btn preset-filled-primary-500 text-lg px-8 py-3 no-underline">
				+ Submit a Market
			</a>
		{/if}
	</section>

	<!-- Stats bar -->
	<div class="grid grid-cols-2 gap-4 mb-10">
		<div class="card preset-tonal-surface p-4 text-center rounded-lg">
			<div class="text-2xl font-bold text-primary-500">{featured ? featured.total : '…'}</div>
			<div class="text-surface-400 text-sm">Active Markets</div>
		</div>
		<div class="card preset-tonal-surface p-4 text-center rounded-lg">
			<div class="text-2xl font-bold text-primary-500">1,000</div>
			<div class="text-surface-400 text-sm">Starting bUEC</div>
		</div>
	</div>

	<!-- Featured markets -->
	<section>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-2xl font-semibold text-surface-100">Featured Markets</h2>
			<a href="/markets" class="text-primary-400 hover:text-primary-300 text-sm no-underline">
				View all →
			</a>
		</div>

		{#if loading}
			<div class="text-surface-400 text-center py-8">Loading markets…</div>
		{:else if !featured || featured.markets.length === 0}
			<div class="card preset-tonal-surface p-8 text-center rounded-lg">
				<p class="text-surface-400">No active markets yet. Be the first to submit one!</p>
			</div>
		{:else}
			<div class="space-y-3">
				{#each featured.markets.slice(0, 5) as market}
					<a
						href="/markets/{market.id}"
						class="card preset-tonal-surface p-4 rounded-lg block hover:preset-tonal-primary transition-all no-underline"
					>
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 min-w-0">
								<div class="text-xs text-surface-400 mb-1">
									{CATEGORY_LABELS[market.category] ?? market.category}
								</div>
								<div class="text-surface-100 font-medium truncate">{market.title}</div>
								<div class="text-surface-400 text-xs mt-1">
									by {market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
								</div>
							</div>
							<div class="flex-shrink-0 text-right">
								<div class="text-primary-500 font-bold">YES</div>
								<div class="text-surface-400 text-xs">50¢</div>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	</section>
</div>
