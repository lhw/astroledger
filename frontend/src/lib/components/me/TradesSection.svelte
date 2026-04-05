<script lang="ts">
	import type { TradeWithMarket } from '$lib/types';

	let {
		tradesLoading,
		trades
	}: {
		tradesLoading: boolean;
		trades: TradeWithMarket[];
	} = $props();
</script>

<section>
	<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Recent Trades</h2>
	{#if tradesLoading}
		<div class="text-surface-400 text-sm py-4">Loading trades…</div>
	{:else if trades.length === 0}
		<div class="text-surface-400 text-sm py-4">No trades yet.</div>
	{:else}
		<div class="sc-card overflow-hidden">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-surface-200">
						<th class="px-4 py-3 text-left text-surface-500 font-bold text-xs uppercase tracking-wider">Market</th>
						<th class="px-4 py-3 text-center text-surface-500 font-bold text-xs uppercase tracking-wider">Outcome</th>
						<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Shares</th>
						<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Cost</th>
					</tr>
				</thead>
				<tbody>
					{#each trades as trade}
						<tr class="border-b border-surface-100 hover:bg-surface-50 transition-colors">
							<td class="px-4 py-3">
								<a
									href="/markets/{trade.market_id}"
									class="text-surface-800 hover:text-primary-600 truncate block max-w-[200px] transition-colors"
								>
									{trade.market_title}
								</a>
							</td>
							<td class="px-4 py-3 text-center">
								<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-primary-100 text-primary-700 border border-primary-200">
									{trade.outcome_label}
								</span>
							</td>
							<td class="px-4 py-3 text-right text-surface-700 font-mono text-sm">{trade.shares}</td>
							<td class="px-4 py-3 text-right text-surface-700 font-mono text-sm">{trade.cost}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>