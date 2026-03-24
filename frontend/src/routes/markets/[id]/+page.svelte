<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getMarket, executeTrade } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { MarketWithPrice } from '$lib/types';

	const id = Number($page.params.id);

	let data: MarketWithPrice | null = null;
	let loading = true;
	let error = '';

	// Trade widget state
	let tradeSide: 'yes' | 'no' = 'yes';
	let tradeShares = 1;
	let trading = false;
	let tradeError = '';
	let tradeSuccess = '';

	onMount(async () => {
		try {
			data = await getMarket(id);
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});

	async function executeTr() {
		if (!$isLoggedIn || !data) return;
		trading = true;
		tradeError = '';
		tradeSuccess = '';
		try {
			const result = await executeTrade(id, tradeSide, 'buy', tradeShares);
			tradeSuccess = `Bought ${tradeShares} ${tradeSide.toUpperCase()} shares for ${result.Cost} ScollyBucks™. New balance: ${result.NewBalance}`;
			// Refresh market data
			data = await getMarket(id);
		} catch (e) {
			tradeError = String(e);
		} finally {
			trading = false;
		}
	}
</script>

<div class="container mx-auto px-4 max-w-4xl py-8">
	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading market…</div>
	{:else if error}
		<div class="alert variant-filled-error mb-4">{error}</div>
	{:else if data}
		{@const market = data.market}

		<!-- Header -->
		<div class="mb-6">
			<div class="flex items-center gap-2 mb-2">
				<a href="/markets" class="text-surface-400 hover:text-primary-400 text-sm">← Markets</a>
				<span class="badge variant-filled-surface text-xs">{market.category.replace('_', ' ')}</span>
				{#if market.status === 'resolved'}
					<span class="badge variant-filled-success text-xs">
						Resolved: {market.resolution?.toUpperCase()}
					</span>
				{:else if market.status === 'active'}
					<span class="badge variant-filled-primary text-xs">Active</span>
				{/if}
			</div>
			<h1 class="h2 text-surface-100">{market.title}</h1>
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
					<div class="card variant-glass-surface p-4 rounded-lg">
						<h3 class="text-surface-300 text-sm font-semibold mb-2 uppercase tracking-wide">Description</h3>
						<p class="text-surface-200 text-sm">{market.description}</p>
					</div>
				{/if}

				{#if market.resolution_criteria}
					<div class="card variant-glass-surface p-4 rounded-lg">
						<h3 class="text-surface-300 text-sm font-semibold mb-2 uppercase tracking-wide">Resolution Criteria</h3>
						<p class="text-surface-200 text-sm">{market.resolution_criteria}</p>
					</div>
				{/if}
			</div>

			<!-- Trade widget + prices -->
			<div class="space-y-4">
				<!-- Current prices -->
				<div class="card variant-glass-surface p-4 rounded-lg">
					<h3 class="text-surface-300 text-sm font-semibold mb-3 uppercase tracking-wide">Current Odds</h3>
					<div class="grid grid-cols-2 gap-3">
						<div class="text-center p-3 rounded bg-surface-700">
							<div class="text-2xl font-bold text-success-400">{data.yes_price}¢</div>
							<div class="text-surface-400 text-xs mt-1">YES</div>
						</div>
						<div class="text-center p-3 rounded bg-surface-700">
							<div class="text-2xl font-bold text-error-400">{data.no_price}¢</div>
							<div class="text-surface-400 text-xs mt-1">NO</div>
						</div>
					</div>
				</div>

				<!-- Buy widget (only for active markets) -->
				{#if market.status === 'active'}
					{#if $isLoggedIn}
						<div class="card variant-glass-surface p-4 rounded-lg">
							<h3 class="text-surface-300 text-sm font-semibold mb-3 uppercase tracking-wide">Buy Shares</h3>

							<div class="flex gap-2 mb-3">
								<button
									on:click={() => tradeSide = 'yes'}
									class="flex-1 btn btn-sm {tradeSide === 'yes' ? 'variant-filled-success' : 'variant-ghost-surface'}"
								>
									YES
								</button>
								<button
									on:click={() => tradeSide = 'no'}
									class="flex-1 btn btn-sm {tradeSide === 'no' ? 'variant-filled-error' : 'variant-ghost-surface'}"
								>
									NO
								</button>
							</div>

							<label class="label mb-3">
								<span class="text-surface-400 text-xs">Shares</span>
								<input
									type="number"
									bind:value={tradeShares}
									min="1"
									max="1000"
									step="1"
									class="input text-sm"
								/>
							</label>

							<button
								on:click={executeTr}
								disabled={trading || tradeShares <= 0}
								class="btn variant-filled-primary w-full"
							>
								{trading ? 'Buying…' : `Buy ${tradeShares} ${tradeSide.toUpperCase()}`}
							</button>

							{#if tradeSuccess}
								<div class="alert variant-filled-success mt-3 text-xs">{tradeSuccess}</div>
							{/if}
							{#if tradeError}
								<div class="alert variant-filled-error mt-3 text-xs">{tradeError}</div>
							{/if}

							{#if $currentUser}
								<p class="text-surface-500 text-xs mt-2 text-center">
									Balance: {$currentUser.balance.toLocaleString()} ScollyBucks™
								</p>
							{/if}
						</div>
					{:else}
						<div class="card variant-glass-surface p-4 rounded-lg text-center">
							<p class="text-surface-400 text-sm mb-3">Login to trade</p>
							<a href="/auth/login" class="btn btn-sm variant-filled-primary">Login with SCID</a>
						</div>
					{/if}
				{/if}
			</div>
		</div>
	{/if}
</div>
