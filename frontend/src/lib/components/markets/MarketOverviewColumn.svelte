<script lang="ts">
	import { renderMarkdown } from '$lib/markdown';
	import MarketCommunitySection from '$lib/components/markets/MarketCommunitySection.svelte';
	import type { Comment, Market, PricePoint, UserOutcomePosition } from '$lib/types';

	let {
		market,
		priceHistory,
		myPositions,
		hasPosition,
		chartLogScale = $bindable(),
		comments,
		requestingResolution,
		resolutionRequestMsg,
		resolutionLink = $bindable(),
		resolutionNote = $bindable(),
		showReportForm = $bindable(),
		reportReason = $bindable(),
		submittingReport,
		reportMsg,
		commentInput = $bindable(),
		postingComment,
		commentError,
		onRequestResolution,
		onSubmitReport,
		onPostComment,
		onDeleteComment
	}: {
		market: Market;
		priceHistory: PricePoint[];
		myPositions: UserOutcomePosition[];
		hasPosition: boolean;
		chartLogScale: boolean;
		comments: Comment[];
		requestingResolution: boolean;
		resolutionRequestMsg: string;
		resolutionLink: string;
		resolutionNote: string;
		showReportForm: boolean;
		reportReason: string;
		submittingReport: boolean;
		reportMsg: string;
		commentInput: string;
		postingComment: boolean;
		commentError: string;
		onRequestResolution: () => Promise<void>;
		onSubmitReport: () => Promise<void>;
		onPostComment: () => Promise<void>;
		onDeleteComment: (commentId: number) => Promise<void>;
	} = $props();
</script>

<div class="md:col-span-2 space-y-4">
	{#if market.description}
		<div class="sc-card p-5">
			<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Description</h3>
			<div class="text-surface-700 text-sm leading-relaxed prose prose-sm max-w-none">{@html renderMarkdown(market.description)}</div>
		</div>
	{/if}

	{#if market.resolution_criteria}
		<div class="sc-card p-5">
			<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Resolution Criteria</h3>
			<div class="text-surface-700 text-sm leading-relaxed prose prose-sm max-w-none">{@html renderMarkdown(market.resolution_criteria)}</div>
		</div>
	{/if}

	<div class="sc-card p-5">
		<div class="flex items-center justify-between mb-3">
			<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em]">Price History</h3>
			{#if priceHistory.length > 1}
				<button
					onclick={() => (chartLogScale = !chartLogScale)}
					class="text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border transition-colors {chartLogScale ? 'border-primary-400 text-primary-600 bg-primary-50' : 'border-surface-300 text-surface-400 hover:border-surface-400'}"
					title="Toggle log scale"
				>Log</button>
			{/if}
		</div>
		{#if priceHistory.length > 1}
			{@const W = 600}
			{@const H = 100}
			{@const n = priceHistory.length}
			{@const scaleP = (p: number) => chartLogScale ? Math.log1p(p * 9) / Math.log1p(9) : p}
			{@const yOf = (p: number) => H - scaleP(Math.max(0.001, Math.min(0.999, p))) * H}
			{@const xOf = (index: number) => (index / (n - 1)) * W}
			{@const lastPrice = priceHistory.at(-1)?.price_at_trade ?? 0}
			{@const resolved = market.resolved_outcome_id !== null}
			{@const outcomeLabels = [...new Set(priceHistory.map((point) => point.outcome_label))]}
			<div class="flex flex-wrap items-center gap-3 mb-2">
				{#each outcomeLabels as label, index}
					{@const colors = ['#D4A843', '#f87171', '#60a5fa', '#34d399', '#a78bfa']}
					<span class="flex items-center gap-1 text-xs font-semibold text-surface-600">
						<span class="inline-block w-3 h-0.5 rounded" style="background:{colors[index % colors.length]}"></span>
						{label}
					</span>
				{/each}
				<span class="text-surface-400 text-xs ml-auto">{priceHistory.length} trades</span>
			</div>
			<svg viewBox="0 0 {W} {H}" class="w-full rounded" style="height:100px;background:#fafaf8">
				<line x1="0" y1={yOf(0.5)} x2={W} y2={yOf(0.5)} stroke="#e5e7eb" stroke-width="1" stroke-dasharray="4 4"/>
				{#each outcomeLabels as label, index}
					{@const colors = ['#D4A843', '#f87171', '#60a5fa', '#34d399', '#a78bfa']}
					{@const points = priceHistory
						.map((point, pointIndex) => point.outcome_label === label ? `${xOf(pointIndex)},${yOf(point.price_at_trade)}` : null)
						.filter(Boolean)
						.join(' ')}
					<polyline points={points} fill="none" stroke={colors[index % colors.length]} stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>
				{/each}
				{#if resolved}
					<line x1={W} y1="0" x2={W} y2={H} stroke="#16a34a" stroke-width="2" stroke-dasharray="4 3" opacity="0.8"/>
				{/if}
				<circle cx={W} cy={yOf(lastPrice)} r="4" fill="#D4A843"/>
			</svg>
			<div class="flex justify-between mt-1">
				{#each [0, 0.25, 0.5, 0.75, 1] as fraction}
					{@const index = Math.round(fraction * (n - 1))}
					<span class="text-surface-400 text-[10px]">
						{new Date(priceHistory[index].created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
					</span>
				{/each}
			</div>
		{:else}
			<p class="text-surface-400 text-xs text-center py-4">No trades yet — be the first!</p>
		{/if}
	</div>

	{#if hasPosition}
		<div class="sc-card p-5 border-l-4 border-l-primary-400 bg-amber-50/30">
			<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Your Position</h3>
			<div class="flex gap-3 flex-wrap">
				{#each myPositions.filter((position) => position.shares > 0) as position}
					<div class="flex-1 text-center p-3 rounded bg-primary-50 border border-primary-100 min-w-20">
						<div class="text-2xl font-bold text-primary-700">{Math.floor(position.shares)}</div>
						<div class="text-primary-600 text-xs mt-0.5 uppercase tracking-wider font-semibold">{position.label}</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<MarketCommunitySection
		{hasPosition}
		marketStatus={market.status}
		{comments}
		{requestingResolution}
		{resolutionRequestMsg}
		bind:resolutionLink
		bind:resolutionNote
		bind:showReportForm
		bind:reportReason
		{submittingReport}
		{reportMsg}
		bind:commentInput
		{postingComment}
		{commentError}
		onRequestResolution={onRequestResolution}
		onSubmitReport={onSubmitReport}
		onPostComment={onPostComment}
		onDeleteComment={onDeleteComment}
	/>
</div>