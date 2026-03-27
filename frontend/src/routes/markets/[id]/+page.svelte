<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getMarket, executeTrade, requestResolution, getMarketPriceHistory, submitReport, getMe, getComments, postComment, deleteComment } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import { buyCost, sellRevenue, maxAffordable } from '$lib/amm';
	import { renderMarkdown } from '$lib/markdown';
	import type { MarketWithPrice, PricePoint, Comment, MarketOutcome } from '$lib/types';
	import { CATEGORY_LABELS } from '$lib/categories';

	// Badge tier map for inline badge display in comments.
	const BADGE_TIERS: Record<string, { tier: number; title: string; symbol: string }> = {
		// ── Earned ──────────────────────────────────────────────────────────
		first_blood:              { tier: 1, title: 'First Blood',               symbol: '▲' },
		quick_shot:               { tier: 1, title: 'Quick Shot',                symbol: '▲' },
		market_founder:           { tier: 1, title: 'Market Founder',            symbol: '▲' },
		eternal_optimist:         { tier: 2, title: 'Eternal Optimist',          symbol: '●' },
		doomsayer:                { tier: 2, title: 'Doomsayer',                 symbol: '●' },
		market_maven:             { tier: 2, title: 'Market Maven',              symbol: '●' },
		seasoned_trader:          { tier: 2, title: 'Seasoned Trader',           symbol: '●' },
		skeptic:                  { tier: 2, title: 'Skeptic',                   symbol: '●' },
		portfolio_manager:        { tier: 2, title: 'Portfolio Manager',         symbol: '●' },
		serial_founder:           { tier: 2, title: 'Serial Founder',            symbol: '●' },
		bug_prophet:              { tier: 3, title: 'Bug Prophet',               symbol: '◆' },
		market_obsessed:          { tier: 3, title: 'Market Obsessed',           symbol: '◆' },
		universe_citizen:         { tier: 3, title: 'Universe Citizen',          symbol: '◆' },
		galaxy_brained:           { tier: 4, title: 'Galaxy Brained',            symbol: '◈' },
		oracle:                   { tier: 4, title: 'Oracle',                    symbol: '◈' },
		// ── FOMO Store ──────────────────────────────────────────────────────
		citizen_backer:           { tier: 1, title: 'Citizen Backer',            symbol: '▲' },
		professional_bug_finder:  { tier: 1, title: 'Professional Bug Finder',   symbol: '▲' },
		aurora_pilot:             { tier: 1, title: 'Aurora Pilot',              symbol: '▲' },
		roadmap_reader:           { tier: 1, title: 'Roadmap Reader',            symbol: '▲' },
		warp_speed:               { tier: 1, title: 'Warp Speed',                symbol: '▲' },
		mostly_backer:            { tier: 2, title: 'Mostly Backer',             symbol: '●' },
		hangar_queen:             { tier: 2, title: 'Hangar Queen',              symbol: '●' },
		tech_preview_survivor:    { tier: 2, title: 'Tech Preview Survivor',     symbol: '●' },
		star_gazer:               { tier: 2, title: 'Star Gazer',               symbol: '●' },
		alpha_tester:             { tier: 2, title: 'Alpha Tester',              symbol: '●' },
		space_whale:              { tier: 2, title: 'Space Whale',               symbol: '●' },
		bugged_not_broken:        { tier: 2, title: 'Bugged, Not Broken',        symbol: '●' },
		verse_veteran:            { tier: 2, title: "'Verse Veteran",            symbol: '●' },
		alpha_optimist:           { tier: 2, title: 'Alpha Optimist',            symbol: '●' },
		q4_enjoyer:               { tier: 2, title: 'Q4 Enjoyer',               symbol: '●' },
		persistent_citizen:       { tier: 3, title: 'Persistent Universe Citizen', symbol: '◆' },
		org_leader:               { tier: 3, title: 'Org Leader',                symbol: '◆' },
		'900i_enjoyer':           { tier: 3, title: '900i Enjoyer',              symbol: '◆' },
		system_colonist:          { tier: 3, title: 'System Colonist',           symbol: '◆' },
		citizencon_pilgrim:       { tier: 3, title: 'CitizenCon Pilgrim',        symbol: '◆' },
		idris_captain:            { tier: 4, title: 'Idris Captain',             symbol: '◈' },
		backer_royalty:           { tier: 4, title: 'Backer Royalty',            symbol: '◈' },
		fleet_commander_badge:    { tier: 4, title: 'Fleet Commander',           symbol: '◈' },
		golden_ticket:            { tier: 5, title: 'Golden Ticket',             symbol: '★' },
		unobtainium:              { tier: 5, title: 'Unobtainium Tier',          symbol: '★' },
		// ── Admiral Rank ────────────────────────────────────────────────────
		ensign:                   { tier: 1, title: 'Ensign',                    symbol: '⚓' },
		lieutenant:               { tier: 2, title: 'Lieutenant',                symbol: '⚔' },
		commander:                { tier: 3, title: 'Commander',                 symbol: '🛡' },
		captain:                  { tier: 4, title: 'Captain',                   symbol: '👑' },
		coin_admiral:             { tier: 5, title: 'Coin Admiral',              symbol: '🌟' },
	};

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
			currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
			tradeSuccess = `Bought ${sharesToBuy} ${selectedOutcome.label} share${sharesToBuy !== 1 ? 's' : ''} for ${result.Cost.toLocaleString()} bUEC.`;
			} else {
				const result = await executeTrade(id, selectedOutcome.id, 'sell', sellShares);
				currentUser.update((u) => (u ? { ...u, balance: result.NewBalance } : u));
				const received = Math.abs(result.Cost);
				tradeSuccess = `Sold ${sellShares} ${selectedOutcome.label} share${sellShares !== 1 ? 's' : ''} for ${received.toLocaleString()} bUEC.`;
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
					<div class="flex items-center justify-between mb-3">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em]">Price History</h3>
						{#if priceHistory.length > 1}
							<button
								onclick={() => (chartLogScale = !chartLogScale)}
								class="text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded border transition-colors
									{chartLogScale ? 'border-primary-400 text-primary-600 bg-primary-50' : 'border-surface-300 text-surface-400 hover:border-surface-400'}"
								title="Toggle log scale"
							>Log</button>
						{/if}
					</div>
					{#if priceHistory.length > 1}
						{@const W = 600}
						{@const H = 100}
						{@const n = priceHistory.length}
						{@const scaleP = (p: number) => chartLogScale
							? Math.log1p(p * 9) / Math.log1p(9)
							: p}
						{@const yOf = (p: number) => H - scaleP(Math.max(0.001, Math.min(0.999, p))) * H}
						{@const xOf = (i: number) => (i / (n - 1)) * W}
						{@const lastP  = priceHistory.at(-1)!.price_at_trade}
						{@const resolved = market.resolved_outcome_id !== null}
						{@const outcomeLabels = [...new Set(priceHistory.map((p) => p.outcome_label))]}
						<!-- Legend -->
						<div class="flex flex-wrap items-center gap-3 mb-2">
							{#each outcomeLabels as label, i}
								{@const colors = ['#D4A843','#f87171','#60a5fa','#34d399','#a78bfa']}
								<span class="flex items-center gap-1 text-xs font-semibold text-surface-600">
									<span class="inline-block w-3 h-0.5 rounded" style="background:{colors[i % colors.length]}"></span>
									{label}
								</span>
							{/each}
							<span class="text-surface-400 text-xs ml-auto">{priceHistory.length} trades</span>
						</div>
						<svg viewBox="0 0 {W} {H}" class="w-full rounded" style="height:100px;background:#fafaf8">
							<!-- 50% reference line -->
							<line x1="0" y1={yOf(0.5)} x2={W} y2={yOf(0.5)} stroke="#e5e7eb" stroke-width="1" stroke-dasharray="4 4"/>
							{#each outcomeLabels as label, i}
								{@const colors = ['#D4A843','#f87171','#60a5fa','#34d399','#a78bfa']}
								{@const pts = priceHistory
									.map((p, idx) => p.outcome_label === label ? `${xOf(idx)},${yOf(p.price_at_trade)}` : null)
									.filter(Boolean).join(' ')}
								<polyline points={pts} fill="none" stroke={colors[i % colors.length]} stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>
							{/each}
							<!-- Resolution marker -->
							{#if resolved}
								<line x1={W} y1="0" x2={W} y2={H} stroke="#16a34a" stroke-width="2" stroke-dasharray="4 3" opacity="0.8"/>
							{/if}
							<!-- Latest dot -->
							<circle cx={W} cy={yOf(lastP)} r="4" fill="#D4A843"/>
						</svg>
						<!-- Time axis -->
						<div class="flex justify-between mt-1">
							{#each [0, 0.25, 0.5, 0.75, 1] as frac}
								{@const idx = Math.round(frac * (n - 1))}
								<span class="text-surface-400 text-[10px]">
									{new Date(priceHistory[idx].created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
								</span>
							{/each}
						</div>
					{:else}
						<p class="text-surface-400 text-xs text-center py-4">No trades yet — be the first!</p>
					{/if}
				</div>

				<!-- My current position -->
				{#if hasPosition}
					<div class="sc-card p-5 border-l-4 border-l-primary-400 bg-amber-50/30">
						<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-3">Your Position</h3>
						<div class="flex gap-3 flex-wrap">
							{#each myPositions.filter((p) => p.shares > 0) as pos}
								<div class="flex-1 text-center p-3 rounded bg-primary-50 border border-primary-100 min-w-20">
									<div class="text-2xl font-bold text-primary-700">{Math.floor(pos.shares)}</div>
									<div class="text-primary-600 text-xs mt-0.5 uppercase tracking-wider font-semibold">{pos.label}</div>
								</div>
							{/each}
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

				<!-- Discussion / Comments -->
				<div class="sc-card p-5">
					<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-4">
						Discussion ({comments.length})
					</h3>

					{#if comments.length === 0}
						<p class="text-surface-400 text-xs text-center py-3">No comments yet — start the conversation.</p>
					{:else}
						<div class="space-y-5 mb-5">
							{#each comments as comment}
								<div>
									<div class="flex items-center gap-1.5 mb-1 flex-wrap">
										<span class="font-semibold text-surface-800 text-xs">{comment.author_name}</span>
										{#if comment.author_top_badge && BADGE_TIERS[comment.author_top_badge]}
											{@const b = BADGE_TIERS[comment.author_top_badge]}
											<span
												class="comment-badge tier-{b.tier}"
												title={b.title}
											>{b.symbol} {b.title}</span>
										{/if}
										<span class="text-surface-400 text-[10px]">{new Date(comment.created_at).toLocaleString()}</span>
										{#if $currentUser?.is_moderator}
											<button
												onclick={() => doDeleteComment(comment.id)}
												class="text-[10px] text-red-400 hover:text-red-600 transition-colors ml-1"
											>Delete</button>
										{/if}
									</div>
									{#if comment.hidden}
										<div class="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded p-2 mb-1.5">
											<span class="font-bold">⚠ Under review</span> — your comment is only visible to you until a moderator clears it.
										</div>
										<div class="text-surface-500 text-sm prose prose-sm max-w-none opacity-60">
											{@html renderMarkdown(comment.content)}
										</div>
									{:else}
										<div class="text-surface-700 text-sm prose prose-sm max-w-none">
											{@html renderMarkdown(comment.content)}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}

					{#if $isLoggedIn}
						<div class="{comments.length > 0 ? 'border-t border-surface-100 pt-4' : ''}">
							<textarea
								bind:value={commentInput}
								rows="3"
								maxlength="1000"
								placeholder="Add a comment… Markdown supported. Be excellent to each other."
								class="sc-input text-sm resize-none w-full"
							></textarea>
							<div class="flex items-center justify-between mt-2">
								<span class="text-[10px] text-surface-400">{commentInput.length}/1000 · auto-checked for abuse</span>
								<button
									onclick={doPostComment}
									disabled={postingComment || commentInput.trim().length === 0}
									class="btn btn-sm preset-filled-primary-500 text-xs uppercase tracking-wider disabled:opacity-50"
								>{postingComment ? 'Posting…' : 'Post'}</button>
							</div>
							{#if commentError}
								<p class="text-xs mt-2 {commentError.startsWith('⚠') ? 'text-amber-700' : 'text-red-600'}">{commentError}</p>
							{/if}
						</div>
					{:else}
						<p class="text-surface-400 text-xs text-center pt-3 border-t border-surface-100">
							<a href="/auth/login" class="text-primary-600 hover:underline">Log in</a> to join the discussion.
						</p>
					{/if}
				</div>
			</div>

			<!-- Trade widget + prices -->
			<div class="space-y-4">
				<!-- Current prices (outcomes) -->
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
					{@const winOutcome = (market.outcomes ?? []).find((o) => o.id === market.resolved_outcome_id)}
					<div class="sc-card p-5 border-2 border-primary-400 bg-primary-50/30">
						<h3 class="text-xs font-bold text-primary-600 uppercase tracking-[0.12em] mb-2">Resolved</h3>
						<div class="text-3xl font-black text-primary-700 mb-1">
							{winOutcome?.label ?? '—'}
						</div>
						<p class="text-xs text-surface-500">
							{#if market.resolver_name}by <span class="font-medium text-surface-700">{market.resolver_name}</span>{/if}
							{#if market.resolved_at} · {new Date(market.resolved_at).toLocaleDateString()}{/if}
						</p>
						{#if market.resolution_evidence}
							<a href={market.resolution_evidence} target="_blank" rel="noopener noreferrer"
							   class="text-xs text-primary-600 hover:underline mt-2 block truncate">
								📋 Evidence: {market.resolution_evidence}
							</a>
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

							<!-- Outcome selector -->
							<div class="mb-4">
								<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold block mb-2">Outcome</span>
								<div class="flex gap-2 flex-wrap">
									{#each (market.outcomes ?? []) as outcome}
										<button
											onclick={() => (selectedOutcomeId = outcome.id)}
											class="flex-1 btn btn-sm {selectedOutcome?.id === outcome.id ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400 hover:text-primary-700'} transition-colors text-xs uppercase tracking-wider font-bold"
										>
											{outcome.label} <span class="opacity-70 normal-case font-normal">{outcome.price}%</span>
										</button>
									{/each}
								</div>
							</div>
							<!-- Buy/Sell inputs -->
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
										Cost: <span class="{canAfford ? 'text-surface-700 font-semibold' : 'text-red-600 font-semibold'}">{estimatedCost.toLocaleString()} bUEC</span>
										{#if tradeShares > maxShares}
											<span class="text-red-500"> — max is {maxShares} shares with your balance</span>
										{:else if !canAfford}
											<span class="text-red-500"> — insufficient balance</span>
										{/if}
									</div>
								{/if}
								<button
									onclick={doTrade}
									disabled={trading || tradeShares <= 0 || !canAfford}
									class="btn preset-filled-primary-500 w-full text-xs uppercase tracking-wider disabled:opacity-50"
								>
										{trading ? 'Buying…' : `Buy ${tradeShares} ${selectedOutcome?.label ?? ''} for ${estimatedCost.toLocaleString()} bUEC`}
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
									<div class="mb-4 text-xs text-surface-500 space-y-0.5">
										{#if floorBudgetShares > 0}
											<div>You'll buy: <span class="text-surface-700 font-semibold">{floorBudgetShares} {selectedOutcome?.label ?? ''} share{floorBudgetShares !== 1 ? 's' : ''}</span></div>
											<div>Actual cost: <span class="{canAffordBudget ? 'text-surface-700 font-semibold' : 'text-red-600 font-semibold'}">{actualBudgetCost.toLocaleString()} bUEC</span>
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
									<button
										onclick={doTrade}
										disabled={trading || floorBudgetShares <= 0 || !canAffordBudget}
										class="btn preset-filled-primary-500 w-full text-xs uppercase tracking-wider disabled:opacity-50"
									>
										{trading ? 'Buying…' : `Buy ${floorBudgetShares} ${selectedOutcome?.label ?? ''} for ${actualBudgetCost.toLocaleString()} bUEC`}
									</button>
								{/if}
							{:else}
								<!-- Sell form -->
									{@const holdingShares = (myPositions ?? []).find((p) => p.outcome_id === selectedOutcome?.id)?.shares ?? 0}
									<div class="mb-3 p-2 bg-surface-50 rounded text-xs text-surface-600">
										You hold: <span class="font-bold text-surface-800">{holdingShares}</span> {selectedOutcome?.label ?? ''} shares
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
										{trading ? 'Selling…' : `Sell ${sellShares} ${selectedOutcome?.label ?? ''}`}
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
<style>
/* ─── Inline comment badges ──────────────────────────────────────── */
.comment-badge {
	display: inline-flex;
	align-items: center;
	gap: 0.2rem;
	padding: 0.1rem 0.45rem;
	border-radius: 9999px;
	font-size: 0.6rem;
	font-weight: 700;
	letter-spacing: 0.04em;
	white-space: nowrap;
}
.comment-badge.tier-1 {
	background: #f5ede0;
	border: 1px solid #d4b896;
	color: #7a5030;
}
.comment-badge.tier-2 {
	background: linear-gradient(90deg, #fef3d0, #fde68a);
	border: 1px solid #e6c96b;
	color: #92400e;
}
.comment-badge.tier-3 {
	background: linear-gradient(90deg, #fef9e0, #fef3c0);
	border: 1px solid #f0c040;
	color: #6b2d06;
}
.comment-badge.tier-4 {
	background: linear-gradient(90deg, #1c1008, #2d1c00);
	border: 1px solid #fbbf24;
	color: #fef3c7;
}
.comment-badge.tier-5 {
	background: linear-gradient(90deg, #0d0d0d, #1a1200);
	border: 1px solid #ffd700;
	color: #ffd700;
}
</style>