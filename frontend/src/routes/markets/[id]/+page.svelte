<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getMarket, executeTrade, requestResolution, getMarketPriceHistory, submitReport, getMe } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import { buyCost, sellRevenue, maxAffordable } from '$lib/amm';
	import { renderMarkdown } from '$lib/markdown';
	import type { MarketWithPrice, PricePoint } from '$lib/types';

	let { data } = $props<{ data: { market: MarketWithPrice | null; history: PricePoint[] } }>();

	const id = $derived(Number($page.params.id));

	// untrack() signals that we intentionally want a one-time snapshot, not a reactive dependency.
	const { market: initialMarket, history: initialHistory } = untrack(() => data);
	let data_ = $state<MarketWithPrice | null>(initialMarket ?? null);
	let loading = $state(initialMarket == null);
	let error = $state('');
	let priceHistory = $state<PricePoint[]>(initialHistory ?? []);

	// Trading
	let tradeAction = $state<'buy' | 'sell'>('buy');
	let tradeSide = $state<'yes' | 'no'>('yes');
	let tradeShares = $state(1);
	let sellShares = $state(1);
	let trading = $state(false);
	let tradeError = $state('');
	let tradeSuccess = $state('');
	// Budget mode — enter bUEC amount instead of share count
	let budgetMode = $state(false);
	let budgetAmount = $state(100);

	// Resolution request
	let requestingResolution = $state(false);
	let resolutionRequestMsg = $state('');
	let resolutionLink = $state('');
	let resolutionNote = $state('');

	// Report
	let showReportForm = $state(false);
	let reportReason = $state('');
	let submittingReport = $state(false);
	let reportMsg = $state('');

	// Reactive cost/revenue estimates
	let estimatedCost = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return buyCost(m.liquidity_param, m.yes_shares, m.no_shares, Math.max(1, tradeShares), tradeSide === 'yes');
	});

	let budgetShares = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return maxAffordable(m.liquidity_param, m.yes_shares, m.no_shares, Math.max(0, budgetAmount), tradeSide === 'yes');
	});

	let estimatedRevenue = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return sellRevenue(m.liquidity_param, m.yes_shares, m.no_shares, Math.max(1, sellShares), tradeSide === 'yes');
	});

	let maxShares = $derived.by(() => {
		if (!data_ || !$currentUser) return 1000;
		const m = data_.market;
		return maxAffordable(m.liquidity_param, m.yes_shares, m.no_shares, $currentUser.balance, tradeSide === 'yes');
	});

	let maxSellShares = $derived.by(() => {
		if (!data_?.my_position) return 0;
		return tradeSide === 'yes'
			? Math.floor(data_.my_position.yes_shares)
			: Math.floor(data_.my_position.no_shares);
	});

	let canAfford = $derived($currentUser ? estimatedCost <= $currentUser.balance : false);
	let canAffordBudget = $derived($currentUser ? budgetAmount <= $currentUser.balance : false);

	onMount(async () => {
		if (initialMarket) {
			// If the market is already resolved and the user is logged in, refresh
			// their balance — payout may have occurred since they last loaded.
			if (initialMarket.market.status === 'resolved' && $isLoggedIn) {
				getMe().then((u) => { if (u) currentUser.set(u); });
			}
			return;
		}
		try {
			[data_, priceHistory] = await Promise.all([
				getMarket(id),
				getMarketPriceHistory(id).catch(() => [] as PricePoint[])
			]);
			// Refresh balance if visiting a resolved market.
			if (data_?.market.status === 'resolved' && $isLoggedIn) {
				getMe().then((u) => { if (u) currentUser.set(u); });
			}
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});

	async function doTrade() {
		if (!$isLoggedIn || !data_) return;
		trading = true;
		tradeError = '';
		tradeSuccess = '';
		try {
			if (tradeAction === 'buy') {
				// Budget mode: use the computed share count from the LMSR inversion.
				const sharesToBuy = budgetMode ? budgetShares : tradeShares;
				if (sharesToBuy <= 0) {
					tradeError = 'Budget too low — cannot purchase any shares at this price.';
					return;
				}
				const result = await executeTrade(id, tradeSide, 'buy', sharesToBuy);
				currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
				tradeSuccess = `Bought ${sharesToBuy.toFixed(2)} ${tradeSide.toUpperCase()} share${sharesToBuy !== 1 ? 's' : ''} for ${result.Cost.toLocaleString()} bUEC.`;
			} else {
				const result = await executeTrade(id, tradeSide, 'sell', sellShares);
				currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
				const received = Math.abs(result.Cost);
				tradeSuccess = `Sold ${sellShares} ${tradeSide.toUpperCase()} share${sellShares !== 1 ? 's' : ''} for ${received.toLocaleString()} bUEC.`;
			}
			// Refresh market data and price history.
			[data_, priceHistory] = await Promise.all([
				getMarket(id),
				getMarketPriceHistory(id).catch(() => [] as PricePoint[])
			]);
		} catch (e) {
			tradeError = String(e);
		} finally {
			trading = false;
		}
	}

	async function doRequestResolution() {
		requestingResolution = true;
		resolutionRequestMsg = '';
		try {
			await requestResolution(id, resolutionLink, resolutionNote);
			resolutionRequestMsg = 'Resolution request sent to mods.';
			data_ = await getMarket(id);
		} catch (e) {
			resolutionRequestMsg = e instanceof Error ? e.message : String(e);
		} finally {
			requestingResolution = false;
		}
	}

	async function doSubmitReport() {
		if (!reportReason.trim()) return;
		submittingReport = true;
		reportMsg = '';
		try {
			await submitReport(id, reportReason.trim());
			reportMsg = 'Report submitted. A moderator will review it.';
			reportReason = '';
			showReportForm = false;
		} catch (e) {
			reportMsg = e instanceof Error ? e.message : String(e);
		} finally {
			submittingReport = false;
		}
	}
</script>

<svelte:head>
	<title>{data_?.market.title ?? 'Market'} — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-10">
	{#if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading market…</div>
	{:else if error}
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-4">{error}</div>
	{:else if data_}
		{@const market = data_.market}
		{@const myPos = data_.my_position}
		{@const hasYes = (myPos?.yes_shares ?? 0) > 0}
		{@const hasNo = (myPos?.no_shares ?? 0) > 0}
		{@const hasPosition = hasYes || hasNo}

		<!-- Header -->
		<div class="mb-8">
			<div class="flex items-center gap-2 mb-3">
				<a href="/markets" class="text-surface-500 hover:text-primary-600 text-xs uppercase tracking-wider no-underline transition-colors">← Markets</a>
				<span class="text-surface-300">/</span>
				<span class="sc-tag">{market.category.replace('_', ' ')}</span>
				{#if market.status === 'resolved'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-green-100 text-green-700 border border-green-200">
						Resolved {market.resolution?.toUpperCase()}
					</span>
				{:else if market.status === 'resolution_requested'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Awaiting Resolution</span>
				{:else if market.status === 'active'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-primary-50 text-primary-700 border border-primary-200">Active</span>
				{/if}
			</div>
			<h1 class="text-2xl font-bold text-surface-900 leading-snug">{market.title}</h1>
			<p class="text-surface-500 text-sm mt-2">
				Submitted by {market.creator_name} ·
				{market.status === 'resolved' ? 'resolved' : 'closes'}
				{new Date(market.resolution_deadline).toLocaleDateString()}
			</p>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-5">
			<!-- Market info -->
			<div class="md:col-span-2 space-y-4">
				{#if market.description}
					<div class="sc-card p-5">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Description</h3>
						<div class="text-surface-700 text-sm leading-relaxed prose prose-sm max-w-none">{@html renderMarkdown(market.description)}</div>
					</div>
				{/if}

				{#if market.resolution_criteria}
					<div class="sc-card p-5">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Resolution Criteria</h3>
						<div class="text-surface-700 text-sm leading-relaxed prose prose-sm max-w-none">{@html renderMarkdown(market.resolution_criteria)}</div>
					</div>
				{/if}

				<!-- Price history chart -->
				<div class="sc-card p-5">
					<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Price History</h3>
					{#if priceHistory.length > 1}
						{@const W = 600}
						{@const H = 100}
						{@const n = priceHistory.length}
						{@const pts = priceHistory.map((p, i) =>
							`${(i / (n - 1)) * W},${H - p.price_at_trade * H}`
						).join(' ')}
						<div class="flex items-end gap-3 mb-2">
							<span class="text-green-700 text-xs font-semibold">
								YES {Math.round((priceHistory.at(-1)?.price_at_trade ?? 0) * 100)}¢
							</span>
							<span class="text-surface-400 text-xs">last {priceHistory.length} trades</span>
						</div>
						<svg viewBox="0 0 {W} {H}" class="w-full rounded" style="height:90px;background:#fafaf8">
							<!-- 50% reference line -->
							<line x1="0" y1={H / 2} x2={W} y2={H / 2} stroke="#e5e7eb" stroke-width="1" stroke-dasharray="4 4"/>
							<polyline points={pts} fill="none" stroke="#D4A843" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>
							<!-- Latest dot -->
							<circle cx={W} cy={H - (priceHistory.at(-1)?.price_at_trade ?? 0.5) * H} r="4" fill="#D4A843"/>
						</svg>
						<div class="flex justify-between mt-1">
							<span class="text-surface-400 text-[10px]">{new Date(priceHistory[0].created_at).toLocaleDateString()}</span>
							<span class="text-surface-400 text-[10px]">{new Date(priceHistory.at(-1)!.created_at).toLocaleDateString()}</span>
						</div>
					{:else}
						<p class="text-surface-400 text-xs text-center py-4">No trades yet — be the first!</p>
					{/if}
				</div>

				<!-- My current position -->
				{#if hasPosition}
					<div class="sc-card p-5 border-l-4 border-l-primary-400 bg-amber-50/30">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Your Position</h3>
						<div class="flex gap-4">
							{#if hasYes}
								<div class="flex-1 text-center p-3 rounded bg-green-50 border border-green-100">
									<div class="text-2xl font-bold text-green-700">{myPos!.yes_shares}</div>
									<div class="text-green-600 text-xs mt-0.5 uppercase tracking-wider font-semibold">YES shares</div>
								</div>
							{/if}
							{#if hasNo}
								<div class="flex-1 text-center p-3 rounded bg-red-50 border border-red-100">
									<div class="text-2xl font-bold text-red-600">{myPos!.no_shares}</div>
									<div class="text-red-500 text-xs mt-0.5 uppercase tracking-wider font-semibold">NO shares</div>
								</div>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Request resolution — available to anyone with shares -->
				{#if hasPosition && market.status === 'active'}
					<div class="sc-card p-4 border border-amber-200 bg-amber-50/30">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-2">Request Resolution</h3>
						<p class="text-surface-600 text-xs mb-3">Think the resolution criteria has been met? Ask the mod team to review this market.</p>
						<label class="block mb-2" for="res-link">
							<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Evidence Link <span class="normal-case font-normal text-surface-400">(optional)</span></span>
							<input
								id="res-link"
								type="url"
								bind:value={resolutionLink}
								placeholder="https://…"
								class="sc-input mt-1 text-sm"
							/>
						</label>
						<label class="block mb-3" for="res-note">
							<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Note to mods <span class="normal-case font-normal text-surface-400">(optional)</span></span>
							<textarea
								id="res-note"
								bind:value={resolutionNote}
								rows="2"
								maxlength="500"
								placeholder="Brief explanation of why this should resolve…"
								class="sc-input mt-1 text-sm resize-none"
							></textarea>
						</label>
						<button
							onclick={doRequestResolution}
							disabled={requestingResolution}
							class="btn btn-sm border border-amber-500 text-amber-700 hover:bg-amber-100 text-xs uppercase tracking-wider transition-colors"
						>
							{requestingResolution ? 'Requesting…' : 'Request Resolution'}
						</button>
						{#if resolutionRequestMsg}
							<p class="text-xs mt-2 {resolutionRequestMsg.startsWith('Resolution request') ? 'text-green-700' : 'text-red-600'}">{resolutionRequestMsg}</p>
						{/if}
					</div>
				{:else if hasPosition && market.status === 'resolution_requested'}
					<div class="sc-card p-4 border border-amber-200 bg-amber-50/30">
						<p class="text-amber-700 text-xs font-semibold">Resolution has been requested — mods will review shortly.</p>
					</div>
				{/if}

				<!-- Report market -->
				{#if $isLoggedIn && (market.status === 'active' || market.status === 'resolution_requested')}
					<div class="mt-2">
						{#if !showReportForm}
							<button
								onclick={() => { showReportForm = true; }}
								class="text-surface-400 hover:text-red-500 text-xs transition-colors"
							>
								⚑ Report this market
							</button>
						{:else}
							<div class="sc-card p-4 border border-red-100">
								<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-2">Report Market</h3>
								<label class="block mb-2" for="report-reason">
									<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Reason</span>
									<textarea
										id="report-reason"
										bind:value={reportReason}
										rows="2"
										maxlength="500"
										placeholder="Why should mods review this market?"
										class="sc-input mt-1 text-sm resize-none"
									></textarea>
								</label>
								<div class="flex gap-2">
									<button
										onclick={doSubmitReport}
										disabled={submittingReport || reportReason.trim().length < 5}
										class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider disabled:opacity-50"
									>
										{submittingReport ? 'Submitting…' : 'Submit Report'}
									</button>
									<button
										onclick={() => { showReportForm = false; reportReason = ''; reportMsg = ''; }}
										class="btn btn-sm border border-surface-300 text-surface-500 text-xs"
									>
										Cancel
									</button>
								</div>
								{#if reportMsg}
									<p class="text-xs mt-2 {reportMsg.startsWith('Report submitted') ? 'text-green-700' : 'text-red-600'}">{reportMsg}</p>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Trade widget + prices -->
			<div class="space-y-4">
				<!-- Current prices -->
				<div class="sc-card p-5">
					<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-4">Current Odds</h3>
					<div class="grid grid-cols-2 gap-3">
						<div class="text-center p-3 rounded bg-green-50 border border-green-100">
							<div class="text-2xl font-bold text-green-700">{data_.yes_price}¢</div>
							<div class="text-green-600 text-xs mt-1 uppercase tracking-wider font-semibold">YES</div>
						</div>
						<div class="text-center p-3 rounded bg-red-50 border border-red-100">
							<div class="text-2xl font-bold text-red-600">{data_.no_price}¢</div>
							<div class="text-red-500 text-xs mt-1 uppercase tracking-wider font-semibold">NO</div>
						</div>
					</div>
				</div>

				<!-- Market stats -->
				<div class="sc-card p-5">
					<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Market Stats</h3>
					<dl class="grid grid-cols-3 gap-2 text-center">
						<div>
							<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Volume</dt>
							<dd class="text-base font-bold text-surface-800 mt-0.5">{data_.total_volume.toLocaleString()}</dd>
							<dd class="text-[10px] text-surface-400">bUEC</dd>
						</div>
						<div>
							<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Traders</dt>
							<dd class="text-base font-bold text-surface-800 mt-0.5">{data_.trader_count}</dd>
						</div>
						<div>
							<dt class="text-[10px] text-surface-400 uppercase tracking-wider font-semibold">Trades</dt>
							<dd class="text-base font-bold text-surface-800 mt-0.5">{data_.trade_count}</dd>
						</div>
					</dl>
					{#if market.resolution_type !== 'binary'}
						<div class="mt-3 pt-3 border-t border-surface-100">
							{#if market.resolution_type === 'date'}
								<p class="text-xs text-surface-500">
									<span class="font-semibold text-surface-700">Date prediction</span> — resolves YES if the event occurs before
									<span class="font-semibold text-surface-700">{market.resolution_threshold ? new Date(market.resolution_threshold).toLocaleDateString() : '(unset)'}</span>
								</p>
							{:else if market.resolution_type === 'numeric'}
								<p class="text-xs text-surface-500">
									<span class="font-semibold text-surface-700">Numeric prediction</span> — resolves YES if the value reaches
									<span class="font-semibold text-surface-700">${market.resolution_threshold ?? '?'}</span>
								</p>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Resolution result (resolved markets) -->
				{#if market.status === 'resolved'}
					<div class="sc-card p-5 border-2 {market.resolution === 'yes' ? 'border-green-400 bg-green-50' : 'border-red-300 bg-red-50'}">
						<h3 class="text-xs font-bold {market.resolution === 'yes' ? 'text-green-600' : 'text-red-500'} uppercase tracking-[0.12em] mb-2">Resolved</h3>
						<div class="text-3xl font-black {market.resolution === 'yes' ? 'text-green-700' : 'text-red-600'} mb-1">
							{market.resolution?.toUpperCase()}
						</div>
						{#if market.resolved_at}
							<p class="text-xs text-surface-500">
								on {new Date(market.resolved_at).toLocaleDateString()}
							</p>
						{/if}
					</div>
				{/if}

				<!-- Trade widget -->
				{#if market.status === 'active' || market.status === 'resolution_requested'}
					{#if $isLoggedIn}
						<div class="sc-card p-5">
							<!-- Buy / Sell action tabs -->
							<div class="flex gap-1 mb-4 p-1 bg-surface-100 rounded-lg">
								<button
									onclick={() => (tradeAction = 'buy')}
									class="flex-1 btn btn-sm text-xs uppercase tracking-wider font-bold transition-colors
										{tradeAction === 'buy' ? 'bg-white shadow text-primary-700 border border-primary-200' : 'text-surface-500 hover:text-surface-700'}"
								>
									Buy
								</button>
								<button
									onclick={() => (tradeAction = 'sell')}
									class="flex-1 btn btn-sm text-xs uppercase tracking-wider font-bold transition-colors
										{tradeAction === 'sell' ? 'bg-white shadow text-surface-700 border border-surface-300' : 'text-surface-500 hover:text-surface-700'}"
								>
									Sell
								</button>
							</div>

							<!-- YES / NO side selector -->
							<div class="flex gap-2 mb-4">
								<button
									onclick={() => (tradeSide = 'yes')}
									class="flex-1 btn btn-sm {tradeSide === 'yes' ? 'bg-green-600 text-white border border-green-600' : 'border border-surface-300 text-surface-600 hover:border-green-400 hover:text-green-700'} transition-colors text-xs uppercase tracking-wider font-bold"
								>
									YES
								</button>
								<button
									onclick={() => (tradeSide = 'no')}
									class="flex-1 btn btn-sm {tradeSide === 'no' ? 'bg-red-600 text-white border border-red-600' : 'border border-surface-300 text-surface-600 hover:border-red-400 hover:text-red-600'} transition-colors text-xs uppercase tracking-wider font-bold"
								>
									NO
								</button>
							</div>

							{#if tradeAction === 'buy'}
								<!-- Shares vs Budget toggle -->
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
									<div class="flex items-center gap-2 mb-1">
										<input
											id="trade-shares"
											type="number"
											bind:value={tradeShares}
											min="1"
											max={maxShares}
											step="1"
											class="sc-input text-sm flex-1"
										/>
										{#if $currentUser && maxShares > 0}
											<button
												onclick={() => (tradeShares = maxShares)}
												class="text-xs text-primary-600 hover:underline whitespace-nowrap"
												title="Max shares you can afford"
											>
												Max ({maxShares})
											</button>
										{/if}
									</div>
									{#if $currentUser}
										<div class="mb-4 text-xs text-surface-500">
											Est. cost: <span class="{canAfford ? 'text-surface-700 font-semibold' : 'text-red-600 font-semibold'}">{estimatedCost.toLocaleString()} bUEC</span>
											{#if !canAfford}
												<span class="text-red-500"> — insufficient balance</span>
											{/if}
										</div>
									{/if}
									<button
										onclick={doTrade}
										disabled={trading || tradeShares <= 0 || !canAfford}
										class="btn preset-filled-primary-500 w-full text-xs uppercase tracking-wider disabled:opacity-50"
									>
										{trading ? 'Buying…' : `Buy ${tradeShares} ${tradeSide.toUpperCase()}`}
									</button>
								{:else}
									<!-- Budget mode: enter how much to spend -->
									<label class="block mb-1" for="trade-budget">
										<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Budget (bUEC)</span>
									</label>
									<div class="flex items-center gap-2 mb-1">
										<input
											id="trade-budget"
											type="number"
											bind:value={budgetAmount}
											min="1"
											max={$currentUser?.balance ?? 99999}
											step="10"
											class="sc-input text-sm flex-1"
										/>
										{#if $currentUser}
											<button
												onclick={() => (budgetAmount = $currentUser!.balance)}
												class="text-xs text-primary-600 hover:underline whitespace-nowrap"
												title="Use entire balance"
											>All in</button>
										{/if}
									</div>
									<div class="mb-4 text-xs text-surface-500">
										You'll get: <span class="text-surface-700 font-semibold">~{budgetShares.toFixed(2)} {tradeSide.toUpperCase()} shares</span>
										{#if !canAffordBudget}
											<span class="text-red-500"> — insufficient balance</span>
										{/if}
									</div>
									<button
										onclick={doTrade}
										disabled={trading || budgetShares <= 0 || !canAffordBudget}
										class="btn preset-filled-primary-500 w-full text-xs uppercase tracking-wider disabled:opacity-50"
									>
										{trading ? 'Buying…' : `Spend ${budgetAmount.toLocaleString()} bUEC on ${tradeSide.toUpperCase()}`}
									</button>
								{/if}
							{:else}
								<!-- Sell form -->
								{@const holdingShares = tradeSide === 'yes' ? (myPos?.yes_shares ?? 0) : (myPos?.no_shares ?? 0)}
								<div class="mb-3 p-2 bg-surface-50 rounded text-xs text-surface-600">
									You hold: <span class="font-bold text-surface-800">{holdingShares}</span> {tradeSide.toUpperCase()} shares
								</div>
								<label class="block mb-1" for="sell-shares">
									<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Shares to Sell</span>
								</label>
								<div class="flex items-center gap-2 mb-1">
									<input
										id="sell-shares"
										type="number"
										bind:value={sellShares}
										min="1"
										max={maxSellShares}
										step="1"
										class="sc-input text-sm flex-1"
									/>
									{#if maxSellShares > 0}
										<button
											onclick={() => (sellShares = maxSellShares)}
											class="text-xs text-primary-600 hover:underline whitespace-nowrap"
											title="Sell all"
										>
											All ({maxSellShares})
										</button>
									{/if}
								</div>
								<div class="mb-4 text-xs text-surface-500">
									Est. revenue: <span class="text-green-700 font-semibold">{estimatedRevenue.toLocaleString()} bUEC</span>
								</div>
								<button
									onclick={doTrade}
									disabled={trading || sellShares <= 0 || sellShares > maxSellShares}
									class="btn w-full border border-surface-400 text-surface-700 hover:bg-surface-100 text-xs uppercase tracking-wider disabled:opacity-50"
								>
									{trading ? 'Selling…' : `Sell ${sellShares} ${tradeSide.toUpperCase()}`}
								</button>
							{/if}

							{#if tradeSuccess}
								<div class="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 mt-3 text-xs">
									{tradeSuccess}
								</div>
							{/if}
							{#if tradeError}
								<div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 mt-3 text-xs">
									{tradeError}
								</div>
							{/if}

							{#if $currentUser}
								<p class="text-surface-400 text-xs mt-3 text-center">
									Balance: {$currentUser.balance.toLocaleString()} bUEC
								</p>
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
		</div>
	{/if}
</div>
