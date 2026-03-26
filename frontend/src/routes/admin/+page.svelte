<script lang="ts">
	import {
		adminTriggerWeeklyPayout,
		adminAdjustBalance,
		adminGetAnalytics,
		adminSearchUsers,
		ApiClientError
	} from '$lib/api';
	import type { AnalyticsStats, UserSearchResult } from '$lib/types';

	// ── Tab state ─────────────────────────────────────────────────────────
	let activeTab = $state<'operations' | 'analytics'>('operations');

	// ── Operations: payout ────────────────────────────────────────────────
	let payoutLoading = $state(false);
	let payoutResult = $state<string | null>(null);
	let payoutError = $state<string | null>(null);

	// ── Operations: balance adjustment ────────────────────────────────────
	let adjSearchQuery = $state('');
	let adjSearchResults = $state<UserSearchResult[]>([]);
	let adjSearchLoading = $state(false);
	let adjSelectedUser = $state<UserSearchResult | null>(null);
	let adjAmount = $state('');
	let adjReason = $state('');
	let adjLoading = $state(false);
	let adjResult = $state<string | null>(null);
	let adjError = $state<string | null>(null);

	let searchTimer: ReturnType<typeof setTimeout> | null = null;

	function onSearchInput() {
		adjSelectedUser = null;
		adjSearchResults = [];
		if (searchTimer) clearTimeout(searchTimer);
		const q = adjSearchQuery.trim();
		if (q.length < 2) return;
		searchTimer = setTimeout(async () => {
			adjSearchLoading = true;
			try {
				adjSearchResults = await adminSearchUsers(q);
			} catch {
				adjSearchResults = [];
			} finally {
				adjSearchLoading = false;
			}
		}, 300);
	}

	function selectUser(u: UserSearchResult) {
		adjSelectedUser = u;
		adjSearchQuery = '';
		adjSearchResults = [];
	}

	function clearSelectedUser() {
		adjSelectedUser = null;
		adjAmount = '';
		adjReason = '';
		adjResult = null;
		adjError = null;
	}

	// ── Analytics state ───────────────────────────────────────────────────
	let analyticsPeriod = $state<'7d' | '30d'>('7d');
	let analyticsLoading = $state(false);
	let analyticsData = $state<AnalyticsStats | null>(null);
	let analyticsError = $state<string | null>(null);

	async function triggerPayout() {
		payoutLoading = true;
		payoutResult = null;
		payoutError = null;
		try {
			const res = await adminTriggerWeeklyPayout();
			payoutResult = `${res.message} — ${res.users_paid} users received ${res.credits_per_user} bUEC each.`;
		} catch (err) {
			payoutError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			payoutLoading = false;
		}
	}

	async function submitAdjustment() {
		adjResult = null;
		adjError = null;
		if (!adjSelectedUser) {
			adjError = 'Select a user first.';
			return;
		}
		const amount = parseInt(adjAmount, 10);
		if (!Number.isInteger(amount) || amount === 0) {
			adjError = 'Amount must be a non-zero integer.';
			return;
		}
		if (!adjReason.trim()) {
			adjError = 'Reason is required.';
			return;
		}
		if (adjReason.length > 200) {
			adjError = 'Reason must be at most 200 characters.';
			return;
		}
		adjLoading = true;
		try {
			const res = await adminAdjustBalance(adjSelectedUser.id, amount, adjReason.trim());
			adjResult = `Done. ${adjSelectedUser.display_name} new balance: ${res.new_balance.toLocaleString()} bUEC.`;
			adjAmount = '';
			adjReason = '';
			adjSelectedUser = null;
		} catch (err) {
			adjError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjLoading = false;
		}
	}

	async function loadAnalytics() {
		analyticsLoading = true;
		analyticsError = null;
		try {
			analyticsData = await adminGetAnalytics(analyticsPeriod);
		} catch (err) {
			analyticsError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			analyticsLoading = false;
		}
	}

	function switchToAnalytics() {
		activeTab = 'analytics';
		if (!analyticsData) loadAnalytics();
	}

	// ── Chart helpers ─────────────────────────────────────────────────────
	// Bar chart is a simple inline SVG — no dependencies needed.
	const CHART_W = 840;
	const CHART_H = 96;

	function chartBars(daily: AnalyticsStats['daily']) {
		if (!daily?.length) return [];
		const maxV = Math.max(...daily.map((d) => d.views), 1);
		const n = daily.length;
		const slotW = CHART_W / n;
		const gap = Math.max(1, slotW * 0.15);
		return daily.map((d, i) => ({
			x: i * slotW + gap / 2,
			y: CHART_H - (d.views / maxV) * CHART_H,
			w: slotW - gap,
			h: (d.views / maxV) * CHART_H,
			views: d.views,
			date: d.date
		}));
	}

	function xLabels(daily: AnalyticsStats['daily']) {
		if (!daily?.length) return [];
		const n = daily.length;
		const slotW = CHART_W / n;
		// For 7 days: show all; for 30 days: every 5th
		const step = n <= 7 ? 1 : 5;
		return daily
			.map((d, i) => ({ i, date: d.date, x: i * slotW + slotW / 2 }))
			.filter((_, i) => i % step === 0 || i === n - 1);
	}

	function fmtDate(iso: string) {
		const [, m, d] = iso.split('-');
		const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
		return `${months[parseInt(m) - 1]} ${parseInt(d)}`;
	}

	function avgDailyViews(data: AnalyticsStats) {
		const n = data.daily?.length || 1;
		return Math.round(data.total_views / n);
	}
</script>

<svelte:head>
	<title>Admin Panel — ScolyMarket</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-6 py-10">
	<h1 class="text-2xl font-bold text-primary-400 tracking-widest uppercase mb-6">Admin Panel</h1>

	<!-- Tab bar -->
	<div class="flex gap-0 border-b border-surface-700 mb-8">
		<button
			onclick={() => (activeTab = 'operations')}
			class="px-5 py-2.5 text-xs font-semibold uppercase tracking-widest border-b-2 -mb-px transition-colors {activeTab === 'operations'
				? 'border-primary-400 text-primary-400'
				: 'border-transparent text-surface-400 hover:text-surface-200'}"
		>
			Operations
		</button>
		<button
			onclick={switchToAnalytics}
			class="px-5 py-2.5 text-xs font-semibold uppercase tracking-widest border-b-2 -mb-px transition-colors {activeTab === 'analytics'
				? 'border-primary-400 text-primary-400'
				: 'border-transparent text-surface-400 hover:text-surface-200'}"
		>
			Analytics
		</button>
	</div>

	<!-- ── Operations Tab ─────────────────────────────────────────────── -->
	{#if activeTab === 'operations'}
		<div class="space-y-8 max-w-2xl">
			<!-- Weekly Payout -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 space-y-4">
				<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">
					Weekly Payout
				</h2>
				<p class="text-surface-400 text-sm leading-relaxed">
					Manually trigger the 200 bUEC weekly credit payout for all users. Idempotent — safe to
					call even if the cron already ran this week.
				</p>
				<button
					onclick={triggerPayout}
					disabled={payoutLoading}
					class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50"
				>
					{payoutLoading ? 'Running…' : 'Trigger Weekly Payout'}
				</button>
				{#if payoutResult}
					<p class="text-green-400 text-sm">{payoutResult}</p>
				{/if}
				{#if payoutError}
					<p class="text-red-400 text-sm">Error: {payoutError}</p>
				{/if}
			</section>

			<!-- Balance Adjustment -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 space-y-4">
				<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">
					Adjust User Balance
				</h2>
				<p class="text-surface-400 text-sm leading-relaxed">
					Add or remove bUEC from a user's balance. Positive to add, negative to remove. Cannot go
					below 0.
				</p>
				<div class="space-y-3">
					<!-- User search / selected user -->
					{#if adjSelectedUser}
						<!-- Selected user chip -->
						<div class="flex items-center justify-between gap-3 bg-surface-700 border border-primary-600 rounded-lg px-3 py-2.5">
							<div class="flex items-center gap-3 min-w-0">
								<div class="w-2 h-2 rounded-full bg-primary-400 shrink-0"></div>
								<div class="min-w-0">
									<p class="text-surface-100 text-sm font-medium truncate">{adjSelectedUser.display_name}</p>
									<p class="text-surface-400 text-xs">
										{#if adjSelectedUser.rsi_handle}{adjSelectedUser.rsi_handle} · {/if}ID {adjSelectedUser.id} · {adjSelectedUser.balance.toLocaleString()} bUEC
									</p>
								</div>
							</div>
							<button onclick={clearSelectedUser} class="text-surface-500 hover:text-surface-200 text-lg leading-none shrink-0" aria-label="Clear selection">×</button>
						</div>
					{:else}
						<!-- Search input -->
						<div class="relative">
							<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-user-search">Search User</label>
							<div class="relative">
								<input
									id="adj-user-search"
									type="text"
									bind:value={adjSearchQuery}
									oninput={onSearchInput}
									placeholder="Type display name or RSI handle…"
									autocomplete="off"
									class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none pr-8"
								/>
								{#if adjSearchLoading}
									<span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-surface-400 text-xs">…</span>
								{/if}
							</div>
							<!-- Search results dropdown -->
							{#if adjSearchResults.length > 0}
								<ul class="absolute z-10 left-0 right-0 top-full mt-1 bg-surface-800 border border-surface-600 rounded-lg overflow-hidden shadow-xl">
									{#each adjSearchResults as u}
										<li>
											<button
												onclick={() => selectUser(u)}
												class="w-full text-left px-3 py-2.5 hover:bg-surface-700 transition-colors flex items-center justify-between gap-3"
											>
												<div class="min-w-0">
													<p class="text-surface-100 text-sm font-medium truncate">{u.display_name}</p>
													<p class="text-surface-400 text-xs">{#if u.rsi_handle}{u.rsi_handle} · {/if}ID {u.id}</p>
												</div>
												<span class="text-surface-400 text-xs tabular-nums shrink-0">{u.balance.toLocaleString()} bUEC</span>
											</button>
										</li>
									{/each}
								</ul>
							{:else if adjSearchQuery.trim().length >= 2 && !adjSearchLoading}
								<div class="absolute z-10 left-0 right-0 top-full mt-1 bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-500 text-xs">
									No users found.
								</div>
							{/if}
						</div>
					{/if}

					<!-- Amount + reason — only shown once a user is selected -->
					{#if adjSelectedUser}
						<div>
							<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-amount">Amount (bUEC)</label>
							<input
								id="adj-amount"
								type="number"
								bind:value={adjAmount}
								placeholder="e.g. 500 or -100"
								class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none"
							/>
						</div>
						<div>
							<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-reason">Reason</label>
							<textarea
								id="adj-reason"
								bind:value={adjReason}
								rows={3}
								maxlength={200}
								placeholder="E.g. compensation for bug, contest prize…"
								class="textarea w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm resize-none focus:border-primary-500 outline-none"
							></textarea>
							<p class="text-surface-500 text-xs mt-1">{adjReason.length}/200</p>
						</div>
						<button
							onclick={submitAdjustment}
							disabled={adjLoading}
							class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50"
						>
							{adjLoading ? 'Adjusting…' : 'Apply Adjustment'}
						</button>
					{/if}
				</div>
				{#if adjResult}
					<p class="text-green-400 text-sm">{adjResult}</p>
				{/if}
				{#if adjError}
					<p class="text-red-400 text-sm">Error: {adjError}</p>
				{/if}
			</section>
		</div>
	{/if}

	<!-- ── Analytics Tab ──────────────────────────────────────────────── -->
	{#if activeTab === 'analytics'}
		<div class="space-y-6">

			<!-- Period toggle -->
			<div class="flex items-center justify-between">
				<p class="text-surface-400 text-xs uppercase tracking-widest">Traffic overview</p>
				<div class="flex gap-1 bg-surface-800 border border-surface-700 rounded-lg p-1">
					{#each (['7d', '30d'] as const) as p}
						<button
							onclick={async () => { analyticsPeriod = p; await loadAnalytics(); }}
							disabled={analyticsLoading}
							class="px-3 py-1 text-xs font-semibold uppercase tracking-wider rounded-md transition-colors disabled:opacity-40 {analyticsPeriod === p
								? 'bg-primary-500 text-white'
								: 'text-surface-400 hover:text-surface-200'}"
						>{p === '7d' ? '7 Days' : '30 Days'}</button>
					{/each}
				</div>
			</div>

			{#if analyticsLoading}
				<div class="text-surface-400 text-sm py-12 text-center tracking-wide">Loading analytics…</div>
			{:else if analyticsError}
				<div class="bg-red-950/40 border border-red-800 rounded-xl p-5 text-red-400 text-sm">
					Failed to load analytics: {analyticsError}
				</div>
			{:else if analyticsData && !analyticsData.configured}
				<!-- Not configured state -->
				<div class="bg-surface-800 border border-surface-700 rounded-xl p-8 text-center space-y-3">
					<p class="text-surface-200 text-sm font-semibold">GoatCounter not configured</p>
					<p class="text-surface-400 text-xs leading-relaxed max-w-md mx-auto">
						Set <code class="text-primary-400 bg-surface-700 px-1 rounded">GOATCOUNTER_API_KEY</code>
						in your environment. Create an API token in the GoatCounter settings UI at your stats
						subdomain, then redeploy.
					</p>
				</div>
			{:else if analyticsData}
				<!-- ── Stat cards ────────────────────────────────────────── -->
				<div class="grid grid-cols-3 gap-4">
					{#each [
						{ label: 'Total Views', value: analyticsData.total_views.toLocaleString(), sub: analyticsPeriod === '7d' ? 'last 7 days' : 'last 30 days' },
						{ label: 'Unique Visitors', value: analyticsData.total_unique.toLocaleString(), sub: 'estimated' },
						{ label: 'Avg Daily Views', value: avgDailyViews(analyticsData).toLocaleString(), sub: 'per day' }
					] as card}
						<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
							<p class="text-surface-400 text-xs uppercase tracking-widest mb-2">{card.label}</p>
							<p class="text-2xl font-bold text-primary-300">{card.value}</p>
							<p class="text-surface-500 text-xs mt-1">{card.sub}</p>
						</div>
					{/each}
				</div>

				<!-- ── Daily traffic bar chart ───────────────────────────── -->
				{#if analyticsData.daily?.length}
					<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
						<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Daily Traffic</p>
						<div class="text-primary-400 w-full overflow-hidden">
							<svg
								viewBox="0 0 {CHART_W} {CHART_H + 28}"
								class="w-full"
								aria-label="Daily traffic bar chart"
							>
								<!-- Baseline -->
								<line x1="0" y1={CHART_H} x2={CHART_W} y2={CHART_H} stroke="currentColor" stroke-opacity="0.15" stroke-width="1" />

								<!-- Bars -->
								{#each chartBars(analyticsData.daily) as bar}
									<g>
										<rect
											x={bar.x}
											y={bar.y}
											width={bar.w}
											height={bar.h}
											fill="currentColor"
											opacity="0.85"
											rx="2"
										/>
										<title>{fmtDate(bar.date)}: {bar.views.toLocaleString()} views</title>
									</g>
								{/each}

								<!-- X axis labels -->
								{#each xLabels(analyticsData.daily) as lbl}
									<text
										x={lbl.x}
										y={CHART_H + 18}
										text-anchor="middle"
										font-size="11"
										fill="currentColor"
										opacity="0.5"
									>{fmtDate(lbl.date)}</text>
								{/each}
							</svg>
						</div>
					</div>
				{/if}

				<!-- ── Bottom row: top pages + top referrers ─────────────── -->
				<div class="grid grid-cols-2 gap-4">

					<!-- Top pages -->
					<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
						<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Top Pages</p>
						{#if analyticsData.top_pages?.length}
							<div class="space-y-2">
								{#each analyticsData.top_pages as page}
									<div class="flex items-center justify-between gap-3 group">
										<span class="text-surface-300 text-xs truncate font-mono flex-1" title={page.path}>{page.path || '/'}</span>
										<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{page.views.toLocaleString()}</span>
									</div>
									<!-- Mini progress bar -->
									<div class="w-full bg-surface-700 rounded-full h-0.5">
										<div
											class="bg-primary-500 h-0.5 rounded-full transition-all"
											style="width: {Math.round((page.views / (analyticsData.top_pages[0]?.views || 1)) * 100)}%"
										></div>
									</div>
								{/each}
							</div>
						{:else}
							<p class="text-surface-500 text-xs">No page data yet.</p>
						{/if}
					</div>

					<!-- Top referrers -->
					<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
						<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Top Referrers</p>
						{#if analyticsData.top_refs?.length}
							<div class="space-y-2">
								{#each analyticsData.top_refs as ref}
									<div class="flex items-center justify-between gap-3">
										<span class="text-surface-300 text-xs truncate flex-1" title={ref.name}>{ref.name || '(direct)'}</span>
										<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{ref.views.toLocaleString()}</span>
									</div>
									<div class="w-full bg-surface-700 rounded-full h-0.5">
										<div
											class="bg-primary-500 h-0.5 rounded-full"
											style="width: {Math.round((ref.views / (analyticsData.top_refs[0]?.views || 1)) * 100)}%"
										></div>
									</div>
								{/each}
							</div>
						{:else}
							<p class="text-surface-500 text-xs">No referrer data yet.</p>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

