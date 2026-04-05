<script lang="ts">
	import { CATEGORY_LABELS } from '$lib/categories';
	import { formatDate } from '$lib/format';
	import MarketStatusBadge from '$lib/components/MarketStatusBadge.svelte';
	import type { Market, TrendingMarket } from '$lib/types';

	let {
		market,
		variant = 'grid',
		owned = false,
		rank
	}: {
		market: Market | TrendingMarket;
		variant?: 'featured' | 'grid' | 'trending';
		owned?: boolean;
		rank?: number;
	} = $props();

	const outcomes = $derived(market.outcomes ?? []);
	const firstPct = $derived(outcomes[0]?.price ?? 50);
	const secondPct = $derived(outcomes[1]?.price ?? (100 - firstPct));
	const firstLabel = $derived(outcomes[0]?.label ?? 'YES');
	const secondLabel = $derived(outcomes[1]?.label ?? 'NO');
	const isTrending = $derived('recent_trade_count' in market);
	const recentTradeCount = $derived(isTrending ? (market as TrendingMarket).recent_trade_count : 0);
</script>

{#if variant === 'featured'}
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
				{market.creator_name} · closes {formatDate(market.resolution_deadline)}
			</div>
		</div>
		<div class="flex-shrink-0 text-right">
			<div class="text-primary-600 font-bold text-sm">{firstLabel}</div>
			<div class="text-surface-400 text-xs">{firstPct}%</div>
		</div>
	</a>
{:else}
	<a
		href="/markets/{market.id}"
		class="market-card p-3 flex flex-col gap-2 transition-all no-underline rounded-lg border h-full"
		class:market-card-owned={owned}
	>
		<div>
			<div class="flex items-center gap-1.5 mb-1 flex-wrap">
				{#if variant === 'trending' && rank !== undefined}
					<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">#{rank + 1}</span>
				{/if}
				<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
				{#if variant === 'grid' && market.status !== 'active'}
					<MarketStatusBadge status={market.status} />
				{/if}
				{#if owned}
					<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Yours</span>
				{/if}
			</div>
			<div class="text-surface-800 font-medium text-sm leading-snug line-clamp-2">{market.title}</div>
			<div class="text-surface-500 text-[11px] mt-1">
				{market.creator_name} · closes {formatDate(market.resolution_deadline)}
				{#if variant === 'trending' && isTrending}
					· <span class="text-primary-600 font-medium">{recentTradeCount} trades / 24h</span>
				{:else if variant === 'grid' && (market.comment_count ?? 0) > 0}
					· 💬 {market.comment_count}
				{/if}
			</div>
		</div>
		{#if variant === 'trending' || market.status === 'active'}
			<div class="mt-auto space-y-1.5">
				<div class="flex items-center justify-between gap-1.5 flex-wrap">
					<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-[11px] font-bold uppercase tracking-wider">
						{firstLabel} {firstPct}%
					</span>
					{#if outcomes.length === 2}
						<span class="inline-flex items-center px-2 py-0.5 rounded-full bg-red-100 text-red-600 text-[11px] font-bold uppercase tracking-wider">
							{secondLabel} {secondPct}%
						</span>
					{/if}
				</div>
				<div class="h-2 sm:h-1.5 w-full rounded-full bg-red-100 overflow-hidden">
					<div class="h-full rounded-full bg-green-500 transition-all" style="width: {firstPct}%"></div>
				</div>
			</div>
		{/if}
	</a>
{/if}

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