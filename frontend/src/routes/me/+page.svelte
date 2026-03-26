<script lang="ts">
	import { onMount } from 'svelte';
	import { getMe, getMyPositions, getMyTrades, getMyBadges, logout } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import type { Position, TradeWithMarket, Badge } from '$lib/types';

	let positions = $state<Position[]>([]);
	let trades = $state<TradeWithMarket[]>([]);
	let badges = $state<Badge[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		if (!$isLoggedIn) { loading = false; return; }
		try {
			[positions, trades, badges] = await Promise.all([getMyPositions(), getMyTrades(0), getMyBadges()]);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>My Profile — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-10">
	{#if !$isLoggedIn}
		<div class="text-center py-16">
			<h1 class="text-2xl font-bold text-surface-900 mb-4">Not logged in</h1>
			<a href="/auth/login" class="btn preset-filled-primary-500 uppercase tracking-wider text-xs">Login with SCID</a>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading profile…</div>
	{:else}
		{@const user = $currentUser}

		<!-- Profile header -->
		{#if user}
			<div class="flex items-start justify-between mb-8 pb-6 border-b border-surface-200">
				<div class="flex items-start gap-4">
					<UserAvatar src={user.avatar_url} name={user.display_name} size={56} />
					<div>
						<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Profile</p>
						<h1 class="text-2xl font-bold text-surface-900">{user.display_name}</h1>
						<p class="text-surface-500 text-sm mt-1">
							Joined {new Date(user.created_at).toLocaleDateString()}
						</p>
						{#if user.is_moderator || user.is_admin}
							<span class="sc-tag mt-2 inline-flex">
								{user.is_admin ? 'Admin' : 'Moderator'}
							</span>
						{/if}
					</div>
				</div>
				<div class="text-right">
					<div class="text-3xl font-bold text-primary-600">{user.balance.toLocaleString()}</div>
					<div class="text-surface-500 text-xs uppercase tracking-widest font-semibold">bUEC</div>
					<button onclick={logout} class="border border-surface-300 text-surface-500 hover:border-surface-500 hover:text-surface-700 transition-colors rounded px-3 py-1 text-xs uppercase tracking-wider mt-3">
						Logout
					</button>
				</div>
			</div>

			<!-- RSI Identity -->
			{#if user.rsi_handle || user.is_rsi_verified}
				<section class="mb-8">
					<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">RSI Identity</h2>
					<div class="sc-card px-4 py-4 flex flex-wrap gap-6 items-center">
						{#if user.rsi_handle}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Handle</p>
								<a
									href="https://robertsspaceindustries.com/citizens/{user.rsi_handle}"
									target="_blank"
									rel="noopener noreferrer"
									class="text-primary-600 font-semibold hover:underline"
								>
									{user.rsi_handle}
								</a>
							</div>
						{/if}
						{#if user.rsi_citizen_record}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Citizen Record</p>
								<p class="text-surface-800 font-medium">#{user.rsi_citizen_record}</p>
							</div>
						{/if}
						{#if user.rsi_enlisted}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Enlisted</p>
								<p class="text-surface-800 font-medium">{user.rsi_enlisted}</p>
							</div>
						{/if}
						{#if user.is_rsi_verified}
							<div class="ml-auto">
								<span class="sc-tag text-green-700 bg-green-50 border-green-200">✓ RSI Verified</span>
							</div>
						{/if}
					</div>
				</section>
			{/if}
		{/if}

		{#if error}
			<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-6">
				{error}
			</div>
		{/if}

		<!-- Badges -->
		<section class="mb-8">
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Badges</h2>
			{#if badges.length === 0}
				<div class="text-surface-400 text-sm py-4">
					No badges yet. Trade more to earn them!
				</div>
			{:else}
				<div class="flex flex-wrap gap-3">
					{#each badges as badge}
						<div class="sc-card px-4 py-3 flex flex-col gap-0.5 min-w-[160px]">
							<span class="text-sm font-bold text-primary-700">{badge.title}</span>
							<span class="text-xs text-surface-600">{badge.description}</span>
							<span class="text-[10px] text-surface-400 mt-1">
								Earned {new Date(badge.awarded_at).toLocaleDateString()}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Positions -->
		<section class="mb-8">
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Open Positions</h2>
			{#if positions.length === 0}
				<div class="text-surface-400 text-sm py-4">No open positions yet. Go make some bets!</div>
			{:else}
				<div class="space-y-2">
					{#each positions as pos}
						<a
							href="/markets/{pos.market_id}"
							class="sc-card p-4 flex justify-between items-start gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block"
						>
							<div class="flex-1 min-w-0">
								<p class="text-surface-800 text-sm font-medium truncate">{pos.market_title}</p>
								<p class="text-surface-500 text-xs mt-0.5">
									{#if pos.yes_shares > 0}
										<span class="text-green-600 font-semibold">{pos.yes_shares} YES</span>
									{/if}
									{#if pos.yes_shares > 0 && pos.no_shares > 0}
										<span class="text-surface-300 mx-1">·</span>
									{/if}
									{#if pos.no_shares > 0}
										<span class="text-red-500 font-semibold">{pos.no_shares} NO</span>
									{/if}
								</p>
							</div>
							<span class="sc-tag-status shrink-0">{pos.market_status}</span>
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Recent trades -->
		<section>
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Recent Trades</h2>
			{#if trades.length === 0}
				<div class="text-surface-400 text-sm py-4">No trades yet.</div>
			{:else}
				<div class="sc-card overflow-hidden">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-surface-200">
								<th class="px-4 py-3 text-left text-surface-500 font-bold text-xs uppercase tracking-wider">Market</th>
								<th class="px-4 py-3 text-center text-surface-500 font-bold text-xs uppercase tracking-wider">Side</th>
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
										<span
											class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider {trade.side === 'yes'
												? 'bg-green-100 text-green-700 border border-green-200'
												: 'bg-red-50 text-red-600 border border-red-200'}"
										>
											{trade.side.toUpperCase()}
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
	{/if}
</div>
