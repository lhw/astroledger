<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { getMarket, executeTrade } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { MarketWithPrice } from '$lib/types';

	const id = $derived(Number($page.params.id));

	let data = $state<MarketWithPrice | null>(null);
	let loading = $state(true);
	let error = $state('');

	let tradeSide = $state<'yes' | 'no'>('yes');
	let tradeShares = $state(1);
	let trading = $state(false);
	let tradeError = $state('');
	let tradeSuccess = $state('');

	onMount(async () => {
		try {
			data = await getMarket(id);
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});

	async function doTrade() {
		if (!$isLoggedIn || !data) return;
		trading = true;
		tradeError = '';
		tradeSuccess = '';
		try {
			const result = await executeTrade(id, tradeSide, 'buy', tradeShares);
							tradeSuccess = `Bought ${tradeShares} ${tradeSide.toUpperCase()} shares for ${result.Cost} bUEC. New balance: ${result.NewBalance}`;
			data = await getMarket(id);
		} catch (e) {
			tradeError = String(e);
		} finally {
			trading = false;
		}
	}
</script>

<svelte:head>
	<title>{data?.market.title ?? 'Market'} — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-8">
	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading market…</div>
	{:else if error}
		<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mb-4">{error}</div>
	{:else if data}
		{@const market = data.market}

		<!-- Header -->
		<div class="mb-6">
			<div class="flex items-center gap-2 mb-2">
				<a href="/markets" class="text-surface-400 hover:text-primary-400 text-sm no-underline">← Markets</a>
				<span class="badge preset-tonal-surface text-xs">{market.category.replace('_', ' ')}</span>
				{#if market.status === 'resolved'}
					<span class="badge preset-filled-success-500 text-xs">
						Resolved: {market.resolution?.toUpperCase()}
					</span>
				{:else if market.status === 'active'}
					<span class="badge preset-filled-primary-500 text-xs">Active</span>
				{/if}
			</div>
			<h1 class="text-3xl font-bold text-surface-100">{market.title}</h1>
			<p class="text-surface-400 text-sm mt-1">
				by {market.creator_name} ·
				{market.status === 'resolved' ? 'resolved' : 'closes'}
				{new Date(market.resolution_deadline).toLocaleDateString()}
			</p>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
			<!-- Market info -->
			<div class="md:col-span-2 space-y-4">
				{#if market.description}
					<div class="card preset-tonal-surface p-4 rounded-lg">
						<h3 class="text-surface-300 text-sm font-semibold mb-2 uppercase tracking-wide">Description</h3>
						<p class="text-surface-200 text-sm">{market.description}</p>
					</div>
				{/if}

				{#if market.resolution_criteria}
					<div class="card preset-tonal-surface p-4 rounded-lg">
						<h3 class="text-surface-300 text-sm font-semibold mb-2 uppercase tracking-wide">
							Resolution Criteria
						</h3>
						<p class="text-surface-200 text-sm">{market.resolution_criteria}</p>
					</div>
				{/if}
			</div>

			<!-- Trade widget + prices -->
			<div class="space-y-4">
				<!-- Current prices -->
				<div class="card preset-tonal-surface p-4 rounded-lg">
					<h3 class="text-surface-300 text-sm font-semibold mb-3 uppercase tracking-wide">Current Odds</h3>
					<div class="grid grid-cols-2 gap-3">
						<div class="text-center p-3 rounded bg-surface-800">
							<div class="text-2xl font-bold text-success-400">{data.yes_price}¢</div>
							<div class="text-surface-400 text-xs mt-1">YES</div>
						</div>
						<div class="text-center p-3 rounded bg-surface-800">
							<div class="text-2xl font-bold text-error-400">{data.no_price}¢</div>
							<div class="text-surface-400 text-xs mt-1">NO</div>
						</div>
					</div>
				</div>

				<!-- Buy widget -->
				{#if market.status === 'active'}
					{#if $isLoggedIn}
						<div class="card preset-tonal-surface p-4 rounded-lg">
							<h3 class="text-surface-300 text-sm font-semibold mb-3 uppercase tracking-wide">Buy Shares</h3>

							<div class="flex gap-2 mb-3">
								<button
									onclick={() => (tradeSide = 'yes')}
									class="flex-1 btn btn-sm {tradeSide === 'yes' ? 'preset-filled-success-500' : 'preset-outlined'}"
								>
									YES
								</button>
								<button
									onclick={() => (tradeSide = 'no')}
									class="flex-1 btn btn-sm {tradeSide === 'no' ? 'preset-filled-error-500' : 'preset-outlined'}"
								>
									NO
								</button>
							</div>

							<label class="block mb-3">
								<span class="text-surface-400 text-xs">Shares</span>
								<input
									type="number"
									bind:value={tradeShares}
									min="1"
									max="1000"
									step="1"
									class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-1.5 text-surface-100 text-sm"
								/>
							</label>

							<button
								onclick={doTrade}
								disabled={trading || tradeShares <= 0}
								class="btn preset-filled-primary-500 w-full"
							>
								{trading ? 'Buying…' : `Buy ${tradeShares} ${tradeSide.toUpperCase()}`}
							</button>

							{#if tradeSuccess}
								<div class="p-3 bg-success-500/20 border border-success-500 rounded-lg text-success-300 mt-3 text-xs">
									{tradeSuccess}
								</div>
							{/if}
							{#if tradeError}
								<div class="p-3 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mt-3 text-xs">
									{tradeError}
								</div>
							{/if}

							{#if $currentUser}
								<p class="text-surface-500 text-xs mt-2 text-center">
								Balance: {$currentUser.balance.toLocaleString()} bUEC
								</p>
							{/if}
						</div>
					{:else}
						<div class="card preset-tonal-surface p-4 rounded-lg text-center">
							<p class="text-surface-400 text-sm mb-3">Login to trade</p>
							<a href="/auth/login" class="btn btn-sm preset-filled-primary-500">Login with SCID</a>
						</div>
					{/if}
				{/if}
			</div>
		</div>
	{/if}
</div>
