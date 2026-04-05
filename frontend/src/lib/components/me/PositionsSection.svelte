<script lang="ts">
	import type { Position } from '$lib/types';

	let {
		positionsLoading,
		openPositions,
		resolvedPositions
	}: {
		positionsLoading: boolean;
		openPositions: Position[];
		resolvedPositions: Position[];
	} = $props();
</script>

<section class="mb-8">
	<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Open Positions</h2>
	{#if positionsLoading}
		<div class="text-surface-400 text-sm py-4">Loading positions…</div>
	{:else if openPositions.length === 0}
		<div class="text-surface-400 text-sm py-4">No open positions yet. Go make some bets!</div>
	{:else}
		<div class="space-y-2 mb-6">
			{#each openPositions as pos}
				<a
					href="/markets/{pos.market_id}"
					class="sc-card p-4 flex justify-between items-start gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block"
				>
					<div class="flex-1 min-w-0">
						<p class="text-surface-800 text-sm font-medium truncate">{pos.market_title}</p>
						<p class="text-surface-500 text-xs mt-0.5">
							<span class="font-semibold text-primary-700">{pos.shares}</span>
							<span class="text-surface-400 ml-1">{pos.outcome_label}</span>
						</p>
					</div>
					<div class="flex flex-col items-end gap-1 shrink-0">
						<span class="sc-tag-status">{pos.market_status}</span>
						{#if pos.cost_basis > 0}
							<span class="text-[10px] text-surface-400">Cost {pos.cost_basis.toLocaleString()} bUEC</span>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}

	{#if resolvedPositions.length > 0}
		<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4 mt-2">Resolved Positions</h2>
		<div class="space-y-2">
			{#each resolvedPositions as pos}
				{@const won = pos.resolved_outcome_id !== null && pos.outcome_id === pos.resolved_outcome_id}
				{@const payout = won ? pos.shares * 100 : 0}
				{@const gain = payout - pos.cost_basis}
				<a
					href="/markets/{pos.market_id}"
					class="sc-card p-4 flex justify-between items-start gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block {won ? 'border-l-2 border-l-green-500' : 'border-l-2 border-l-red-700'}"
				>
					<div class="flex-1 min-w-0">
						<p class="text-surface-800 text-sm font-medium truncate">{pos.market_title}</p>
						<p class="text-surface-500 text-xs mt-0.5">
							<span class="font-semibold text-primary-700">{pos.shares}</span>
							<span class="text-surface-400 ml-1">{pos.outcome_label}</span>
							<span class="ml-2 text-surface-500">· Cost {pos.cost_basis.toLocaleString()} bUEC</span>
						</p>
					</div>
					<div class="flex flex-col items-end gap-1 shrink-0">
						{#if won}
							<span class="text-xs font-bold text-green-400">+{payout.toLocaleString()} bUEC</span>
							<span class="text-[10px] {gain >= 0 ? 'text-green-600' : 'text-red-400'}">{gain >= 0 ? '+' : ''}{gain.toLocaleString()} net</span>
						{:else}
							<span class="text-xs font-bold text-red-400">−{pos.cost_basis.toLocaleString()} bUEC</span>
							<span class="text-[10px] text-surface-500">lost</span>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>