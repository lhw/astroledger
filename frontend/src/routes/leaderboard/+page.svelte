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
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Leaderboard — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-10">
	<div class="mb-7">
		<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Rankings</p>
		<h1 class="text-2xl font-bold text-surface-900 tracking-tight">Leaderboard</h1>
		<p class="text-surface-500 text-sm mt-1">
			Top bUEC holders by portfolio value.
		</p>
	</div>

	{#if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading…</div>
	{:else if error}
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
			{error}
		</div>
	{:else if rows.length === 0}
		<div class="text-surface-400 text-center py-16 text-sm">No data yet.</div>
	{:else}
		<div class="sc-card overflow-hidden">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-surface-200 bg-surface-50">
						<th class="px-4 py-3 text-left text-surface-500 font-bold text-xs uppercase tracking-wider w-12">#</th>
						<th class="px-4 py-3 text-left text-surface-500 font-bold text-xs uppercase tracking-wider">Player</th>
						<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Portfolio</th>
						<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Balance</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row, i}
						<tr class="border-b border-surface-100 hover:bg-surface-50 transition-colors">
							<td class="px-4 py-3">
								{#if i === 0}
									<span class="text-yellow-500 font-bold">🥇</span>
								{:else if i === 1}
									<span class="text-surface-500 font-bold">🥈</span>
								{:else if i === 2}
									<span class="text-amber-600 font-bold">🥉</span>
								{:else}
									<span class="text-surface-400 text-xs">{i + 1}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-surface-800 font-medium">{row.display_name}</td>
							<td class="px-4 py-3 text-right text-primary-600 font-mono font-semibold">
								{(row.portfolio_value ?? 0).toLocaleString()}
							</td>
							<td class="px-4 py-3 text-right text-surface-600 font-mono">
								{row.balance.toLocaleString()}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="text-surface-400 text-xs text-center mt-4">
			Portfolio = balance + estimated share value at current prices.
		</p>
	{/if}
</div>
