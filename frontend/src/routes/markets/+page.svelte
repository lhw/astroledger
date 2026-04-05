<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { listMarkets, getMyPositions, getTrendingMarkets } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import Alert from '$lib/components/Alert.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import MarketCard from '$lib/components/MarketCard.svelte';
	import type { MarketList, MarketCategory, TrendingMarket } from '$lib/types';
	import { CATEGORY_FILTER_OPTIONS } from '$lib/categories';

	let { data } = $props<{ data: { markets: MarketList | null } }>();

	// untrack() signals that we intentionally want a one-time snapshot, not a reactive dependency.
	const { markets: initialMarkets } = untrack(() => data);
	let markets = $state<MarketList | null>(initialMarkets ?? null);
	let trendingMarkets = $state<TrendingMarket[] | null>(null);
	let loading = $state(initialMarkets == null);
	let error = $state('');
	let ownedMarketIds = $state(new Set<number>());

	let statusFilter = $state('active');
	let categoryFilter = $state<MarketCategory | ''>('');
	let sortMode = $state<'newest' | 'trending'>('newest');

	// Read ?sort=trending from URL on mount
	onMount(async () => {
		const params = new URLSearchParams(window.location.search);
		if (params.get('sort') === 'trending') {
			sortMode = 'trending';
		}
		if (initialMarkets && sortMode === 'newest') return; // SSR already provided initial data
		await load();
	});

	async function load() {
		loading = true;
		error = '';
		trendingMarkets = null;
		try {
			if (sortMode === 'trending') {
				const [t, positions] = await Promise.all([
					getTrendingMarkets(),
					$isLoggedIn ? getMyPositions() : Promise.resolve([])
				]);
				trendingMarkets = t;
				markets = null;
				ownedMarketIds = new Set((positions as { market_id: number }[]).map((p) => p.market_id));
			} else {
				const fetches: [Promise<MarketList>, Promise<unknown>] = [
					listMarkets(statusFilter, categoryFilter, offset),
					$isLoggedIn ? getMyPositions() : Promise.resolve([])
				];
				const [m, positions] = await Promise.all(fetches);
				markets = m;
				ownedMarketIds = new Set((positions as { market_id: number }[]).map((p) => p.market_id));
			}
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	let offset = $state(0);

	function prev() {
		offset = Math.max(0, offset - 20);
		load();
	}
	function next() {
		offset += 20;
		load();
	}

	function setSortMode(mode: 'newest' | 'trending') {
		sortMode = mode;
		offset = 0;
		load();
	}
</script>

<svelte:head>
	<title>Markets — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-7xl py-10">
	<div class="flex items-center justify-between mb-8">
		<div>
			<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Browse</p>
			<h1 class="text-2xl font-bold text-surface-900 tracking-tight">Markets</h1>
		</div>
		<a href="/markets/new" class="btn btn-sm preset-filled-primary-500 no-underline uppercase tracking-wider text-xs">+ Submit Market</a>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap gap-3 mb-7">
		<!-- Sort mode toggle -->
		<div class="flex rounded-lg border border-surface-300 overflow-hidden text-xs font-semibold uppercase tracking-wider">
			<button
				onclick={() => setSortMode('newest')}
				class="px-3 py-1.5 transition-colors"
				class:bg-primary-500={sortMode === 'newest'}
				class:text-white={sortMode === 'newest'}
				class:text-surface-600={sortMode !== 'newest'}
			>Newest</button>
			<button
				onclick={() => setSortMode('trending')}
				class="px-3 py-1.5 transition-colors border-l border-surface-300"
				class:bg-primary-500={sortMode === 'trending'}
				class:text-white={sortMode === 'trending'}
				class:text-surface-600={sortMode !== 'trending'}
			>🔥 Trending</button>
		</div>

		{#if sortMode === 'newest'}
			<select
				bind:value={statusFilter}
				onchange={() => { offset = 0; load(); }}
				class="sc-input !w-auto !py-1.5 text-sm"
			>
				<option value="active">Active</option>
				<option value="deadline_passed">Deadline Passed</option>
				<option value="resolved">Resolved</option>
				<option value="pending_review">Pending Review</option>
				<option value="cancelled">Cancelled</option>
			</select>

			<select
				bind:value={categoryFilter}
				onchange={() => { offset = 0; load(); }}
				class="sc-input !w-auto !py-1.5 text-sm"
			>
				{#each CATEGORY_FILTER_OPTIONS as cat}
					<option value={cat.value}>{cat.label}</option>
				{/each}
			</select>
		{:else}
			<p class="text-surface-500 text-xs self-center">Active markets ranked by trade volume in the last 24 hours.</p>
		{/if}
	</div>

	{#if loading}
		<EmptyState message="Loading markets…" card={false} padding="py-16" />
	{:else if error}
		<Alert type="error" message={error} />
	{:else if sortMode === 'trending'}
		{#if !trendingMarkets || trendingMarkets.length === 0}
			<EmptyState message="No trading activity in the last 24 hours." />
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-6">
				{#each trendingMarkets as market, i}
					<MarketCard market={market} variant="trending" owned={ownedMarketIds.has(market.id)} rank={i} />
				{/each}
			</div>
		{/if}
	{:else if !markets || markets.markets.length === 0}
		<EmptyState message="No markets found." />
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-6">
			{#each markets.markets as market}
				<MarketCard {market} owned={ownedMarketIds.has(market.id)} />
			{/each}
		</div>

		<!-- Pagination -->
		<div class="flex items-center justify-between">
			<button onclick={prev} disabled={offset === 0} class="btn btn-sm preset-outlined text-xs uppercase tracking-wider">← Prev</button>
			<span class="text-surface-500 text-xs">{offset + 1}–{offset + markets.markets.length} of {markets.total}</span>
			<button
				onclick={next}
				disabled={offset + markets.markets.length >= markets.total}
				class="btn btn-sm preset-outlined text-xs uppercase tracking-wider"
			>
				Next →
			</button>
		</div>
	{/if}
</div>

