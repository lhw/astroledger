<script lang="ts">
	import {
		adminTriggerWeeklyPayout,
		adminAdjustBalance,
		adminGetAnalytics,
		adminSearchUsers,
		adminGetBadgeCatalog,
		adminListBadgeReleases,
		adminCreateBadgeRelease,
		adminUpdateBadgeRelease,
		adminArchiveBadgeRelease,
		ApiClientError
	} from '$lib/api';
	import type { AnalyticsStats, UserSearchResult, BadgeCatalogEntry, AdminBadgeRelease } from '$lib/types';

	// ── Tab state ─────────────────────────────────────────────────────────
	let activeTab = $state<'operations' | 'analytics' | 'badges'>('operations');

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

	// ── Badge Releases ────────────────────────────────────────────────────
	let catalog = $state<BadgeCatalogEntry[]>([]);
	let releases = $state<AdminBadgeRelease[]>([]);
	let badgesLoading = $state(false);
	let badgesError = $state<string | null>(null);

	// Create form state
	let cfBadgeKey = $state('');
	let cfPrice = $state('');
	let cfStock = $state('');
	let cfReleasedAt = $state('');
	let cfExpiresAt = $state('');
	let cfNotes = $state('');
	let cfLoading = $state(false);
	let cfError = $state<string | null>(null);

	// Inline-edit state: map of release id → edit draft
	type EditDraft = { price: string; stock: string; expiresAt: string; active: boolean; notes: string };
	let editDrafts = $state<Record<number, EditDraft>>({});
	let editSaving = $state<Record<number, boolean>>({});
	let editError = $state<Record<number, string | null>>({});

	async function loadBadges() {
		badgesLoading = true;
		badgesError = null;
		try {
			[catalog, releases] = await Promise.all([
				adminGetBadgeCatalog(),
				adminListBadgeReleases()
			]);
		} catch (e) {
			badgesError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			badgesLoading = false;
		}
	}

	function switchToBadges() {
		activeTab = 'badges';
		if (!catalog.length && !releases.length) loadBadges();
	}

	async function createRelease() {
		cfError = null;
		const price = parseInt(cfPrice, 10);
		if (!cfBadgeKey) { cfError = 'Select a badge.'; return; }
		if (!Number.isInteger(price) || price < 0) { cfError = 'Price must be a non-negative integer.'; return; }
		const stock = cfStock.trim() ? parseInt(cfStock, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { cfError = 'Stock must be a positive integer.'; return; }
		const releasedAt = cfReleasedAt ? new Date(cfReleasedAt).toISOString() : new Date().toISOString();
		const expiresAt = cfExpiresAt ? new Date(cfExpiresAt).toISOString() : null;
		cfLoading = true;
		try {
			const created = await adminCreateBadgeRelease({
				badge_key: cfBadgeKey,
				price,
				stock,
				released_at: releasedAt,
				expires_at: expiresAt,
				notes: cfNotes.trim() || null
			});
			releases = [created, ...releases];
			cfBadgeKey = ''; cfPrice = ''; cfStock = ''; cfReleasedAt = ''; cfExpiresAt = ''; cfNotes = '';
		} catch (e) {
			cfError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			cfLoading = false;
		}
	}

	function startEdit(rel: AdminBadgeRelease) {
		editDrafts[rel.id] = {
			price: String(rel.price),
			stock: rel.stock != null ? String(rel.stock) : '',
			expiresAt: rel.expires_at ? rel.expires_at.slice(0, 16) : '',
			active: rel.active,
			notes: rel.notes ?? ''
		};
	}

	function cancelEdit(id: number) {
		const { [id]: _, ...rest } = editDrafts;
		editDrafts = rest;
		const { [id]: _e, ...eRest } = editError;
		editError = eRest;
	}

	async function saveEdit(rel: AdminBadgeRelease) {
		const d = editDrafts[rel.id];
		if (!d) return;
		editError[rel.id] = null;
		const price = parseInt(d.price, 10);
		if (!Number.isInteger(price) || price < 0) { editError[rel.id] = 'Price must be a non-negative integer.'; return; }
		const stock = d.stock.trim() ? parseInt(d.stock, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { editError[rel.id] = 'Stock must be positive.'; return; }
		editSaving[rel.id] = true;
		try {
			const updated = await adminUpdateBadgeRelease(rel.id, {
				price,
				stock,
				expires_at: d.expiresAt ? new Date(d.expiresAt).toISOString() : null,
				active: d.active,
				notes: d.notes.trim() || null
			});
			releases = releases.map((r) => (r.id === rel.id ? updated : r));
			cancelEdit(rel.id);
		} catch (e) {
			editError[rel.id] = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			editSaving[rel.id] = false;
		}
	}

	async function archiveRelease(id: number) {
		editSaving[id] = true;
		try {
			await adminArchiveBadgeRelease(id);
			releases = releases.map((r) => (r.id === id ? { ...r, active: false } : r));
			cancelEdit(id);
		} catch (e) {
			editError[id] = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			editSaving[id] = false;
		}
	}

	const TIER_LABELS: Record<number, string> = { 1: 'Common', 2: 'Uncommon', 3: 'Rare', 4: 'Epic', 5: 'Legendary' };
	const TIER_COLORS: Record<number, string> = {
		1: 'text-surface-400',
		2: 'text-green-400',
		3: 'text-blue-400',
		4: 'text-purple-400',
		5: 'text-primary-400'
	};
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
		<button
			onclick={switchToBadges}
			class="px-5 py-2.5 text-xs font-semibold uppercase tracking-widest border-b-2 -mb-px transition-colors {activeTab === 'badges'
				? 'border-primary-400 text-primary-400'
				: 'border-transparent text-surface-400 hover:text-surface-200'}"
		>
			Badges
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

	<!-- ── Badges Tab ───────────────────────────────────────────────── -->
	{#if activeTab === 'badges'}
		<div class="space-y-8">

			{#if badgesError}
				<div class="px-4 py-3 rounded bg-error-900 border border-error-700 text-error-300 text-sm">{badgesError}</div>
			{/if}

			<!-- Create Release -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl p-6">
				<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest mb-4">New Badge Release</h2>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<!-- Badge selector -->
					<div class="sm:col-span-2">
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-badge">Badge</label>
						<select id="cf-badge" bind:value={cfBadgeKey}
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none">
							<option value="">— select a badge —</option>
							{#each catalog as entry}
								<option value={entry.key}>[T{entry.tier}] {entry.title}</option>
							{/each}
						</select>
					</div>
					<!-- Price -->
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-price">Price (bUEC)</label>
						<input id="cf-price" type="number" min="0" bind:value={cfPrice}
							placeholder="e.g. 500"
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<!-- Stock -->
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-stock">Stock (blank = unlimited)</label>
						<input id="cf-stock" type="number" min="1" bind:value={cfStock}
							placeholder="e.g. 25"
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<!-- Released At -->
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-released">Release Date/Time (blank = now)</label>
						<input id="cf-released" type="datetime-local" bind:value={cfReleasedAt}
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<!-- Expires At -->
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-expires">Expiry Date/Time (blank = never)</label>
						<input id="cf-expires" type="datetime-local" bind:value={cfExpiresAt}
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<!-- Notes -->
					<div class="sm:col-span-2">
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="cf-notes">Admin Notes (optional)</label>
						<input id="cf-notes" type="text" bind:value={cfNotes}
							placeholder="e.g. CitizenCon 2026 special drop"
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
				</div>
				{#if cfError}
					<p class="text-red-400 text-sm mt-3">{cfError}</p>
				{/if}
				<div class="mt-4">
					<button onclick={createRelease} disabled={cfLoading}
						class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50">
						{cfLoading ? 'Creating…' : 'Create Release'}
					</button>
				</div>
			</section>

			<!-- Releases Table -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl overflow-hidden">
				<div class="px-6 py-4 border-b border-surface-700 flex items-center justify-between">
					<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">All Releases</h2>
					{#if badgesLoading}
						<span class="text-surface-500 text-xs">Loading…</span>
					{:else}
						<button onclick={loadBadges} class="text-xs text-primary-400 hover:text-primary-300 uppercase tracking-wider">Refresh</button>
					{/if}
				</div>

				{#if releases.length === 0 && !badgesLoading}
					<p class="text-surface-500 text-sm px-6 py-8 text-center">No releases yet. Create one above.</p>
				{:else}
					<div class="divide-y divide-surface-700">
						{#each releases as rel}
							{@const draft = editDrafts[rel.id]}
							{@const saving = editSaving[rel.id] ?? false}
							<div class="px-6 py-4 {rel.active ? '' : 'opacity-50'}">
								{#if draft}
									<!-- Edit mode -->
									<div class="space-y-3">
										<div class="flex items-center gap-3 mb-1">
											<span class="{TIER_COLORS[rel.tier] ?? 'text-surface-300'} text-xs font-bold uppercase tracking-widest">{TIER_LABELS[rel.tier] ?? `T${rel.tier}`}</span>
											<span class="text-surface-100 text-sm font-semibold">{rel.title}</span>
											<span class="text-surface-500 text-xs font-mono">{rel.badge_key}</span>
										</div>
										<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
											<div>
												<label class="block text-surface-400 text-xs mb-0.5">Price (bUEC)
													<input type="number" min="0" bind:value={draft.price}
														class="input w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
												</label>
											</div>
											<div>
												<label class="block text-surface-400 text-xs mb-0.5">Stock (blank=∞)
													<input type="number" min="1" bind:value={draft.stock}
														class="input w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
												</label>
											</div>
											<div class="sm:col-span-2">
												<label class="block text-surface-400 text-xs mb-0.5">Expires (blank=never)
													<input type="datetime-local" bind:value={draft.expiresAt}
														class="input w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
												</label>
											</div>
											<div class="sm:col-span-3">
												<label class="block text-surface-400 text-xs mb-0.5">Notes
													<input type="text" bind:value={draft.notes}
														class="input w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
												</label>
											</div>
											<div class="flex items-end pb-0.5">
												<label class="flex items-center gap-2 cursor-pointer text-sm text-surface-300">
													<input type="checkbox" bind:checked={draft.active} class="w-4 h-4 accent-primary-400" />
													Active
												</label>
											</div>
										</div>
										{#if editError[rel.id]}
											<p class="text-red-400 text-xs">{editError[rel.id]}</p>
										{/if}
										<div class="flex gap-2">
											<button onclick={() => saveEdit(rel)} disabled={saving}
												class="btn preset-filled-primary-500 text-xs uppercase tracking-wider disabled:opacity-50 px-3 py-1">
												{saving ? 'Saving…' : 'Save'}
											</button>
											<button onclick={() => cancelEdit(rel.id)} disabled={saving}
												class="btn bg-surface-700 text-surface-300 hover:bg-surface-600 text-xs uppercase tracking-wider px-3 py-1">
												Cancel
											</button>
											{#if rel.active}
												<button onclick={() => archiveRelease(rel.id)} disabled={saving}
													class="btn bg-red-900 text-red-300 hover:bg-red-800 text-xs uppercase tracking-wider px-3 py-1 ml-auto">
													Archive
												</button>
											{/if}
										</div>
									</div>
								{:else}
									<!-- View mode -->
									<div class="flex items-start justify-between gap-4">
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-2 flex-wrap mb-0.5">
												<span class="{TIER_COLORS[rel.tier] ?? 'text-surface-300'} text-xs font-bold uppercase tracking-widest">{TIER_LABELS[rel.tier] ?? `T${rel.tier}`}</span>
												<span class="text-surface-100 text-sm font-semibold">{rel.title}</span>
												<span class="text-surface-500 text-xs font-mono">{rel.badge_key}</span>
												{#if !rel.active}
													<span class="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase bg-surface-700 text-surface-500">Archived</span>
												{/if}
												{#if rel.expires_at && new Date(rel.expires_at) < new Date()}
													<span class="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase bg-red-900/60 text-red-400">Expired</span>
												{/if}
											</div>
											<div class="flex flex-wrap gap-x-4 gap-y-1 mt-1 text-xs text-surface-400">
												<span><span class="text-primary-300 font-semibold">{rel.price.toLocaleString()} bUEC</span></span>
												{#if rel.stock != null}
													<span>{rel.sold}/{rel.stock} sold</span>
												{:else}
													<span>{rel.sold} sold · unlimited</span>
												{/if}
												<span>Released {new Date(rel.released_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })}</span>
												{#if rel.expires_at}
													<span>Expires {new Date(rel.expires_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })}</span>
												{/if}
												{#if rel.notes}
													<span class="text-surface-500 italic">{rel.notes}</span>
												{/if}
											</div>
										</div>
										<button onclick={() => startEdit(rel)}
											class="shrink-0 text-xs text-surface-400 hover:text-surface-200 uppercase tracking-wider border border-surface-600 hover:border-surface-400 rounded px-2 py-1 transition-colors">
											Edit
										</button>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</section>
		</div>
	{/if}
</div>

