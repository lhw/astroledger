<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { listMarkets, getTrendingMarkets } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import { loginWithSCID } from '$lib/api';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import MarketCard from '$lib/components/MarketCard.svelte';
	import type { MarketList, TrendingMarket } from '$lib/types';

	let { data } = $props<{ data: { featured: MarketList | null; trending: TrendingMarket[] | null } }>();

	// untrack() signals that we intentionally want a one-time snapshot, not a reactive dependency.
	const { featured: initialFeatured, trending: initialTrending } = untrack(() => data);
	let featured = $state<MarketList | null>(initialFeatured ?? null);
	let trending = $state<TrendingMarket[] | null>(initialTrending ?? null);
	let loading = $state(initialFeatured == null);
	let trendingLoading = $state(initialTrending == null);

	// Trending strip scroll / drag state
	let trendingEl = $state<HTMLDivElement | null>(null);
	let isDragging = false;
	let dragStartX = 0;
	let scrollStartLeft = 0;

	function onPointerDown(e: PointerEvent) {
		if (!trendingEl) return;
		isDragging = true;
		dragStartX = e.clientX;
		scrollStartLeft = trendingEl.scrollLeft;
		trendingEl.setPointerCapture(e.pointerId);
		trendingEl.style.cursor = 'grabbing';
	}

	function onPointerMove(e: PointerEvent) {
		if (!isDragging || !trendingEl) return;
		const dx = e.clientX - dragStartX;
		trendingEl.scrollLeft = scrollStartLeft - dx;
	}

	function onPointerUp(e: PointerEvent) {
		if (!trendingEl) return;
		isDragging = false;
		trendingEl.style.cursor = 'grab';
		trendingEl.releasePointerCapture(e.pointerId);
	}

	function onWheel(e: WheelEvent) {
		if (!trendingEl) return;
		// Convert vertical wheel delta to horizontal scroll
		if (Math.abs(e.deltaX) > Math.abs(e.deltaY)) return; // let native horizontal scroll pass
		e.preventDefault();
		trendingEl.scrollLeft += e.deltaY;
	}

	onMount(async () => {
		const fetches: Promise<void>[] = [];
		if (!initialFeatured) {
			fetches.push(
				listMarkets('active', '', 0)
					.then((r) => { featured = r; })
					.catch(() => { featured = null; })
					.finally(() => { loading = false; })
			);
		}
		if (!initialTrending) {
			fetches.push(
				getTrendingMarkets()
					.then((r) => { trending = r; })
					.catch(() => { trending = null; })
					.finally(() => { trendingLoading = false; })
			);
		}
		await Promise.all(fetches);
	});

	// Non-passive wheel listener so we can call preventDefault to redirect
	// vertical scroll into horizontal scroll on the trending strip.
	$effect(() => {
		const el = trendingEl;
		if (!el) return;
		el.addEventListener('wheel', onWheel, { passive: false });
		return () => el.removeEventListener('wheel', onWheel);
	});
</script>

<svelte:head>
	<title>AstroLedger — Star Citizen Prediction Markets</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-12">
	<!-- Hero -->
	<section class="text-center mb-12">
		<p class="text-xs font-bold uppercase tracking-[0.2em] text-primary-600 mb-3">Star Citizen Community</p>
		<h1 class="text-4xl font-bold text-surface-900 mb-3 tracking-tight">AstroLedger</h1>
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

	<!-- Trending markets -->
	{#if trendingLoading}
		<section class="mb-10">
			<div class="flex items-center gap-3 mb-4">
				<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">🔥 Trending Now</h2>
				<div class="flex-1 h-px bg-surface-200 w-16"></div>
			</div>
			<EmptyState message="Loading…" card={false} padding="py-4" />
		</section>
	{:else if trending && trending.length > 0}
		<section class="mb-10">
			<div class="flex items-center justify-between mb-4">
				<div class="flex items-center gap-3">
					<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">🔥 Trending Now</h2>
					<div class="flex-1 h-px bg-surface-200 w-16"></div>
				</div>
				<a href="/markets?sort=trending" class="text-primary-600 hover:text-primary-700 text-xs uppercase tracking-wider font-semibold no-underline transition-colors">
					See all →
				</a>
			</div>

			<div
				bind:this={trendingEl}
				role="list"
				class="trending-strip flex gap-3 snap-x"
				onpointerdown={onPointerDown}
				onpointermove={onPointerMove}
				onpointerup={onPointerUp}
				onpointercancel={onPointerUp}
			>
				{#each trending as market}
					<div class="trending-card snap-start flex-shrink-0 w-56">
						<MarketCard {market} variant="trending" />
					</div>
				{/each}
			</div>
		</section>
	{/if}

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
			<EmptyState message="Loading markets…" card={false} padding="py-8" />
		{:else if !featured || featured.markets.length === 0}
			<EmptyState message="No active markets yet. Be the first to submit one!" />
		{:else}
			<div class="space-y-2">
				{#each featured.markets.slice(0, 5) as market}
					<MarketCard {market} variant="featured" />
				{/each}
			</div>
		{/if}
	</section>
</div>
<style>
	.trending-strip {
		overflow-x: auto;
		scrollbar-width: none; /* Firefox */
		-ms-overflow-style: none; /* IE/Edge */
		cursor: grab;
		user-select: none;
		-webkit-user-select: none;
		padding-bottom: 2px;
	}
	.trending-strip::-webkit-scrollbar {
		display: none; /* Chrome/Safari/Opera */
	}
	.trending-card {
		background: var(--card-bg);
		border-color: var(--color-surface-300);
		box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.06);
	}
	.trending-card:hover {
		border-color: var(--color-primary-300);
		box-shadow: 0 4px 14px 0 rgba(0, 0, 0, 0.14);
	}
	:global(:root[data-theme='dark']) .trending-card {
		box-shadow: 0 2px 8px 0 rgba(0, 0, 0, 0.35);
	}
</style>