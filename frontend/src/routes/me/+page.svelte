<script lang="ts">
	import { onMount } from 'svelte';
	import { getMe, getMyPositions, getMyTrades, logout } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { Position, TradeWithMarket } from '$lib/types';

	let positions = $state<Position[]>([]);
	let trades = $state<TradeWithMarket[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		if (!$isLoggedIn) { loading = false; return; }
		try {
			[positions, trades] = await Promise.all([getMyPositions(), getMyTrades(0)]);
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>My Profile — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-8">
	{#if !$isLoggedIn}
		<div class="text-center py-16">
			<h1 class="text-3xl font-bold text-surface-100 mb-4">Not logged in</h1>
			<a href="/auth/login" class="btn preset-filled-primary-500">Login with SCID</a>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16">Loading profile…</div>
	{:else}
		{@const user = $currentUser}

		<!-- Profile header -->
		{#if user}
			<div class="flex items-start justify-between mb-8">
				<div>
					<h1 class="text-3xl font-bold text-surface-100">{user.display_name}</h1>
					<p class="text-surface-400 text-sm mt-1">
						Joined {new Date(user.created_at).toLocaleDateString()}
					</p>
					{#if user.is_moderator || user.is_admin}
						<span class="badge preset-filled-primary-500 text-xs mt-1">
							{user.is_admin ? 'Admin' : 'Moderator'}
						</span>
					{/if}
				</div>
				<div class="text-right">
					<div class="text-2xl font-bold text-primary-400">{user.balance.toLocaleString()}</div>
					<div class="text-surface-400 text-xs">bUEC</div>
					<button onclick={logout} class="btn btn-sm preset-outlined mt-3 text-xs">Logout</button>
				</div>
			</div>
		{/if}

		{#if error}
			<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mb-6">
				{error}
			</div>
		{/if}

		<!-- Positions -->
		<section class="mb-8">
			<h2 class="text-lg font-semibold text-surface-200 mb-3">Open Positions</h2>
			{#if positions.length === 0}
				<div class="text-surface-500 text-sm py-4">No open positions yet. Go make some bets!</div>
			{:else}
				<div class="space-y-2">
					{#each positions as pos}
						<a
							href="/markets/{pos.market_id}"
							class="block card preset-tonal-surface p-4 rounded-lg hover:bg-surface-700/40 transition-colors no-underline"
						>
							<div class="flex justify-between items-start gap-4">
								<div class="flex-1 min-w-0">
									<p class="text-surface-100 text-sm font-medium truncate">{pos.market_title}</p>
									<p class="text-surface-400 text-xs mt-0.5">
										{#if pos.yes_shares > 0}
											<span class="text-success-400">{pos.yes_shares} YES</span>
										{/if}
										{#if pos.yes_shares > 0 && pos.no_shares > 0}
											<span class="text-surface-600 mx-1">·</span>
										{/if}
										{#if pos.no_shares > 0}
											<span class="text-error-400">{pos.no_shares} NO</span>
										{/if}
									</p>
								</div>
								<span class="badge preset-tonal-surface text-xs shrink-0">{pos.market_status}</span>
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Recent trades -->
		<section>
			<h2 class="text-lg font-semibold text-surface-200 mb-3">Recent Trades</h2>
			{#if trades.length === 0}
				<div class="text-surface-500 text-sm py-4">No trades yet.</div>
			{:else}
				<div class="card preset-tonal-surface rounded-lg overflow-hidden">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-surface-700">
								<th class="px-4 py-2 text-left text-surface-400 font-medium">Market</th>
								<th class="px-4 py-2 text-center text-surface-400 font-medium">Side</th>
								<th class="px-4 py-2 text-right text-surface-400 font-medium">Shares</th>
								<th class="px-4 py-2 text-right text-surface-400 font-medium">Cost</th>
							</tr>
						</thead>
						<tbody>
							{#each trades as trade}
								<tr class="border-b border-surface-800">
									<td class="px-4 py-2">
										<a
											href="/markets/{trade.market_id}"
											class="text-surface-200 hover:text-primary-400 truncate block max-w-[200px]"
										>
											{trade.market_title}
										</a>
									</td>
									<td class="px-4 py-2 text-center">
										<span
											class="badge text-xs {trade.side === 'yes'
												? 'preset-filled-success-500'
												: 'preset-filled-error-500'}"
										>
											{trade.side.toUpperCase()} {trade.action}
										</span>
									</td>
									<td class="px-4 py-2 text-right text-surface-300 font-mono">{trade.shares}</td>
									<td class="px-4 py-2 text-right text-surface-300 font-mono">{trade.cost}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}
</div>
