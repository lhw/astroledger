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
		bug_fixes: '🐛 Bug Fixes',
		feature_delivery: '🚀 Feature Delivery',
		patch_timing: '⏰ Patch Timing',
		cig_drama: '🎭 CIG Drama',
		community_events: '🎉 Community Events',
		meta: '🤔 Meta'
	};
</script>

<div class="container mx-auto px-4 max-w-4xl py-8">
	<!-- Hero -->
	<section class="text-center mb-12">
		<h1 class="h1 text-primary-400 mb-4">
			⚖ ScolyMarket
		</h1>
		<p class="text-xl text-surface-200 mb-2">
			The galaxy's most prestigious prediction market for Star Citizen's
			<span class="text-primary-400 italic">perpetual development journey</span>.
		</p>
		<p class="text-surface-400 mb-8">
			Bet imaginary ScollyBucks™ on real predictions. No real money. No real ships. No ETA™.
		</p>

		{#if !$isLoggedIn}
			<button
				on:click={loginWithSCID}
				class="btn variant-filled-primary text-lg px-8 py-3"
			>
				Login with SCID to Start Betting
			</button>
		{:else}
			<a href="/markets/new" class="btn variant-filled-primary text-lg px-8 py-3">
				+ Submit a Market
			</a>
		{/if}
	</section>

	<!-- Stats bar -->
	<div class="grid grid-cols-3 gap-4 mb-10">
		<div class="card variant-glass-surface p-4 text-center rounded-lg">
			<div class="text-2xl font-bold text-primary-400">
				{featured ? featured.total : '…'}
			</div>
			<div class="text-surface-400 text-sm">Active Markets</div>
		</div>
		<div class="card variant-glass-surface p-4 text-center rounded-lg">
			<div class="text-2xl font-bold text-primary-400">1,000</div>
			<div class="text-surface-400 text-sm">Starting ScollyBucks™</div>
		</div>
		<div class="card variant-glass-surface p-4 text-center rounded-lg">
			<div class="text-2xl font-bold text-primary-400">∞</div>
			<div class="text-surface-400 text-sm">Years Until 1.0</div>
		</div>
	</div>

	<!-- Featured markets -->
	<section>
		<div class="flex items-center justify-between mb-4">
			<h2 class="h3 text-surface-100">Featured Markets</h2>
			<a href="/markets" class="text-primary-400 hover:text-primary-300 text-sm">
				View all →
			</a>
		</div>

		{#if loading}
			<div class="text-surface-400 text-center py-8">Loading markets…</div>
		{:else if !featured || featured.markets.length === 0}
			<div class="card variant-ghost-surface p-8 text-center rounded-lg">
				<p class="text-surface-400">No active markets yet. Be the first to submit one!</p>
			</div>
		{:else}
			<div class="space-y-3">
				{#each featured.markets.slice(0, 5) as market}
					<a
						href="/markets/{market.id}"
						class="card variant-glass-surface p-4 rounded-lg block hover:variant-glass-primary transition-all"
					>
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 min-w-0">
								<div class="text-xs text-surface-400 mb-1">
									{CATEGORY_LABELS[market.category] ?? market.category}
								</div>
								<div class="text-surface-100 font-medium truncate">
									{market.title}
								</div>
								<div class="text-surface-400 text-xs mt-1">
									by {market.creator_name} ·
									closes {new Date(market.resolution_deadline).toLocaleDateString()}
								</div>
							</div>
							<!-- Placeholder price — will be enhanced on the market list page -->
							<div class="flex-shrink-0 text-right">
								<div class="text-primary-400 font-bold">YES</div>
								<div class="text-surface-400 text-xs">50¢</div>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	</section>
</div>
