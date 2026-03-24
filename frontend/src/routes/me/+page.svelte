<script lang="ts">
	import { onMount } from 'svelte';
	import { getMyPositions, getMyTrades } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { Position, TradeWithMarket } from '$lib/types';

	let positions: Position[] = [];
	let trades: TradeWithMarket[] = [];
	let loading = true;

	onMount(async () => {
		if (!$isLoggedIn) return;
		try {
			[positions, trades] = await Promise.all([getMyPositions(), getMyTrades()]);
		} finally {
			loading = false;
		}
	});
</script>

<div class="container mx-auto px-4 max-w-3xl py-8">
	{#if !$isLoggedIn}
		<div class="alert variant-filled-warning">
			Please <a href="/auth/login" class="underline">login</a> to view your profile.
		</div>
	{:else if $currentUser}
		<div class="mb-8">
			<h1 class="h2 text-surface-100">{$currentUser.display_name}</h1>
			<p class="text-surface-400 text-sm mt-1">
				{$currentUser.email} · Joined {new Date($currentUser.created_at).toLocaleDateString()}
			</p>
		</div>

		<div class="grid grid-cols-2 gap-4 mb-8">
			<div class="card variant-glass-surface p-5 rounded-lg text-center">
				<div class="text-3xl font-bold text-primary-400">{$currentUser.balance.toLocaleString()}</div>
				<div class="text-surface-400 text-sm mt-1">ScollyBucks™ Balance</div>
			</div>
			<div class="card variant-glass-surface p-5 rounded-lg text-center">
				<div class="text-3xl font-bold text-secondary-300">{positions.length}</div>
				<div class="text-surface-400 text-sm mt-1">Open Positions</div>
			</div>
		</div>

		<!-- Positions -->
		<section class="mb-8">
			<h2 class="h3 text-surface-200 mb-4">Your Positions</h2>

			{#if loading}
				<div class="text-surface-400 text-sm">Loading…</div>
			{:else if positions.length === 0}
				<div class="card variant-ghost-surface p-6 text-center rounded-lg">
					<p class="text-surface-400 text-sm">No open positions yet.</p>
					<a href="/markets" class="btn btn-sm variant-filled-primary mt-3">Browse Markets</a>
				</div>
			{:else}
				<div class="space-y-2">
					{#each positions as pos}
						<a
							href="/markets/{pos.market_id}"
							class="card variant-glass-surface p-4 rounded-lg flex items-center justify-between hover:variant-glass-primary transition-all"
						>
							<div>
								<div class="text-surface-100 text-sm font-medium">{pos.market_title}</div>
								<div class="text-surface-500 text-xs mt-0.5">
									{#if pos.yes_shares > 0}YES: {pos.yes_shares.toFixed(2)} shares{/if}
									{#if pos.yes_shares > 0 && pos.no_shares > 0} · {/if}
									{#if pos.no_shares > 0}NO: {pos.no_shares.toFixed(2)} shares{/if}
								</div>
							</div>
							<span class="badge variant-filled-surface text-xs">{pos.market_status}</span>
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Trade history -->
		<section>
			<h2 class="h3 text-surface-200 mb-4">Trade History</h2>

			{#if loading}
				<div class="text-surface-400 text-sm">Loading…</div>
			{:else if trades.length === 0}
				<div class="card variant-ghost-surface p-6 text-center rounded-lg">
					<p class="text-surface-400 text-sm">No trades yet.</p>
				</div>
			{:else}
				<div class="card variant-glass-surface rounded-lg overflow-hidden">
					<table class="table w-full text-sm">
						<thead class="bg-surface-800">
							<tr>
								<th class="text-surface-400 text-left py-3 px-4">Market</th>
								<th class="text-surface-400 text-center py-3 px-4">Action</th>
								<th class="text-surface-400 text-right py-3 px-4">Shares</th>
								<th class="text-surface-400 text-right py-3 px-4">Cost</th>
								<th class="text-surface-400 text-right py-3 px-4">Date</th>
							</tr>
						</thead>
						<tbody>
							{#each trades as trade}
								<tr class="border-t border-surface-700">
									<td class="py-3 px-4 text-surface-200 max-w-xs truncate">
										<a href="/markets/{trade.market_id}" class="hover:text-primary-400">
											{trade.market_title}
										</a>
									</td>
									<td class="py-3 px-4 text-center">
										<span class="badge {trade.action === 'buy' ? 'variant-filled-success' : 'variant-filled-warning'} text-xs">
											{trade.action.toUpperCase()} {trade.side.toUpperCase()}
										</span>
									</td>
									<td class="py-3 px-4 text-right text-surface-300 font-mono">{trade.shares.toFixed(2)}</td>
									<td class="py-3 px-4 text-right font-mono {trade.action === 'sell' ? 'text-success-400' : 'text-surface-300'}">
										{trade.action === 'sell' ? '+' : '-'}{trade.cost}
									</td>
									<td class="py-3 px-4 text-right text-surface-500 text-xs">
										{new Date(trade.created_at).toLocaleDateString()}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}
</div>
