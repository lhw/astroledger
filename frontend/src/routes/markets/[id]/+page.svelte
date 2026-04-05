<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getMarket, executeTrade, requestResolution, getMarketPriceHistory, submitReport, getMe, getComments, postComment, deleteComment } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import { buyCost, sellRevenue, maxAffordable } from '$lib/amm';
	import MarketOverviewColumn from '$lib/components/markets/MarketOverviewColumn.svelte';
	import MarketTradeSidebar from '$lib/components/markets/MarketTradeSidebar.svelte';
	import type { MarketWithPrice, PricePoint, Comment, MarketOutcome } from '$lib/types';
	import { CATEGORY_LABELS } from '$lib/categories';

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
	let selectedOutcomeId = $state<number | null>(null);
	let tradeShares = $state(1);
	let sellShares = $state(1);
	let trading = $state(false);
	let tradeError = $state('');
	let tradeSuccess = $state('');
	// Budget mode — enter bUEC amount instead of share count
	let budgetMode = $state(false);
	let budgetAmount = $state(100);

	// Derived: selected outcome object
	let selectedOutcome = $derived.by(() => {
		if (!data_) return null as MarketOutcome | null;
		const outs = data_.market.outcomes ?? [];
		if (selectedOutcomeId === null && outs.length > 0) return outs[0] as MarketOutcome;
		return (outs.find((o) => o.id === selectedOutcomeId) ?? outs[0] ?? null) as MarketOutcome | null;
	});

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

	// Comments
	let comments = $state<Comment[]>([]);
	let commentInput = $state('');
	let postingComment = $state(false);
	let commentError = $state('');

	// Reactive cost/revenue estimates
	let allShares = $derived((data_?.market.outcomes ?? []).map((o) => o.shares));
	let outcomeIdx = $derived.by(() => {
		const outs = data_?.market.outcomes ?? [];
		const id = selectedOutcome?.id;
		const idx = outs.findIndex((o) => o.id === id);
		return idx >= 0 ? idx : 0;
	});

	let estimatedCost = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return buyCost(m.liquidity_param, allShares, outcomeIdx, Math.max(1, tradeShares));
	});

	let budgetShares = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return maxAffordable(m.liquidity_param, allShares, outcomeIdx, Math.max(0, budgetAmount));
	});

	// Whole shares actually purchased (backend works with integers).
	let floorBudgetShares = $derived(Math.floor(budgetShares));

	// Exact cost for the floored share count — may be less than budgetAmount.
	let actualBudgetCost = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		if (floorBudgetShares <= 0) return 0;
		return buyCost(m.liquidity_param, allShares, outcomeIdx, floorBudgetShares);
	});

	let estimatedRevenue = $derived.by(() => {
		if (!data_) return 0;
		const m = data_.market;
		return sellRevenue(m.liquidity_param, allShares, outcomeIdx, Math.max(1, sellShares));
	});

	let maxShares = $derived.by(() => {
		if (!data_ || !$currentUser) return 1000;
		const m = data_.market;
		return Math.floor(maxAffordable(m.liquidity_param, allShares, outcomeIdx, $currentUser.balance));
	});

	let maxSellShares = $derived.by(() => {
		if (!data_) return 0;
		const pos = (data_.my_positions ?? []).find((p) => p.outcome_id === selectedOutcome?.id);
		return pos ? Math.floor(pos.shares) : 0;
	});

	let canAfford = $derived($currentUser ? estimatedCost <= $currentUser.balance && tradeShares <= maxShares : false);
	let canAffordBudget = $derived($currentUser ? actualBudgetCost <= $currentUser.balance && floorBudgetShares > 0 : false);

	// Chart UI state
	let chartLogScale = $state(false);

	function formatShares(value: number): string {
		if (!Number.isFinite(value)) return '0';
		if (Number.isInteger(value)) return value.toLocaleString();
		return value.toLocaleString(undefined, {
			minimumFractionDigits: 0,
			maximumFractionDigits: 4
		});
	}

	async function loadComments() {
		try {
			comments = await getComments(id);
		} catch {
			// Non-fatal — comments are supplementary
		}
	}

	async function doPostComment() {
		if (commentInput.trim().length === 0) return;
		postingComment = true;
		commentError = '';
		try {
			const c = await postComment(id, commentInput.trim());
			commentInput = '';
			await loadComments();
			if (c.hidden) {
				commentError = '⚠ Your comment was flagged for review and is only visible to you until a moderator clears it.';
			}
		} catch (e) {
			commentError = e instanceof Error ? e.message : 'Could not post comment.';
		} finally {
			postingComment = false;
		}
	}

	async function doDeleteComment(commentId: number) {
		if (!confirm('Delete this comment?')) return;
		try {
			await deleteComment(commentId);
			comments = comments.filter((c) => c.id !== commentId);
		} catch (e) {
			alert(e instanceof Error ? e.message : 'Could not delete comment.');
		}
	}

	onMount(async () => {
		if (initialMarket) {
			// If the market is already resolved and the user is logged in, refresh
			// their balance — payout may have occurred since they last loaded.
			if (initialMarket.market.status === 'resolved' && $isLoggedIn) {
				getMe().then((u) => { if (u) currentUser.set(u); });
			}
			await loadComments();
			return;
		}
		try {
			[data_, priceHistory] = await Promise.all([
				getMarket(id),
				getMarketPriceHistory(id).catch(() => [] as PricePoint[])
			]);
			await loadComments();
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
		if (!$isLoggedIn || !data_ || !selectedOutcome) return;
		trading = true;
		tradeError = '';
		tradeSuccess = '';
		try {
			if (tradeAction === 'buy') {
			// Budget mode: use the floored whole-share count derived from the LMSR inversion.
			const sharesToBuy = budgetMode ? floorBudgetShares : tradeShares;
			if (sharesToBuy <= 0) {
				tradeError = 'Budget too low — cannot purchase any shares at this price.';
				return;
			}
			const result = await executeTrade(id, selectedOutcome.id, 'buy', sharesToBuy);
			const executedShares = result.Shares;
			currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
			tradeSuccess = `Bought ${formatShares(executedShares)} ${selectedOutcome.label} share${executedShares !== 1 ? 's' : ''} for ${result.Cost.toLocaleString()} bUEC.`;
			} else {
				const result = await executeTrade(id, selectedOutcome.id, 'sell', sellShares);
				const executedShares = result.Shares;
				currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
				const received = Math.abs(result.Cost);
				tradeSuccess = `Sold ${formatShares(executedShares)} ${selectedOutcome.label} share${executedShares !== 1 ? 's' : ''} for ${received.toLocaleString()} bUEC.`;
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
	<title>{data_?.market.title ?? 'Market'} — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-10">
	{#if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading market…</div>
	{:else if error}
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-4">{error}</div>
	{:else if data_}
		{@const market = data_.market}
		{@const myPositions = data_.my_positions ?? []}
		{@const hasPosition = myPositions.some((p) => p.shares > 0)}

		<!-- Header -->
		<div class="mb-8">
			<div class="flex items-center gap-2 mb-3">
				<a href="/markets" class="text-surface-500 hover:text-primary-600 text-xs uppercase tracking-wider no-underline transition-colors">← Markets</a>
				<span class="text-surface-300">/</span>
				<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
				{#if market.status === 'resolved'}
					{@const winLabel = market.outcomes?.find((o) => o.id === market.resolved_outcome_id)?.label ?? 'RESOLVED'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-green-100 text-green-700 border border-green-200">
						Resolved: {winLabel}
					</span>
				{:else if market.status === 'resolution_requested'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Awaiting Resolution</span>
				{:else if market.status === 'deadline_passed'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-orange-100 text-orange-700 border border-orange-200">Deadline Passed</span>
				{:else if market.status === 'active'}
					<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-primary-50 text-primary-700 border border-primary-200">Active</span>
				{/if}
			</div>
			<h1 class="text-2xl font-bold text-surface-900 leading-snug">{market.title}</h1>
			<p class="text-surface-500 text-sm mt-2 flex flex-wrap items-center gap-x-1.5 gap-y-1">
				<span>Submitted by</span>
				<span class="font-medium text-surface-800">{market.creator_name}</span>
				<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-primary-50 text-primary-700 border border-primary-200">Creator</span>
				<span>· {market.status === 'resolved' ? 'resolved' : 'closes'} {new Date(market.resolution_deadline).toLocaleDateString()}</span>
			</p>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-5">
			<MarketOverviewColumn
				{market}
				{priceHistory}
				{myPositions}
				{hasPosition}
				bind:chartLogScale
				{comments}
				{requestingResolution}
				{resolutionRequestMsg}
				bind:resolutionLink
				bind:resolutionNote
				bind:showReportForm
				bind:reportReason
				{submittingReport}
				{reportMsg}
				bind:commentInput
				{postingComment}
				{commentError}
				onRequestResolution={doRequestResolution}
				onSubmitReport={doSubmitReport}
				onPostComment={doPostComment}
				onDeleteComment={doDeleteComment}
			/>

			<MarketTradeSidebar
				{market}
				{myPositions}
				totalVolume={data_.total_volume}
				traderCount={data_.trader_count}
				tradeCount={data_.trade_count}
				bind:tradeAction
				bind:selectedOutcomeId
				{selectedOutcome}
				bind:tradeShares
				bind:sellShares
				bind:budgetMode
				bind:budgetAmount
				{trading}
				{tradeSuccess}
				{tradeError}
				{estimatedCost}
				{estimatedRevenue}
				{actualBudgetCost}
				{maxShares}
				{maxSellShares}
				{floorBudgetShares}
				{canAfford}
				{canAffordBudget}
				onTrade={doTrade}
			/>
		</div>
	{/if}
</div>