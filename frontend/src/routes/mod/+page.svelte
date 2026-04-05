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
	import Alert from '$lib/components/Alert.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import ModPatchesPanel from '$lib/components/mod/ModPatchesPanel.svelte';
	import ModPendingReviewPanel from '$lib/components/mod/ModPendingReviewPanel.svelte';
	import ModDeadlinePassedPanel from '$lib/components/mod/ModDeadlinePassedPanel.svelte';
	import ModReportsPanel from '$lib/components/mod/ModReportsPanel.svelte';
	import ModResolutionRequestsPanel from '$lib/components/mod/ModResolutionRequestsPanel.svelte';
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
		<EmptyState message="You must be a moderator to access this page." />
	{:else if loading}
		<EmptyState message="Loading queue…" card={false} padding="py-16" />
	{:else if error}
		<Alert type="error" message={error} />
	{:else}
		{#if actionError}
			<Alert type="error" message={actionError} />
		{/if}
		{#if bulkSummary}
			<Alert type="success" message={bulkSummary} />
		{/if}

		<ModPatchesPanel patches={patches} {unseenPatches} {actingId} onMarkSeen={doMarkSeen} />

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
			<ModPendingReviewPanel
				{pending}
				{filteredPending}
				{selectedPending}
				{selectedPendingCount}
				{actingId}
				{bulkAction}
				onTogglePendingSelection={togglePendingSelection}
				onToggleAllFilteredPending={toggleAllFilteredPending}
				onApprove={doApprove}
				onReject={doReject}
				onBulkPending={doBulkPending}
			/>
		{/if}

		<!-- Tab: Deadline Passed -->
		{#if activeTab === 'deadline_passed'}
			<ModDeadlinePassedPanel
				{deadlinePassed}
				{filteredDeadlinePassed}
				bind:resolveEvidence
				{actingId}
				onResolve={doResolve}
			/>
		{/if}

		<!-- Tab: Reports -->
		{#if activeTab === 'reports'}
			<ModReportsPanel
				{reports}
				{filteredReports}
				{actingId}
				onReview={doReviewReport}
				onDismiss={doDismissReport}
			/>
		{/if}

		<!-- Tab: Resolution Requests -->
		{#if activeTab === 'resolution'}
			<ModResolutionRequestsPanel
				resolutionRequests={resolutionRequests}
				{filteredResolutionRequests}
				bind:resolveEvidence
				{actingId}
				onResolve={doResolve}
				onDeny={doDenyResolution}
			/>
		{/if}
	{/if}
</div>


