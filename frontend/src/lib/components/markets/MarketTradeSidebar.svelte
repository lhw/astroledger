<script lang="ts">
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { Market, MarketOutcome, UserOutcomePosition } from '$lib/types';

	let {
		market,
		myPositions,
		totalVolume,
		traderCount,
		tradeCount,
		tradeAction = $bindable(),
		selectedOutcomeId = $bindable(),
		selectedOutcome,
		tradeShares = $bindable(),
		sellShares = $bindable(),
		budgetMode = $bindable(),
		budgetAmount = $bindable(),
		trading,
		tradeSuccess,
		tradeError,
		estimatedCost,
		estimatedRevenue,
		actualBudgetCost,
		maxShares,
		maxSellShares,
		floorBudgetShares,
		canAfford,
		canAffordBudget,
		onTrade
	}: {
		market: Market;
		myPositions: UserOutcomePosition[];
		totalVolume: number;
		traderCount: number;
		tradeCount: number;
		tradeAction: 'buy' | 'sell';
		selectedOutcomeId: number | null;
		selectedOutcome: MarketOutcome | null;
		tradeShares: number;
		sellShares: number;
		budgetMode: boolean;
		budgetAmount: number;
		trading: boolean;
		tradeSuccess: string;
		tradeError: string;
		estimatedCost: number;
		estimatedRevenue: number;
		actualBudgetCost: number;
		maxShares: number;
		maxSellShares: number;
		floorBudgetShares: number;
		canAfford: boolean;
		canAffordBudget: boolean;
		onTrade: () => Promise<void>;
	} = $props();

	let holdingShares = $derived((myPositions ?? []).find((position) => position.outcome_id === selectedOutcome?.id)?.shares ?? 0);
</script>

<div class="space-y-4">
	<div class="sc-card p-5">
		<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-4">Current Odds</h3>
		<div class="flex flex-wrap gap-2">
			{#each (market.outcomes ?? []) as outcome}
				<div class="text-center p-3 rounded bg-primary-50 border border-primary-100 flex-1 min-w-[80px]">
					<div class="text-2xl font-bold text-primary-700">{outcome.price}%</div>
					<div class="text-primary-600 text-xs mt-1 uppercase tracking-wider font-semibold truncate">{outcome.label}</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="sc-card p-5">
		<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Market Stats</h3>
		<dl class="grid grid-cols-3 gap-2 text-center">
			<div>
				<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Volume</dt>
				<dd class="text-base font-bold text-surface-800 mt-0.5">{totalVolume.toLocaleString()}</dd>
				<dd class="text-[10px] text-surface-400">bUEC</dd>
			</div>
			<div>
				<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Traders</dt>
				<dd class="text-base font-bold text-surface-800 mt-0.5">{traderCount}</dd>
			</div>
			<div>
				<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Trades</dt>
				<dd class="text-base font-bold text-surface-800 mt-0.5">{tradeCount}</dd>
			</div>
		</dl>
		{#if market.resolution_type !== 'binary'}
			<div class="mt-3 pt-3 border-t border-surface-100">
				{#if market.resolution_type === 'date'}
					{@const dateThreshold = market.resolution_threshold ? new Date(market.resolution_threshold).toLocaleDateString() : '(unset)'}
					{@const resolvedLabel = (market.outcomes ?? []).find((outcome) => outcome.id === market.resolved_outcome_id)?.label ?? '—'}
					{#if market.status === 'resolved'}
						<p class="text-xs text-surface-500">
							<span class="font-semibold text-surface-700">Date prediction</span> — resolved
							<span class="font-semibold text-surface-700"> {resolvedLabel}</span> against threshold
							<span class="font-semibold text-surface-700"> {dateThreshold}</span>
						</p>
					{:else}
						<p class="text-xs text-surface-500">
							<span class="font-semibold text-surface-700">Date prediction</span> — resolves YES if the event occurs before
							<span class="font-semibold text-surface-700"> {dateThreshold}</span>
						</p>
					{/if}
				{:else if market.resolution_type === 'numeric'}
					{@const numericThreshold = market.resolution_threshold ?? '?'}
					{@const resolvedLabel = (market.outcomes ?? []).find((outcome) => outcome.id === market.resolved_outcome_id)?.label ?? '—'}
					{#if market.status === 'resolved'}
						<p class="text-xs text-surface-500">
							<span class="font-semibold text-surface-700">Numeric prediction</span> — resolved
							<span class="font-semibold text-surface-700"> {resolvedLabel}</span> against threshold
							<span class="font-semibold text-surface-700"> ${numericThreshold}</span>
						</p>
					{:else}
						<p class="text-xs text-surface-500">
							<span class="font-semibold text-surface-700">Numeric prediction</span> — resolves YES if the value reaches
							<span class="font-semibold text-surface-700"> ${numericThreshold}</span>
						</p>
					{/if}
				{/if}
			</div>
		{/if}
	</div>

	{#if market.status === 'resolved'}
		{@const winOutcome = (market.outcomes ?? []).find((outcome) => outcome.id === market.resolved_outcome_id)}
		<div class="sc-card p-5 border-2 border-primary-400 bg-primary-50/30">
			<h3 class="text-xs font-bold text-primary-600 uppercase tracking-[0.12em] mb-2">Resolved</h3>
			<div class="text-3xl font-black text-primary-700 mb-1">{winOutcome?.label ?? '—'}</div>
			<p class="text-xs text-surface-500">
				{#if market.resolver_name}by <span class="font-medium text-surface-700">{market.resolver_name}</span>{/if}
				{#if market.resolved_at} · {new Date(market.resolved_at).toLocaleDateString()}{/if}
			</p>
			{#if market.resolution_evidence}
				<a href={market.resolution_evidence} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 hover:underline mt-2 block truncate">
					📋 Evidence: {market.resolution_evidence}
				</a>
			{/if}
		</div>
	{/if}

	{#if market.status === 'active' || market.status === 'resolution_requested'}
		{#if $isLoggedIn}
			<div class="sc-card p-5">
				<div class="flex gap-1 mb-4 p-1 bg-surface-100 rounded-lg">
					<button
						onclick={() => (tradeAction = 'buy')}
						class="flex-1 btn btn-sm text-xs uppercase tracking-wider font-bold transition-colors {tradeAction === 'buy' ? 'bg-white shadow text-primary-700 border border-primary-200' : 'text-surface-500 hover:text-surface-700'}"
					>
						Buy
					</button>
					<button
						onclick={() => (tradeAction = 'sell')}
						class="flex-1 btn btn-sm text-xs uppercase tracking-wider font-bold transition-colors {tradeAction === 'sell' ? 'bg-white shadow text-surface-700 border border-surface-300' : 'text-surface-500 hover:text-surface-700'}"
					>
						Sell
					</button>
				</div>

				<div class="mb-4">
					<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold block mb-2">Outcome</span>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
						{#each (market.outcomes ?? []) as outcome}
							<button
								onclick={() => (selectedOutcomeId = outcome.id)}
								class="w-full min-w-0 btn btn-sm {selectedOutcome?.id === outcome.id ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400 hover:text-primary-700'} transition-colors text-[11px] sm:text-xs uppercase tracking-wider font-bold leading-tight py-2"
							>
								<span class="block truncate">{outcome.label}</span>
								<span class="block opacity-70 normal-case font-normal">{outcome.price}%</span>
							</button>
						{/each}
					</div>
				</div>

				{#if tradeAction === 'buy'}
					<div class="flex gap-0.5 mb-3 p-0.5 bg-surface-100 rounded text-[11px]">
						<button
							onclick={() => (budgetMode = false)}
							class="flex-1 py-1 rounded font-semibold uppercase tracking-wider transition-colors {!budgetMode ? 'bg-white shadow text-surface-700' : 'text-surface-400 hover:text-surface-600'}"
						>By Shares</button>
						<button
							onclick={() => (budgetMode = true)}
							class="flex-1 py-1 rounded font-semibold uppercase tracking-wider transition-colors {budgetMode ? 'bg-white shadow text-surface-700' : 'text-surface-400 hover:text-surface-600'}"
						>By Budget</button>
					</div>

					{#if !budgetMode}
						<label class="block mb-1" for="trade-shares">
							<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Shares to Buy</span>
						</label>
						<div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-1">
							<input id="trade-shares" type="number" bind:value={tradeShares} min="1" max={maxShares} step="1" class="sc-input text-sm flex-1" />
							{#if $currentUser && maxShares > 0}
								<button onclick={() => (tradeShares = maxShares)} class="text-xs text-primary-600 hover:underline whitespace-nowrap self-start sm:self-auto" title="Max shares you can afford">
									Max ({maxShares})
								</button>
							{/if}
						</div>
						{#if $currentUser}
							<div class="mb-4 text-xs text-surface-500">
								Cost: <span class={canAfford ? 'text-surface-700 font-semibold' : 'text-red-600 font-semibold'}>{estimatedCost.toLocaleString()} bUEC</span>
								{#if tradeShares > maxShares}
									<span class="text-red-500"> — max is {maxShares} shares with your balance</span>
								{:else if !canAfford}
									<span class="text-red-500"> — insufficient balance</span>
								{/if}
							</div>
						{/if}
						<button onclick={onTrade} disabled={trading || tradeShares <= 0 || !canAfford} class="btn preset-filled-primary-500 w-full text-[11px] sm:text-xs uppercase tracking-wider whitespace-normal break-words leading-tight py-2 h-auto disabled:opacity-50">
							{trading ? 'Buying…' : `Buy ${tradeShares} ${selectedOutcome?.label ?? ''} for ${estimatedCost.toLocaleString()} bUEC`}
						</button>
					{:else}
						<label class="block mb-1" for="trade-budget">
							<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Budget (bUEC)</span>
						</label>
						<div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-1">
							<input id="trade-budget" type="number" bind:value={budgetAmount} min="1" max={$currentUser?.balance ?? 99999} step="10" class="sc-input text-sm flex-1" />
							{#if $currentUser}
								<button onclick={() => (budgetAmount = $currentUser.balance)} class="text-xs text-primary-600 hover:underline whitespace-nowrap self-start sm:self-auto" title="Use entire balance">All in</button>
							{/if}
						</div>
						<div class="mb-4 text-xs text-surface-500 space-y-0.5">
							{#if floorBudgetShares > 0}
								<div>You'll buy: <span class="text-surface-700 font-semibold">{floorBudgetShares} {selectedOutcome?.label ?? ''} share{floorBudgetShares !== 1 ? 's' : ''}</span></div>
								<div>Actual cost: <span class={canAffordBudget ? 'text-surface-700 font-semibold' : 'text-red-600 font-semibold'}>{actualBudgetCost.toLocaleString()} bUEC</span>
									{#if budgetAmount > actualBudgetCost}
										<span class="text-surface-400"> ({(budgetAmount - actualBudgetCost).toLocaleString()} bUEC unused)</span>
									{/if}
								</div>
							{:else}
								<div class="text-red-500">Budget too low — can't buy any shares at this price.</div>
							{/if}
							{#if !canAffordBudget && floorBudgetShares > 0}
								<div class="text-red-500">Insufficient balance.</div>
							{/if}
						</div>
						<button onclick={onTrade} disabled={trading || floorBudgetShares <= 0 || !canAffordBudget} class="btn preset-filled-primary-500 w-full text-[11px] sm:text-xs uppercase tracking-wider whitespace-normal break-words leading-tight py-2 h-auto disabled:opacity-50">
							{trading ? 'Buying…' : `Buy ${floorBudgetShares} ${selectedOutcome?.label ?? ''} for ${actualBudgetCost.toLocaleString()} bUEC`}
						</button>
					{/if}
				{:else}
					<div class="mb-3 p-2 bg-surface-50 rounded text-xs text-surface-600">
						You hold: <span class="font-bold text-surface-800">{holdingShares}</span> {selectedOutcome?.label ?? ''} shares
					</div>
					<label class="block mb-1" for="sell-shares">
						<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Shares to Sell</span>
					</label>
					<div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-1">
						<input id="sell-shares" type="number" bind:value={sellShares} min="1" max={maxSellShares} step="1" class="sc-input text-sm flex-1" />
						{#if maxSellShares > 0}
							<button onclick={() => (sellShares = maxSellShares)} class="text-xs text-primary-600 hover:underline whitespace-nowrap self-start sm:self-auto" title="Sell all">
								All ({maxSellShares})
							</button>
						{/if}
					</div>
					<div class="mb-4 text-xs text-surface-500">
						Est. revenue: <span class="text-green-700 font-semibold">{estimatedRevenue.toLocaleString()} bUEC</span>
					</div>
					<button onclick={onTrade} disabled={trading || sellShares <= 0 || sellShares > maxSellShares} class="btn w-full border border-surface-400 text-surface-700 hover:bg-surface-100 text-[11px] sm:text-xs uppercase tracking-wider whitespace-normal break-words leading-tight py-2 h-auto disabled:opacity-50">
						{trading ? 'Selling…' : `Sell ${sellShares} ${selectedOutcome?.label ?? ''}`}
					</button>
				{/if}

				{#if tradeSuccess}
					<div class="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 mt-3 text-xs">{tradeSuccess}</div>
				{/if}
				{#if tradeError}
					<div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 mt-3 text-xs">{tradeError}</div>
				{/if}

				{#if $currentUser}
					<p class="text-surface-400 text-xs mt-3 text-center">Balance: {$currentUser.balance.toLocaleString()} bUEC</p>
				{/if}
			</div>
		{:else}
			<div class="sc-card p-5 text-center">
				<p class="text-surface-600 text-sm mb-3">Login to trade</p>
				<a href="/auth/login" class="btn btn-sm preset-filled-primary-500 text-xs uppercase tracking-wider">Login with SCID</a>
			</div>
		{/if}
	{/if}
</div>