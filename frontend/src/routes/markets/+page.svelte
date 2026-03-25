<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { listMarkets, getMyPositions } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import { yesProb } from '$lib/amm';
	import type { MarketList, MarketCategory } from '$lib/types';

	let { data } = $props<{ data: { markets: MarketList | null } }>();

	// untrack() signals that we intentionally want a one-time snapshot, not a reactive dependency.
	const { markets: initialMarkets } = untrack(() => data);
	let markets = $state<MarketList | null>(initialMarkets ?? null);
	let loading = $state(initialMarkets == null);
	let error = $state('');
	let ownedMarketIds = $state(new Set<number>());

	let statusFilter = $state('active');
	let categoryFilter = $state<MarketCategory | ''>('');
	let offset = $state(0);

	const CATEGORIES: { value: MarketCategory | ''; label: string }[] = [
		{ value: '', label: 'All Categories' },
		{ value: 'bug_fixes', label: 'Bug Fixes' },
		{ value: 'feature_delivery', label: 'Feature Delivery' },
		{ value: 'patch_timing', label: 'Patch Timing' },
		{ value: 'community_events', label: 'Community Events' },
		{ value: 'meta', label: 'Meta' }
	];

	async function load() {
		loading = true;
		error = '';
		try {
			const fetches: [Promise<MarketList>, Promise<unknown>] = [
				listMarkets(statusFilter, categoryFilter, offset),
				$isLoggedIn ? getMyPositions() : Promise.resolve([])
			];
			const [m, positions] = await Promise.all(fetches);
			markets = m;
			ownedMarketIds = new Set((positions as { market_id: number }[]).map((p) => p.market_id));
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		if (initialMarkets) return; // SSR already provided initial data
		await load();
	});

	function prev() {
		offset = Math.max(0, offset - 20);
		load();
	}
	function next() {
		offset += 20;
		load();
	}
</script>

<svelte:head>
	<title>Markets — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-10">
	<div class="flex items-center justify-between mb-8">
		<div>
			<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Browse</p>
			<h1 class="text-2xl font-bold text-surface-900 tracking-tight">Markets</h1>
		</div>
		<a href="/markets/new" class="btn btn-sm preset-filled-primary-500 no-underline uppercase tracking-wider text-xs">+ Submit Market</a>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap gap-3 mb-7">
		<select
			bind:value={statusFilter}
			onchange={() => { offset = 0; load(); }}
			class="sc-input !w-auto !py-1.5 text-sm"
		>
			<option value="active">Active</option>
			<option value="resolved">Resolved</option>
			<option value="pending_review">Pending Review</option>
			<option value="cancelled">Cancelled</option>
		</select>

		<select
			bind:value={categoryFilter}
			onchange={() => { offset = 0; load(); }}
			class="sc-input !w-auto !py-1.5 text-sm"
		>
			{#each CATEGORIES as cat}
				<option value={cat.value}>{cat.label}</option>
			{/each}
		</select>
	</div>

	{#if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading markets…</div>
	{:else if error}
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-4">{error}</div>
	{:else if !markets || markets.markets.length === 0}
		<div class="sc-card p-8 text-center">
			<p class="text-surface-500 text-sm">No markets found.</p>
		</div>
	{:else}
		<div class="space-y-2 mb-6">
			{#each markets.markets as market}
				{@const owned = ownedMarketIds.has(market.id)}
				{@const prob = yesProb(market.liquidity_param, market.yes_shares, market.no_shares)}
				{@const yesPct = Math.round(prob * 100)}
				<a
					href="/markets/{market.id}"
					class="p-4 flex items-start justify-between gap-4 transition-all no-underline block rounded-lg border
						{owned
							? 'bg-amber-50 border-primary-300 shadow-sm hover:border-primary-400 hover:shadow-md'
							: 'bg-white border-surface-200 shadow-[0_1px_4px_0_rgba(0,0,0,0.06)] hover:border-primary-300 hover:shadow-md'}"
				>
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-1.5">
							<span class="sc-tag">{market.category.replace('_', ' ')}</span>
							{#if market.status === 'resolved'}
								<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-green-100 text-green-700 border border-green-200">Resolved</span>
							{/if}
							{#if market.status === 'cancelled'}
								<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-surface-100 text-surface-500 border border-surface-200">Cancelled</span>
							{/if}
							{#if owned}
								<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Your Position</span>
							{/if}
						</div>
						<div class="text-surface-800 font-medium text-sm">{market.title}</div>
						<div class="text-surface-500 text-xs mt-1">
							{market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
						</div>
						<!-- Probability bar -->
						{#if market.status === 'active'}
							<div class="mt-2.5">
								<div class="flex justify-between text-[10px] font-bold uppercase tracking-wider mb-1">
									<span class="text-green-600">YES {yesPct}%</span>
									<span class="text-red-500">NO {100 - yesPct}%</span>
								</div>
								<div class="h-1.5 w-full rounded-full bg-red-100 overflow-hidden">
									<div class="h-full rounded-full bg-green-500 transition-all" style="width: {yesPct}%"></div>
								</div>
							</div>
						{/if}
					</div>
					<!-- Right: YES price pill -->
					{#if market.status === 'active'}
						<div class="flex-shrink-0 flex flex-col items-center justify-center min-w-[52px]">
							<div class="text-xl font-black leading-none {yesPct >= 50 ? 'text-green-600' : 'text-red-500'}">{yesPct}%</div>
							<div class="text-[10px] text-surface-400 font-semibold uppercase tracking-wider mt-0.5">YES</div>
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
