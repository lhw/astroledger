<script lang="ts">
	import {
		adminTriggerWeeklyPayout,
		adminAdjustBalance,
		adminGetAnalytics,
		adminSearchUsers,
		adminBanUser,
		adminShadowBanUser,
		adminGetBadgeCatalog,
		adminListBadgeReleases,
		adminCreateBadgeRelease,
		adminUpdateBadgeRelease,
		adminArchiveBadgeRelease,
		adminListBadgeDefinitions,
		adminCreateBadgeDefinition,
		adminUpdateBadgeDefinition,
		ApiClientError
	} from '$lib/api';
	import type { AnalyticsStats, AnalyticsStat, UserSearchResult, BadgeCatalogEntry, AdminBadgeRelease, AdminBadgeDefinition } from '$lib/types';
	import { authReady, isAdmin } from '$lib/stores/auth';
	import BadgeReleaseCalendar from '$lib/components/BadgeReleaseCalendar.svelte';
	import TabBar from '$lib/components/TabBar.svelte';

	// ── Tab state ─────────────────────────────────────────────────────────
	let activeTab = $state<'operations' | 'analytics' | 'badges' | 'defs'>('operations');

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
	let adjIsBanned = $state(false);
	let adjIsShadowBanned = $state(false);
	let adjBanLoading = $state(false);
	let adjShadowBanLoading = $state(false);
	let adjModError = $state<string | null>(null);

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
		adjIsBanned = u.is_banned === 1;
		adjIsShadowBanned = u.is_shadow_banned === 1;
		adjSearchQuery = '';
		adjSearchResults = [];
		adjModError = null;
	}

	function clearSelectedUser() {
		adjSelectedUser = null;
		adjAmount = '';
		adjReason = '';
		adjResult = null;
		adjError = null;
		adjModError = null;
		adjIsBanned = false;
		adjIsShadowBanned = false;
	}

	async function toggleBan() {
		if (!adjSelectedUser) return;
		adjBanLoading = true;
		adjModError = null;
		try {
			await adminBanUser(adjSelectedUser.id, !adjIsBanned);
			adjIsBanned = !adjIsBanned;
		} catch (err) {
			adjModError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjBanLoading = false;
		}
	}

	async function toggleShadowBan() {
		if (!adjSelectedUser) return;
		adjShadowBanLoading = true;
		adjModError = null;
		try {
			await adminShadowBanUser(adjSelectedUser.id, !adjIsShadowBanned);
			adjIsShadowBanned = !adjIsShadowBanned;
		} catch (err) {
			adjModError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjShadowBanLoading = false;
		}
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

	function statBarWidth(item: AnalyticsStat, list: AnalyticsStat[]) {
		const max = list[0]?.count || 1;
		return Math.round((item.count / max) * 100);
	}

	function statPanels(data: AnalyticsStats): { label: string; items: AnalyticsStat[] }[] {
		return [
			{ label: 'Browsers', items: data.browsers ?? [] },
			{ label: 'Operating Systems', items: data.systems ?? [] },
			{ label: 'Locations', items: data.locations ?? [] },
			{ label: 'Languages', items: data.languages ?? [] }
		];
	}

	function defaultBadgeSymbol(tier: number) {
		if (tier >= 5) return '★';
		if (tier === 4) return '◈';
		if (tier === 3) return '◆';
		if (tier === 2) return '●';
		return '▲';
	}

	function badgeDefIcon(def: AdminBadgeDefinition) {
		return def.icon.trim() || defaultBadgeSymbol(def.tier);
	}

	function insuranceLabel(insurance: string) {
		if (insurance === '6w') return '6 Weeks';
		if (insurance === '120w') return '120 Weeks';
		if (insurance === 'lti') return 'LTI';
		return '—';
	}

	// ── Badge Releases ────────────────────────────────────────────────────
	let catalog = $state<BadgeCatalogEntry[]>([]);
	let releases = $state<AdminBadgeRelease[]>([]);
	let badgesLoading = $state(false);
	let badgesError = $state<string | null>(null);

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

	// API wrappers forwarded into BadgeReleaseCalendar
	async function onBadgeCreate(body: Parameters<typeof adminCreateBadgeRelease>[0]) {
		return adminCreateBadgeRelease(body);
	}
	async function onBadgeUpdate(id: number, body: Parameters<typeof adminUpdateBadgeRelease>[1]) {
		return adminUpdateBadgeRelease(id, body);
	}
	async function onBadgeArchive(id: number) {
		await adminArchiveBadgeRelease(id);
	}

	// ── Badge Definitions ─────────────────────────────────────────────────
	let badgeDefs = $state<AdminBadgeDefinition[]>([]);
	let defsLoading = $state(false);
	let defsError = $state<string | null>(null);

	// New definition form
	let newDefKey = $state('');
	let newDefTitle = $state('');
	let newDefDesc = $state('');
	let newDefTier = $state(1);
	let newDefIcon = $state('');
	let newDefInsurance = $state('');
	let newDefSaving = $state(false);
	let newDefError = $state<string | null>(null);

	// Per-row edit state
	let editingDefKey = $state<string | null>(null);
	let editDefTitle = $state('');
	let editDefDesc = $state('');
	let editDefTier = $state(1);
	let editDefIcon = $state('');
	let editDefInsurance = $state('');
	let editDefSaving = $state(false);
	let editDefError = $state<string | null>(null);

	async function loadDefs() {
		defsLoading = true;
		defsError = null;
		try {
			badgeDefs = await adminListBadgeDefinitions();
		} catch (e) {
			defsError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			defsLoading = false;
		}
	}

	function switchToDefs() {
		activeTab = 'defs';
		if (!badgeDefs.length) loadDefs();
	}

	async function submitNewDef() {
		newDefError = null;
		const key = newDefKey.trim();
		const title = newDefTitle.trim();
		const desc = newDefDesc.trim();
		const icon = newDefIcon.trim();
		if (!key || !title) { newDefError = 'Key and title are required.'; return; }
		if (!/^[a-z0-9_-]+$/.test(key)) { newDefError = 'Key must be lowercase letters, digits, underscores, or hyphens.'; return; }
		newDefSaving = true;
		try {
			const def = await adminCreateBadgeDefinition({ key, title, description: desc, tier: newDefTier, icon, insurance: newDefInsurance });
			badgeDefs = [...badgeDefs, def];
			newDefKey = ''; newDefTitle = ''; newDefDesc = ''; newDefTier = 1; newDefIcon = ''; newDefInsurance = '';
		} catch (e) {
			newDefError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			newDefSaving = false;
		}
	}

	function startEditDef(def: AdminBadgeDefinition) {
		editingDefKey = def.key;
		editDefTitle = def.title;
		editDefDesc = def.description;
		editDefTier = def.tier;
		editDefIcon = def.icon;
		editDefInsurance = def.insurance;
		editDefError = null;
	}

	async function saveEditDef(key: string) {
		editDefError = null;
		const title = editDefTitle.trim();
		if (!title) { editDefError = 'Title is required.'; return; }
		editDefSaving = true;
		try {
			const updated = await adminUpdateBadgeDefinition(key, {
				title,
				description: editDefDesc.trim(),
				tier: editDefTier,
				icon: editDefIcon.trim(),
				insurance: editDefInsurance
			});
			badgeDefs = badgeDefs.map(d => d.key === key ? updated : d);
			editingDefKey = null;
		} catch (e) {
			editDefError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			editDefSaving = false;
		}
	}
</script>

<svelte:head>
	<title>Admin Panel — AstroLedger</title>
</svelte:head>

<div class="admin-shell max-w-4xl mx-auto px-6 py-10">
	{#if !$authReady}
		<p class="text-surface-400 text-sm">Checking authentication…</p>
	{:else if !$isAdmin}
		<div class="bg-surface-800 border border-surface-700 rounded-xl p-8 text-center space-y-3">
			<p class="text-red-400 text-sm font-semibold">Access denied.</p>
			<p class="text-surface-500 text-xs">This page is restricted to administrators.</p>
		</div>
	{:else}
	<h1 class="text-2xl font-bold text-primary-400 tracking-widest uppercase mb-6">Admin Panel</h1>

	<div class="mb-8">
		<TabBar
			tabs={[
				{ id: 'operations', label: 'Operations' },
				{ id: 'analytics', label: 'Analytics' },
				{ id: 'badges', label: 'Badges' },
				{ id: 'defs', label: 'Badge Defs' }
			]}
			bind:active={activeTab}
			onTabChange={(id) => {
				if (id === 'analytics') switchToAnalytics();
				else if (id === 'badges') switchToBadges();
				else if (id === 'defs') switchToDefs();
			}}
		/>
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

			<!-- User Management -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 space-y-4">
				<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">
					User Management
				</h2>
				<p class="text-surface-400 text-sm leading-relaxed">
					Search for a user to adjust their balance, ban them from logging in, or shadow-ban them to silently hide all their comments.
				</p>
				<div class="space-y-3">
					<!-- User search / selected user -->
					{#if adjSelectedUser}
						<!-- Selected user chip -->
					<div class="bg-surface-700 border border-primary-600 rounded-lg px-3 py-2.5 space-y-2.5">
						<div class="flex items-center justify-between gap-3">
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
						<!-- Moderation actions -->
						<div class="flex flex-wrap items-center gap-2 pt-1 border-t border-surface-600">
							<span class="text-surface-400 text-xs uppercase tracking-wide">Moderation:</span>
							{#if adjIsBanned}
								<span class="text-xs font-semibold text-red-300 bg-red-900/40 border border-red-700 rounded px-2 py-0.5">Banned</span>
							{/if}
							{#if adjIsShadowBanned}
								<span class="text-xs font-semibold text-amber-300 bg-amber-900/40 border border-amber-700 rounded px-2 py-0.5">Shadow Banned</span>
							{/if}
							{#if !adjIsBanned && !adjIsShadowBanned}
								<span class="text-xs text-surface-500">No restrictions</span>
							{/if}
							<div class="flex gap-2 ml-auto">
								<button
									onclick={toggleBan}
									disabled={adjBanLoading || adjShadowBanLoading}
									class="text-xs px-2.5 py-1 rounded font-semibold uppercase tracking-wide transition-colors disabled:opacity-50 {adjIsBanned ? 'bg-green-800/60 hover:bg-green-700/60 text-green-300 border border-green-700' : 'bg-red-900/60 hover:bg-red-800/60 text-red-300 border border-red-700'}"
								>
									{adjBanLoading ? '…' : adjIsBanned ? 'Unban' : 'Ban'}
								</button>
								<button
									onclick={toggleShadowBan}
									disabled={adjBanLoading || adjShadowBanLoading}
									class="text-xs px-2.5 py-1 rounded font-semibold uppercase tracking-wide transition-colors disabled:opacity-50 {adjIsShadowBanned ? 'bg-surface-700 hover:bg-surface-600 text-surface-300 border border-surface-500' : 'bg-amber-900/60 hover:bg-amber-800/60 text-amber-300 border border-amber-700'}"
								>
									{adjShadowBanLoading ? '…' : adjIsShadowBanned ? 'Un-shadow-ban' : 'Shadow Ban'}
								</button>
							</div>
						</div>
						{#if adjModError}
							<p class="text-red-400 text-xs">Error: {adjModError}</p>
						{/if}
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
									opacity="0.75"
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

				<!-- ── Browsers / OS / Locations / Languages ─────────────── -->
				<div class="grid grid-cols-2 gap-4">
					{#each statPanels(analyticsData) as panel}
						<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
							<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">{panel.label}</p>
							{#if panel.items?.length}
								<div class="space-y-2">
									{#each panel.items as item}
										<div class="flex items-center justify-between gap-3">
											<span class="text-surface-300 text-xs truncate flex-1" title={item.name}>{item.name}</span>
											<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{item.count.toLocaleString()}</span>
										</div>
										<div class="w-full bg-surface-700 rounded-full h-0.5">
											<div class="bg-primary-500 h-0.5 rounded-full" style="width: {statBarWidth(item, panel.items)}%"></div>
										</div>
									{/each}
								</div>
							{:else}
								<p class="text-surface-500 text-xs">No data yet.</p>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- ── Badges Tab ───────────────────────────────────────────────── -->
	{#if activeTab === 'badges'}
		{#if badgesError}
			<div class="mb-4 px-4 py-3 rounded bg-red-950 border border-red-800 text-red-300 text-sm">{badgesError}</div>
		{:else if badgesLoading}
			<p class="text-surface-500 text-sm">Loading…</p>
		{:else}
			<BadgeReleaseCalendar
				{catalog}
				bind:releases
				onCreate={onBadgeCreate}
				onUpdate={onBadgeUpdate}
				onArchive={onBadgeArchive}
			/>
		{/if}
	{/if}

	<!-- ── Badge Definitions Tab ─────────────────────────────────────── -->
	{#if activeTab === 'defs'}
		{#if defsError}
			<div class="mb-4 px-4 py-3 rounded bg-red-950 border border-red-800 text-red-300 text-sm">{defsError}</div>
		{:else if defsLoading}
			<p class="text-surface-500 text-sm">Loading…</p>
		{:else}
			<!-- ── Create new definition ──────────────────────────────── -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 mb-6 space-y-4">
				<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">New Badge Definition</h2>
				<div class="grid grid-cols-2 gap-3">
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-key">Key <span class="text-red-400">*</span></label>
						<input id="nd-key" type="text" bind:value={newDefKey} placeholder="e.g. explorer_badge" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none font-mono" />
					</div>
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-title">Title <span class="text-red-400">*</span></label>
						<input id="nd-title" type="text" bind:value={newDefTitle} placeholder="Explorer" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<div class="col-span-2">
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-desc">Description</label>
						<input id="nd-desc" type="text" bind:value={newDefDesc} placeholder="Awarded for exploring the unknown" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-tier">Tier (1–5)</label>
						<input id="nd-tier" type="number" min="1" max="5" bind:value={newDefTier} class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
					<div>
						<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-icon">Icon (emoji or short text)</label>
						<input id="nd-icon" type="text" bind:value={newDefIcon} placeholder="🔭" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
					</div>
				</div>
				{#if newDefError}<p class="text-red-400 text-xs">{newDefError}</p>{/if}
				<button onclick={submitNewDef} disabled={newDefSaving} class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50">
					{newDefSaving ? 'Creating…' : 'Create Badge Definition'}
				</button>
			</section>

			<!-- ── Definitions table ──────────────────────────────────── -->
			<section class="bg-surface-800 border border-surface-700 rounded-xl overflow-hidden">
				<table class="w-full table-fixed text-left admin-defs-table">
					<thead class="bg-surface-700">
						<tr class="text-surface-400 text-xs uppercase tracking-widest">
							<th class="px-4 py-3 w-[18%]">Key</th>
							<th class="px-4 py-3 w-[18%]">Title</th>
							<th class="px-4 py-3 w-[8%]">Tier</th>
							<th class="px-4 py-3 w-[28%]">Description</th>
							<th class="px-4 py-3 w-[10%] text-center">Icon</th>
							<th class="px-4 py-3 w-[12%]">Insurance</th>
							<th class="px-4 py-3 w-[10%]"></th>
						</tr>
					</thead>
					<tbody>
						{#each badgeDefs as def}
							<tr class="border-t border-surface-700 hover:bg-surface-700/40 transition-colors">
								{#if editingDefKey === def.key}
									<!-- Edit row -->
									<td class="px-4 py-3 font-mono text-xs text-surface-400 break-all align-top">{def.key}</td>
									<td class="px-4 py-3">
										<input type="text" bind:value={editDefTitle} class="input w-full bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
									</td>
									<td class="px-4 py-3 align-top">
										<input type="number" min="1" max="5" bind:value={editDefTier} class="input w-16 bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
									</td>
									<td class="px-4 py-3">
										<input type="text" bind:value={editDefDesc} class="input w-full bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
									</td>
									<td class="px-4 py-3 align-top">
										<input type="text" bind:value={editDefIcon} class="input w-16 bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
										<p class="text-surface-500 text-[10px] mt-1 text-center">{defaultBadgeSymbol(editDefTier)}</p>
									</td>
									<td class="px-4 py-3 align-top">
										<select bind:value={editDefInsurance} class="bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none">
											<option value="">None</option>
											<option value="6w">6 Weeks</option>
											<option value="120w">120 Weeks</option>
											<option value="lti">LTI</option>
										</select>
									</td>
									<td class="px-4 py-3 align-top">
										<div class="flex gap-2 items-center">
											{#if editDefError}<span class="text-red-400 text-xs">{editDefError}</span>{/if}
											<button onclick={() => saveEditDef(def.key)} disabled={editDefSaving} class="btn preset-filled-primary-500 text-xs py-1 px-3 disabled:opacity-50">{editDefSaving ? '…' : 'Save'}</button>
											<button onclick={() => { editingDefKey = null; }} class="text-surface-500 hover:text-surface-200 text-xs">Cancel</button>
										</div>
									</td>
								{:else}
									<!-- View row -->
									<td class="px-4 py-3 font-mono text-xs text-surface-400 break-all align-top">{def.key}</td>
									<td class="px-4 py-3 text-surface-100 text-sm font-medium align-top">{def.title}</td>
									<td class="px-4 py-3 align-top">
										<span class="def-tier-pip tier-{def.tier}">{def.tier}</span>
									</td>
									<td class="px-4 py-3 text-surface-400 text-xs align-top">
										<div class="line-clamp-2 break-words" title={def.description}>{def.description || '—'}</div>
									</td>
									<td class="px-4 py-3 text-center align-top">
										<span class="def-icon-chip" title={def.icon.trim() ? 'Custom icon' : 'Tier default icon'}>{badgeDefIcon(def)}</span>
									</td>
									<td class="px-4 py-3 align-top">
										{#if def.insurance}
											<span class="text-xs px-2 py-0.5 rounded-full bg-amber-900/40 border border-amber-800/50 text-amber-300">
												{insuranceLabel(def.insurance)}
											</span>
										{:else}
											<span class="text-surface-600 text-xs">—</span>
										{/if}
									</td>
									<td class="px-4 py-3 align-top text-right">
										{#if def.purchasable}
											<button onclick={() => startEditDef(def)} class="text-xs text-primary-400 hover:text-primary-200 font-semibold uppercase tracking-wide">Edit</button>
										{/if}
									</td>
								{/if}
							</tr>
						{/each}
					</tbody>
				</table>
			</section>
		{/if}
	{/if}
	{/if}
</div>

<style>
	/* Tier pill in badge defs table */
	.def-tier-pip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		border-radius: 50%;
		font-size: 0.65rem;
		font-weight: 700;
	}
	.def-tier-pip.tier-1 { background: #f5ede0; color: #b08050; border: 1.5px solid #d4b896; }
	.def-tier-pip.tier-2 { background: #fde68a; color: #78350f; border: 1.5px solid #f59e0b; }
	.def-tier-pip.tier-3 { background: linear-gradient(135deg, #fde047, #ea580c); color: #fff; border: 1.5px solid #f59e0b; }
	.def-tier-pip.tier-4 { background: #fbbf24; color: #1c1008; border: 1.5px solid #fde68a; }
	.def-tier-pip.tier-5 { background: conic-gradient(from 0deg, #ff0080, #ff8c00, #ffd700, #00ff88, #ff0080); color: #fff; border: 1.5px solid rgba(255,255,255,0.5); }
	.def-icon-chip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 9999px;
		background: color-mix(in oklch, var(--color-primary-100) 70%, white 30%);
		border: 1px solid color-mix(in oklch, var(--color-primary-300) 60%, var(--color-surface-300) 40%);
		color: var(--color-surface-900);
		font-size: 1rem;
		line-height: 1;
	}
	.admin-defs-table td,
	.admin-defs-table th {
		vertical-align: top;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .bg-surface-800) {
		background: var(--card-bg) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .bg-surface-700) {
		background: color-mix(in oklch, var(--color-surface-50) 84%, var(--color-primary-100) 16%) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .border-surface-700),
	:global(:root:not([data-theme='dark']) .admin-shell .border-surface-600) {
		border-color: var(--color-surface-300) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-100) {
		color: var(--color-surface-900) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-300) {
		color: var(--color-surface-700) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-400) {
		color: var(--color-surface-500) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-500) {
		color: var(--color-surface-600) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .hover\:bg-surface-700:hover) {
		background: color-mix(in oklch, var(--color-surface-100) 85%, var(--color-primary-100) 15%) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-primary-300),
	:global(:root:not([data-theme='dark']) .admin-shell .text-primary-400) {
		color: var(--color-primary-700) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .hover\:text-surface-100:hover) {
		color: var(--color-surface-900) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .bg-surface-800) {
		background: var(--card-bg) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .bg-surface-700) {
		background: color-mix(in oklch, var(--card-bg) 84%, var(--color-surface-300) 16%) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .border-surface-700),
	:global(:root[data-theme='dark'] .admin-shell .border-surface-600) {
		border-color: var(--color-surface-300) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-300) {
		color: var(--color-surface-700) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-100) {
		color: var(--color-surface-900) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-400),
	:global(:root[data-theme='dark'] .admin-shell .text-surface-500) {
		color: var(--color-surface-600) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-primary-300),
	:global(:root[data-theme='dark'] .admin-shell .text-primary-400) {
		color: var(--color-primary-500) !important;
	}
</style>


