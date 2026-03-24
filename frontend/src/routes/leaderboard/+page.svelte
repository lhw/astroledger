<script lang="ts">
	import { onMount } from 'svelte';
	import { getLeaderboard } from '$lib/api';
	import type { LeaderboardRow } from '$lib/types';

	let rows: LeaderboardRow[] = [];
	let loading = true;
	let error = '';

	onMount(async () => {
		try {
			rows = await getLeaderboard(50);
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});
</script>

<div class="container mx-auto px-4 max-w-2xl py-8">
	<h1 class="h2 text-surface-100 mb-6">Leaderboard</h1>
	<p class="text-surface-400 text-sm mb-6">Ranked by balance + portfolio value. ScollyBucks™ only.</p>

	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading…</div>
	{:else if error}
		<div class="alert variant-filled-error">{error}</div>
	{:else if rows.length === 0}
		<div class="card variant-glass-surface p-8 text-center rounded-lg">
			<p class="text-surface-400">No users yet. Be the first!</p>
		</div>
	{:else}
		<div class="card variant-glass-surface rounded-lg overflow-hidden">
			<table class="table table-hover w-full">
				<thead class="bg-surface-800">
					<tr>
						<th class="text-surface-400 text-left text-sm font-medium py-3 px-4 w-12">#</th>
						<th class="text-surface-400 text-left text-sm font-medium py-3 px-4">Pilot</th>
						<th class="text-surface-400 text-right text-sm font-medium py-3 px-4">Balance</th>
						<th class="text-surface-400 text-right text-sm font-medium py-3 px-4">Portfolio</th>
						<th class="text-surface-400 text-right text-sm font-medium py-3 px-4">Total</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row, i}
						{@const portfolioVal = typeof row.portfolio_value === 'number' ? row.portfolio_value : 0}
						{@const total = row.balance + portfolioVal}
						<tr class="border-t border-surface-700 hover:bg-surface-700 transition-colors">
							<td class="py-3 px-4 text-surface-500 text-sm">
								{#if i === 0}🥇
								{:else if i === 1}🥈
								{:else if i === 2}🥉
								{:else}{i + 1}
								{/if}
							</td>
							<td class="py-3 px-4 text-surface-100">{row.display_name}</td>
							<td class="py-3 px-4 text-right text-surface-200 text-sm font-mono">
								{row.balance.toLocaleString()}
							</td>
							<td class="py-3 px-4 text-right text-surface-400 text-sm font-mono">
								+{portfolioVal.toLocaleString()}
							</td>
							<td class="py-3 px-4 text-right text-primary-400 font-bold font-mono">
								{total.toLocaleString()}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
