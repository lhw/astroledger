<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { listMarkets, getMyPositions, getTrendingMarkets } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	// yesProb no longer needed — prices come from outcomes array
	import type { MarketList, MarketCategory, TrendingMarket } from '$lib/types';
	import { CATEGORY_LABELS, CATEGORY_FILTER_OPTIONS } from '$lib/categories';

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
	<title>Markets — ScolyMarket</title>
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
		<div class="text-surface-400 text-center py-16 text-sm">Loading markets…</div>
	{:else if error}
		<div class="p-4 rounded-lg text-sm mb-4 border border-red-500/40 bg-red-500/10 text-red-400">{error}</div>
	{:else if sortMode === 'trending'}
		{#if !trendingMarkets || trendingMarkets.length === 0}
			<div class="sc-card p-8 text-center">
				<p class="text-surface-500 text-sm">No trading activity in the last 24 hours.</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-6">
				{#each trendingMarkets as market, i}
					{@const owned = ownedMarketIds.has(market.id)}
					{@const outs = market.outcomes ?? []}
					{@const yesPct = outs.length >= 1 ? outs[0].price : 50}
					{@const firstLabel = outs[0]?.label ?? 'YES'}
					{@const secondLabel = outs[1]?.label ?? 'NO'}
					{@const secondPct = outs[1]?.price ?? (100 - yesPct)}
					<a
						href="/markets/{market.id}"
						class="market-card p-3 flex flex-col gap-2 transition-all no-underline rounded-lg border h-full"
						class:market-card-owned={owned}
					>
						<div>
							<div class="flex items-center gap-1.5 mb-1 flex-wrap">
								<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">#{i + 1}</span>
								<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
								{#if owned}
									<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Yours</span>
								{/if}
							</div>
							<div class="text-surface-800 font-medium text-sm leading-snug line-clamp-2">{market.title}</div>
							<div class="text-surface-500 text-[11px] mt-1">
								{market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()} · <span class="text-primary-600 font-medium">{market.recent_trade_count} trades / 24h</span>
							</div>
						</div>
						<div class="mt-auto space-y-1.5">
							<div class="flex items-center justify-between gap-1.5 flex-wrap">
								<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-[11px] font-bold uppercase tracking-wider">
									{firstLabel} {yesPct}%
								</span>
								{#if outs.length === 2}
									<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-red-100 text-red-600 text-[11px] font-bold uppercase tracking-wider">
										{secondLabel} {secondPct}%
									</span>
								{/if}
							</div>
							<div class="h-2 sm:h-1.5 w-full rounded-full bg-red-100 overflow-hidden">
								<div class="h-full rounded-full bg-green-500 transition-all" style="width: {yesPct}%"></div>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	{:else if !markets || markets.markets.length === 0}
		<div class="sc-card p-8 text-center">
			<p class="text-surface-500 text-sm">No markets found.</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-6">
			{#each markets.markets as market}
				{@const owned = ownedMarketIds.has(market.id)}
				{@const outs = market.outcomes ?? []}
				{@const prob = outs.length >= 2 ? outs[0].price : 50}
				{@const yesPct = prob}
				<a
					href="/markets/{market.id}"
					class="market-card p-3 flex flex-col gap-2 transition-all no-underline rounded-lg border h-full"
					class:market-card-owned={owned}
				>
					<div>
						<div class="flex items-center gap-1.5 mb-1 flex-wrap">
							<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
							{#if market.status === 'resolved'}
								<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-green-100 text-green-700 border border-green-200">Resolved</span>
							{/if}
							{#if market.status === 'cancelled'}
								<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-surface-100 text-surface-500 border border-surface-200">Cancelled</span>
							{/if}
							{#if owned}
								<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Yours</span>
							{/if}
						</div>
						<div class="text-surface-800 font-medium text-sm leading-snug line-clamp-2">{market.title}</div>
						<div class="text-surface-500 text-[11px] mt-1">
							{market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}{#if (market.comment_count ?? 0) > 0} · 💬 {market.comment_count}{/if}
						</div>
					</div>
					{#if market.status === 'active'}
						{@const firstLabel = outs[0]?.label ?? 'YES'}
						{@const secondLabel = outs[1]?.label ?? 'NO'}
						{@const secondPct = outs[1]?.price ?? (100 - yesPct)}
						<div class="mt-auto space-y-1.5">
							<div class="flex items-center justify-between gap-1.5 flex-wrap">
								<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-[11px] font-bold uppercase tracking-wider">
									{firstLabel} {yesPct}%
								</span>
								{#if outs.length === 2}
									<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-red-100 text-red-600 text-[11px] font-bold uppercase tracking-wider">
										{secondLabel} {secondPct}%
									</span>
								{/if}
							</div>
							<div class="h-2 sm:h-1.5 w-full rounded-full bg-red-100 overflow-hidden">
								<div class="h-full rounded-full bg-green-500 transition-all" style="width: {yesPct}%"></div>
							</div>
						</div>
					{/if}
				</a>
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

<style>
	.market-card {
		background: var(--card-bg);
		border-color: var(--color-surface-300);
		box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.06);
	}

	.market-card:hover {
		border-color: var(--color-primary-300);
		box-shadow: 0 4px 14px 0 rgba(0, 0, 0, 0.14);
	}

	.market-card-owned {
		background: color-mix(in oklch, var(--card-bg) 78%, var(--color-primary-300) 22%);
		border-color: var(--color-primary-300);
	}

	:global(:root[data-theme='dark']) .market-card {
		box-shadow: 0 2px 10px 0 rgba(0, 0, 0, 0.35);
	}
</style>
