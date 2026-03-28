<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listPendingMarkets,
		listDeadlinePassedMarkets,
		listResolutionRequestedMarkets,
		approveMarket,
		rejectMarket,
		resolveMarket,
		denyResolutionRequest,
		listPendingReports,
		reviewReport,
		dismissReport,
		getPatches,
		markPatchNotified
	} from '$lib/api';
	import { isModerator } from '$lib/stores/auth';
	import type { Market, ResolutionRequestMarket, Report, DetectedPatch } from '$lib/types';
	import { CATEGORY_LABELS } from '$lib/categories';
	import TabBar from '$lib/components/TabBar.svelte';

	let pending = $state<(Market & { creator_name: string })[]>([]);
	let deadlinePassed = $state<(Market & { creator_name: string })[]>([]);
	let resolutionRequests = $state<ResolutionRequestMarket[]>([]);
	let reports = $state<Report[]>([]);
	let patches = $state<DetectedPatch[]>([]);
	let loading = $state(true);
	let error = $state('');
	let actionError = $state('');
	let actingId = $state<number | null>(null);
	let activeTab = $state<'review' | 'deadline_passed' | 'resolution' | 'reports'>('review');
	let keywordFilter = $state('');
	let categoryFilter = $state('all');
	let reporterFilter = $state('all');
	let selectedPending = $state<Record<number, boolean>>({});
	let bulkAction = $state<'approve' | 'reject' | null>(null);
	let bulkSummary = $state<string | null>(null);
	// Per-market evidence links for the resolve action.
	let resolveEvidence = $state<Record<number, string>>({});

	let unseenPatches = $derived(patches.filter((p) => p.notified === 0));
	let categoryOptions = $derived(
		[
			...new Set([
				...pending.map((m) => m.category),
				...deadlinePassed.map((m) => m.category),
				...resolutionRequests.map((m) => m.category),
				...reports.map((r) => r.category)
			])
		]
			.sort((a, b) => a.localeCompare(b))
	);
	let reporterOptions = $derived(
		[...new Set(reports.map((r) => r.reporter_name))].sort((a, b) => a.localeCompare(b))
	);
	let keyword = $derived(keywordFilter.trim().toLowerCase());

	function includesKeyword(parts: Array<string | null | undefined>) {
		if (!keyword) return true;
		return parts.some((p) => (p ?? '').toLowerCase().includes(keyword));
	}

	let filteredPending = $derived(
		pending.filter((market) => {
			if (categoryFilter !== 'all' && market.category !== categoryFilter) return false;
			return includesKeyword([market.title, market.description, market.resolution_criteria, market.creator_name]);
		})
	);

	let filteredResolutionRequests = $derived(
		resolutionRequests.filter((rr) => {
			if (categoryFilter !== 'all' && rr.category !== categoryFilter) return false;
			return includesKeyword([
				rr.title,
				rr.description,
				rr.resolution_criteria,
				rr.creator_name,
				rr.requester_name,
				rr.request_note,
				rr.request_link
			]);
		})
	);

	let filteredDeadlinePassed = $derived(
		deadlinePassed.filter((market) => {
			if (categoryFilter !== 'all' && market.category !== categoryFilter) return false;
			return includesKeyword([
				market.title,
				market.description,
				market.resolution_criteria,
				market.creator_name
			]);
		})
	);

	let filteredReports = $derived(
		reports.filter((report) => {
			if (categoryFilter !== 'all' && report.category !== categoryFilter) return false;
			if (reporterFilter !== 'all' && report.reporter_name !== reporterFilter) return false;
			return includesKeyword([report.market_title, report.reason, report.reporter_name]);
		})
	);

	let selectedPendingIds = $derived(
		filteredPending.map((m) => m.id).filter((id) => selectedPending[id])
	);
	let selectedPendingCount = $derived(selectedPendingIds.length);

	onMount(async () => {
		await load();
	});

	async function load() {
		loading = true;
		error = '';
		bulkSummary = null;
		try {
			[pending, deadlinePassed, resolutionRequests, reports, patches] = await Promise.all([
				listPendingMarkets(),
				listDeadlinePassedMarkets(),
				listResolutionRequestedMarkets(),
				listPendingReports(),
				getPatches()
			]);
			// Auto-switch to the non-empty tab.
			if (pending.length === 0 && deadlinePassed.length > 0) {
				activeTab = 'deadline_passed';
			} else if (pending.length === 0 && resolutionRequests.length > 0) {
				activeTab = 'resolution';
			}
			selectedPending = {};
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function withAction(id: number, fn: () => Promise<void>) {
		actingId = id;
		actionError = '';
		try {
			await fn();
			await load();
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		} finally {
			actingId = null;
		}
	}

	async function doApprove(id: number) {
		await withAction(id, () => approveMarket(id));
	}

	async function doReject(id: number) {
		await withAction(id, () => rejectMarket(id));
	}

	async function doResolve(id: number, outcomeId: number) {
		const evidence = resolveEvidence[id]?.trim() || undefined;
		await withAction(id, () => resolveMarket(id, outcomeId, evidence));
	}

	async function doDenyResolution(id: number) {
		await withAction(id, () => denyResolutionRequest(id));
	}

	async function doReviewReport(id: number) {
		await withAction(id, () => reviewReport(id));
	}

	async function doDismissReport(id: number) {
		await withAction(id, () => dismissReport(id));
	}

	async function doMarkSeen(id: number) {
		await withAction(id, () => markPatchNotified(id));
	}

	function togglePendingSelection(id: number) {
		selectedPending = { ...selectedPending, [id]: !selectedPending[id] };
	}

	function toggleAllFilteredPending() {
		const allSelected = filteredPending.length > 0 && selectedPendingCount === filteredPending.length;
		const next = { ...selectedPending };
		for (const market of filteredPending) {
			next[market.id] = !allSelected;
		}
		selectedPending = next;
	}

	async function doBulkPending(action: 'approve' | 'reject') {
		if (selectedPendingIds.length === 0) return;
		bulkAction = action;
		bulkSummary = null;
		actionError = '';

		let success = 0;
		const failed: number[] = [];
		for (const id of selectedPendingIds) {
			try {
				if (action === 'approve') {
					await approveMarket(id);
				} else {
					await rejectMarket(id);
				}
				success += 1;
			} catch {
				failed.push(id);
			}
		}

		if (failed.length === 0) {
			bulkSummary = `${action === 'approve' ? 'Approved' : 'Rejected'} ${success} market${success === 1 ? '' : 's'}.`;
		} else {
			actionError = `${action === 'approve' ? 'Approve' : 'Reject'} completed with errors. Failed IDs: ${failed.join(', ')}`;
		}

		bulkAction = null;
		await load();
	}
</script>

<svelte:head>
	<title>Mod Queue — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-10">
	<div class="mb-7">
		<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Moderation</p>
		<h1 class="text-2xl font-bold text-surface-900 tracking-tight">Mod Queue</h1>
		<p class="text-surface-500 text-sm mt-1">Review and manage markets pending approval or resolution.</p>
	</div>

	{#if !$isModerator}
		<div class="sc-card p-8 text-center">
			<p class="text-surface-600 text-sm">You must be a moderator to access this page.</p>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading queue…</div>
	{:else if error}
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-4">{error}</div>
	{:else}
		{#if actionError}
			<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-4">{actionError}</div>
		{/if}
		{#if bulkSummary}
			<div class="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm mb-4">{bulkSummary}</div>
		{/if}

		<!-- New Patch Detections -->
		<div class="mb-8">
			<div class="flex items-center gap-2 mb-3">
				<h2 class="text-sm font-bold uppercase tracking-[0.15em] text-surface-700">New Patch Detections</h2>
				{#if unseenPatches.length > 0}
					<span class="inline-flex items-center justify-center px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-xs font-bold border border-green-200">
						{unseenPatches.length} new
					</span>
				{/if}
			</div>
			{#if patches.length === 0}
				<div class="sc-card p-5 text-center text-surface-400 text-sm">No patches detected yet — scraper runs every 30 minutes.</div>
			{:else}
				<div class="space-y-2">
					{#each patches as patch}
						<div class="sc-card p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3
							{patch.notified === 0 ? 'border-l-4 border-l-green-400' : 'opacity-60'}">
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 flex-wrap">
									<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-surface-100 text-surface-700 border border-surface-200 font-mono">
										{patch.patch_version}
									</span>
									{#if patch.notified === 0}
										<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-green-100 text-green-700">new</span>
									{/if}
								</div>
								<p class="text-surface-800 text-sm font-medium mt-1">{patch.title}</p>
								<p class="text-surface-400 text-xs mt-0.5">
									Detected {new Date(patch.first_seen_at).toLocaleString()}
								</p>
							</div>
							<div class="flex items-center gap-2 shrink-0">
								<a
									href={patch.thread_url}
									target="_blank"
									rel="noopener noreferrer"
									class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider"
								>
									View Thread
								</a>
								{#if patch.notified === 0}
									<button
										onclick={() => doMarkSeen(patch.id)}
										disabled={actingId === patch.id}
										class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider"
									>
										Mark Seen
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Filters -->
		<div class="sc-card p-4 mb-6 space-y-3">
			<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
				<div>
					<label for="mod-filter-keyword" class="block text-[10px] font-bold uppercase tracking-[0.15em] text-surface-500 mb-1">Keyword</label>
					<input
						id="mod-filter-keyword"
						type="text"
						bind:value={keywordFilter}
						placeholder="Search title, criteria, notes, reason..."
						class="sc-input text-sm"
					/>
				</div>
				<div>
					<label for="mod-filter-category" class="block text-[10px] font-bold uppercase tracking-[0.15em] text-surface-500 mb-1">Category</label>
					<select id="mod-filter-category" bind:value={categoryFilter} class="sc-input text-sm">
						<option value="all">All categories</option>
						{#each categoryOptions as cat}
							<option value={cat}>{CATEGORY_LABELS[cat] ?? cat}</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="mod-filter-reporter" class="block text-[10px] font-bold uppercase tracking-[0.15em] text-surface-500 mb-1">Reporter (Reports tab)</label>
					<select id="mod-filter-reporter" bind:value={reporterFilter} class="sc-input text-sm">
						<option value="all">All reporters</option>
						{#each reporterOptions as reporter}
							<option value={reporter}>{reporter}</option>
						{/each}
					</select>
				</div>
			</div>
			<div class="flex items-center justify-between text-xs text-surface-500">
				<span>
					Pending: {filteredPending.length}/{pending.length} ·
					Deadline Passed: {filteredDeadlinePassed.length}/{deadlinePassed.length} ·
					Resolution: {filteredResolutionRequests.length}/{resolutionRequests.length} ·
					Reports: {filteredReports.length}/{reports.length}
				</span>
				<button
					onclick={() => {
						keywordFilter = '';
						categoryFilter = 'all';
						reporterFilter = 'all';
					}}
					class="text-primary-600 hover:text-primary-700 font-semibold uppercase tracking-wider"
				>
					Reset
				</button>
			</div>
		</div>

		<!-- Tabs -->
		<div class="mb-6">
			<TabBar
				tabs={[
					{ id: 'review', label: 'Pending Review', badge: pending.length || undefined },
					{ id: 'deadline_passed', label: 'Deadline Passed', badge: deadlinePassed.length || undefined },
					{ id: 'resolution', label: 'Resolution Requests', badge: resolutionRequests.length || undefined },
					{ id: 'reports', label: 'Reports', badge: reports.length || undefined }
				]}
				bind:active={activeTab}
			/>
		</div>

		<!-- Tab: Pending Review -->
		{#if activeTab === 'review'}
			{#if pending.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No markets pending review.</div>
			{:else if filteredPending.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No pending markets match the current filters.</div>
			{:else}
				<div class="mb-3 sc-card p-3 flex flex-wrap items-center justify-between gap-3">
					<div class="flex items-center gap-3">
						<label class="inline-flex items-center gap-2 text-xs text-surface-600">
							<input
								type="checkbox"
								checked={filteredPending.length > 0 && selectedPendingCount === filteredPending.length}
								onchange={toggleAllFilteredPending}
							/>
							Select all filtered
						</label>
						<span class="text-xs text-surface-500">{selectedPendingCount} selected</span>
					</div>
					<div class="flex items-center gap-2">
						<button
							onclick={() => doBulkPending('approve')}
							disabled={selectedPendingCount === 0 || bulkAction !== null || actingId !== null}
							class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider disabled:opacity-50"
						>
							{bulkAction === 'approve' ? 'Approving…' : 'Bulk Approve'}
						</button>
						<button
							onclick={() => doBulkPending('reject')}
							disabled={selectedPendingCount === 0 || bulkAction !== null || actingId !== null}
							class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider disabled:opacity-50"
						>
							{bulkAction === 'reject' ? 'Rejecting…' : 'Bulk Reject'}
						</button>
					</div>
				</div>

				<div class="space-y-3">
					{#each filteredPending as market}
						<div class="sc-card p-5">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-2 flex-wrap">
										<input
											type="checkbox"
											checked={Boolean(selectedPending[market.id])}
											onchange={() => togglePendingSelection(market.id)}
										/>
										<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
									</div>
									<h2 class="text-surface-800 font-semibold text-base">{market.title}</h2>
									{#if market.description}
										<p class="text-surface-600 text-sm mt-1 line-clamp-2">{market.description}</p>
									{/if}
									{#if market.resolution_criteria}
										<p class="text-surface-500 text-xs mt-1 italic">Criteria: {market.resolution_criteria}</p>
									{/if}
									<p class="text-surface-400 text-xs mt-2">
										{market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
									</p>
									{#if (market.auto_filter_matches?.length ?? 0) > 0}
										<div class="mt-2 flex flex-wrap items-center gap-1.5">
											<span class="text-[10px] font-bold uppercase tracking-wider text-amber-700">Auto-filter matches:</span>
											{#each market.auto_filter_matches ?? [] as match}
												<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-100 text-amber-800 border border-amber-200">{match}</span>
											{/each}
										</div>
									{:else}
										<p class="text-[11px] text-green-700 mt-2">Auto-filter: clean</p>
									{/if}
								</div>
								<div class="flex flex-col gap-2 shrink-0">
									<button
										onclick={() => doApprove(market.id)}
										disabled={actingId === market.id}
										class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider"
									>
										Approve
									</button>
									<button
										onclick={() => doReject(market.id)}
										disabled={actingId === market.id}
										class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider"
									>
										Reject
									</button>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<!-- Tab: Deadline Passed -->
		{#if activeTab === 'deadline_passed'}
			{#if deadlinePassed.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No deadline-passed markets awaiting resolution.</div>
			{:else if filteredDeadlinePassed.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No deadline-passed markets match the current filters.</div>
			{:else}
				<div class="space-y-3">
					{#each filteredDeadlinePassed as market}
						<div class="sc-card p-5 border-l-4 border-l-amber-500 bg-amber-50/20">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-2 flex-wrap">
										<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
										<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Deadline Passed</span>
									</div>
									<h2 class="text-surface-800 font-semibold text-base">
										<a href="/markets/{market.id}" class="hover:text-primary-600 transition-colors">{market.title}</a>
									</h2>
									{#if market.description}
										<p class="text-surface-600 text-sm mt-1 line-clamp-2">{market.description}</p>
									{/if}
									{#if market.resolution_criteria}
										<p class="text-surface-500 text-xs mt-1 italic">Criteria: {market.resolution_criteria}</p>
									{/if}
									<p class="text-surface-400 text-xs mt-2">
										{market.creator_name} · closed {new Date(market.resolution_deadline).toLocaleDateString()}
									</p>
								</div>

								<div class="flex flex-col gap-2 shrink-0">
									<input
										type="url"
										bind:value={resolveEvidence[market.id]}
										placeholder="Evidence link (optional)"
										class="sc-input text-xs w-44"
									/>
									{#each (market.outcomes ?? []) as outcome, oi}
										<button
											onclick={() => doResolve(market.id, outcome.id)}
											disabled={actingId === market.id}
											class="btn btn-sm text-white text-xs uppercase tracking-wider {oi === 0 ? 'bg-green-600 hover:bg-green-700' : oi === 1 ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'}"
										>
											Resolve {outcome.label}
										</button>
									{/each}
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<!-- Tab: Reports -->
		{#if activeTab === 'reports'}
			{#if reports.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No pending reports.</div>
			{:else if filteredReports.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No reports match the current filters.</div>
			{:else}
				<div class="space-y-3">
					{#each filteredReports as report}
						<div class="sc-card p-5 border-l-4 border-l-red-400">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<p class="text-surface-500 text-xs mb-1">
										Reported by <span class="font-semibold text-surface-700">{report.reporter_name}</span>
										&middot; {new Date(report.created_at).toLocaleString()}
									</p>
									<p class="text-surface-400 text-[11px] mb-1 uppercase tracking-wider">
										{CATEGORY_LABELS[report.category] ?? report.category}
									</p>
									<h2 class="text-surface-800 font-semibold text-base">
										<a href="/markets/{report.market_id}" class="hover:text-primary-600 transition-colors">{report.market_title}</a>
									</h2>
									<p class="text-surface-600 text-sm mt-2 bg-surface-50 border border-surface-200 rounded p-2 italic">
										&ldquo;{report.reason}&rdquo;
									</p>
								</div>
								<div class="flex flex-col gap-2 shrink-0">
									<button
										onclick={() => doReviewReport(report.id)}
										disabled={actingId === report.id}
										class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider"
										title="Mark as reviewed — take action on the market separately if needed"
									>
										Reviewed
									</button>
									<button
										onclick={() => doDismissReport(report.id)}
										disabled={actingId === report.id}
										class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider"
									>
										Dismiss
									</button>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<!-- Tab: Resolution Requests -->
		{#if activeTab === 'resolution'}
			{#if resolutionRequests.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No resolution requests.</div>
			{:else if filteredResolutionRequests.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No resolution requests match the current filters.</div>
			{:else}
				<div class="space-y-3">
					{#each filteredResolutionRequests as rr}
						<div class="sc-card p-5 border-l-4 border-l-amber-400 bg-amber-50/20">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-2">
										<span class="sc-tag">{CATEGORY_LABELS[rr.category] ?? rr.category}</span>
										<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Resolution Requested</span>
									</div>
									<h2 class="text-surface-800 font-semibold text-base">
										<a href="/markets/{rr.id}" class="hover:text-primary-600 transition-colors">{rr.title}</a>
									</h2>
									{#if rr.resolution_criteria}
										<p class="text-surface-500 text-xs mt-1 italic">Criteria: {rr.resolution_criteria}</p>
									{/if}
									<p class="text-surface-400 text-xs mt-1">
										Created by {rr.creator_name} · closes {new Date(rr.resolution_deadline).toLocaleDateString()}
									</p>

									<!-- Requester details -->
									<div class="mt-3 pt-3 border-t border-amber-100 space-y-1.5">
										<p class="text-xs text-surface-600">
											<span class="font-semibold">Requested by:</span> {rr.requester_name}
											· {new Date(rr.requested_at).toLocaleString()}
										</p>
										{#if rr.request_link}
											<p class="text-xs">
												<span class="font-semibold text-surface-600">Evidence:</span>
												<a href={rr.request_link} target="_blank" rel="noopener noreferrer"
													class="text-primary-600 hover:underline break-all ml-1">{rr.request_link}</a>
											</p>
										{/if}
										{#if rr.request_note}
											<p class="text-xs text-surface-600">
												<span class="font-semibold">Note:</span>
												<span class="ml-1 italic">{rr.request_note}</span>
											</p>
										{/if}
									</div>
								</div>

								<div class="flex flex-col gap-2 shrink-0">
									<input
										type="url"
										bind:value={resolveEvidence[rr.id]}
										placeholder="Evidence link (optional)"
										class="sc-input text-xs w-44"
									/>
									{#each (rr.outcomes ?? []) as outcome, oi}
										<button
											onclick={() => doResolve(rr.id, outcome.id)}
											disabled={actingId === rr.id}
											class="btn btn-sm text-white text-xs uppercase tracking-wider {oi === 0 ? 'bg-green-600 hover:bg-green-700' : oi === 1 ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'}"
										>
											Resolve {outcome.label}
										</button>
									{/each}
									<button
										onclick={() => doDenyResolution(rr.id)}
										disabled={actingId === rr.id}
										class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider"
									>
										Deny
									</button>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		{/if}
	{/if}
</div>


