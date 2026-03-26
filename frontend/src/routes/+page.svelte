<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { listMarkets } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import { loginWithSCID } from '$lib/api';
	import type { MarketList } from '$lib/types';

	let { data } = $props<{ data: { featured: MarketList | null } }>();

	// untrack() signals that we intentionally want a one-time snapshot, not a reactive dependency.
	const { featured: initialFeatured } = untrack(() => data);
	let featured = $state<MarketList | null>(initialFeatured ?? null);
	let loading = $state(initialFeatured == null);

	const CATEGORY_LABELS: Record<string, string> = {
		bug_fixes: 'Bug Fix',
		feature_delivery: 'Feature / Patch',
		patch_timing: 'Patch Timing',
		community_events: 'Community Event',
		meta: 'Meta'
	};

	onMount(async () => {
		if (initialFeatured) return; // SSR already provided data
		try {
			featured = await listMarkets('active', '', 0);
		} catch {
			featured = null;
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>ScolyMarket — Star Citizen Prediction Markets</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-12">
	<!-- Hero -->
	<section class="text-center mb-12">
		<p class="text-xs font-bold uppercase tracking-[0.2em] text-primary-600 mb-3">Star Citizen Community</p>
		<h1 class="text-4xl font-bold text-surface-900 mb-3 tracking-tight">ScolyMarket</h1>
		<div class="w-12 h-px bg-primary-400 mx-auto mb-4"></div>
		<p class="text-surface-600 mb-8 max-w-md mx-auto">
			Prediction markets for Star Citizen events and development milestones.
			Bet bUEC on real predictions. No real money involved.
		</p>

		{#if !$isLoggedIn}
			<button onclick={loginWithSCID} class="btn preset-filled-primary-500 px-8 uppercase tracking-widest text-sm">
				Login with SCID
			</button>
		{:else}
			<a href="/markets/new" class="btn preset-filled-primary-500 px-8 uppercase tracking-widest text-sm no-underline">
				+ Submit a Market
			</a>
		{/if}
	</section>

	<!-- Stats bar -->
	<div class="grid grid-cols-2 gap-4 mb-12">
		<div class="sc-card p-5 text-center">
			<div class="text-3xl font-bold text-primary-600 mb-0.5">{featured ? featured.total : '—'}</div>
			<div class="text-surface-500 text-xs uppercase tracking-widest font-semibold">Active Markets</div>
		</div>
		<div class="sc-card p-5 text-center">
			<div class="text-3xl font-bold text-primary-600 mb-0.5">1,000</div>
			<div class="text-surface-500 text-xs uppercase tracking-widest font-semibold">Starting bUEC</div>
		</div>
	</div>

	<!-- Featured markets -->
	<section>
		<div class="flex items-center justify-between mb-5">
			<div class="flex items-center gap-3">
				<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">Featured Markets</h2>
				<div class="flex-1 h-px bg-surface-200 w-16"></div>
			</div>
			<a href="/markets" class="text-primary-600 hover:text-primary-700 text-xs uppercase tracking-wider font-semibold no-underline transition-colors">
				View all →
			</a>
		</div>

		{#if loading}
			<div class="text-surface-400 text-center py-8 text-sm">Loading markets…</div>
		{:else if !featured || featured.markets.length === 0}
			<div class="sc-card p-8 text-center">
				<p class="text-surface-500 text-sm">No active markets yet. Be the first to submit one!</p>
			</div>
		{:else}
			<div class="space-y-2">
				{#each featured.markets.slice(0, 5) as market}
					<a
						href="/markets/{market.id}"
						class="sc-card p-4 flex items-start justify-between gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block"
					>
						<div class="flex-1 min-w-0">
							<div class="mb-1.5">
								<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
							</div>
							<div class="text-surface-800 font-medium text-sm truncate">{market.title}</div>
							<div class="text-surface-500 text-xs mt-1">
								{market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
							</div>
						</div>
						<div class="flex-shrink-0 text-right">
							<div class="text-primary-600 font-bold text-sm">YES</div>
							<div class="text-surface-400 text-xs">50%</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	</section>
</div>
