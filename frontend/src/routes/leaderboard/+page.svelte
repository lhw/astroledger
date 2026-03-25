<script lang="ts">
	import { onMount } from 'svelte';
	import { getLeaderboard } from '$lib/api';
	import type { LeaderboardRow } from '$lib/types';

	let rows = $state<LeaderboardRow[]>([]);
	let loading = $state(true);
	let error = $state('');

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

<svelte:head>
	<title>Leaderboard — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-8">
	<h1 class="text-3xl font-bold text-surface-100 mb-2">Leaderboard</h1>
	<p class="text-surface-400 text-sm mb-8">
		Top ScollyBucks™ holders. Totally meaningless, which makes it matter more.
	</p>

	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading…</div>
	{:else if error}
		<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300">
			{error}
		</div>
	{:else if rows.length === 0}
		<div class="text-surface-400 text-center py-16">No data yet.</div>
	{:else}
		<div class="card preset-tonal-surface rounded-lg overflow-hidden">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-surface-700">
						<th class="px-4 py-3 text-left text-surface-400 font-medium w-12">#</th>
						<th class="px-4 py-3 text-left text-surface-400 font-medium">Player</th>
						<th class="px-4 py-3 text-right text-surface-400 font-medium">Portfolio</th>
						<th class="px-4 py-3 text-right text-surface-400 font-medium">Balance</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row, i}
						<tr class="border-b border-surface-800 hover:bg-surface-800/40 transition-colors">
							<td class="px-4 py-3">
								{#if i === 0}
									<span class="text-yellow-400 font-bold">🥇</span>
								{:else if i === 1}
									<span class="text-surface-300 font-bold">🥈</span>
								{:else if i === 2}
									<span class="text-amber-600 font-bold">🥉</span>
								{:else}
									<span class="text-surface-500">{i + 1}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-surface-100 font-medium">{row.display_name}</td>
							<td class="px-4 py-3 text-right text-primary-400 font-mono">
								{(row.portfolio_value ?? 0).toLocaleString()}
							</td>
							<td class="px-4 py-3 text-right text-surface-300 font-mono">
								{row.balance.toLocaleString()}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="text-surface-600 text-xs text-center mt-4">
			Portfolio = balance + estimated share value at current prices.
		</p>
	{/if}
</div>
