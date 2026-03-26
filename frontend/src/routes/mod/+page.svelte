<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listPendingMarkets,
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

	let pending = $state<(Market & { creator_name: string })[]>([]);
	let resolutionRequests = $state<ResolutionRequestMarket[]>([]);
	let reports = $state<Report[]>([]);
	let patches = $state<DetectedPatch[]>([]);
	let loading = $state(true);
	let error = $state('');
	let actionError = $state('');
	let actingId = $state<number | null>(null);
	let activeTab = $state<'review' | 'resolution' | 'reports'>('review');
	// Per-market evidence links for the resolve action.
	let resolveEvidence = $state<Record<number, string>>({});

	let unseenPatches = $derived(patches.filter((p) => p.notified === 0));

	onMount(async () => {
		await load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			[pending, resolutionRequests, reports, patches] = await Promise.all([
				listPendingMarkets(),
				listResolutionRequestedMarkets(),
				listPendingReports(),
				getPatches()
			]);
			// Auto-switch to the non-empty tab.
			if (pending.length === 0 && resolutionRequests.length > 0) {
				activeTab = 'resolution';
			}
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

	async function doResolve(id: number, resolution: 'yes' | 'no') {
		const evidence = resolveEvidence[id]?.trim() || undefined;
		await withAction(id, () => resolveMarket(id, resolution, evidence));
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
</script>

<svelte:head>
	<title>Mod Queue — ScolyMarket</title>
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

		<!-- Tabs -->
		<div class="flex border-b border-surface-200 mb-6">
			<button
				onclick={() => (activeTab = 'review')}
				class="px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] border-b-2 transition-colors
					{activeTab === 'review'
						? 'border-primary-500 text-primary-700'
						: 'border-transparent text-surface-500 hover:text-surface-700 hover:border-surface-300'}"
			>
				Pending Review
				{#if pending.length > 0}
					<span class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full bg-primary-100 text-primary-700 text-[10px] font-bold">{pending.length}</span>
				{/if}
			</button>
			<button
				onclick={() => (activeTab = 'resolution')}
				class="px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] border-b-2 transition-colors
					{activeTab === 'resolution'
						? 'border-amber-500 text-amber-700'
						: 'border-transparent text-surface-500 hover:text-surface-700 hover:border-surface-300'}"
			>
				Resolution Requests
				{#if resolutionRequests.length > 0}
					<span class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full bg-amber-100 text-amber-700 text-[10px] font-bold">{resolutionRequests.length}</span>
				{/if}
			</button>
			<button
				onclick={() => (activeTab = 'reports')}
				class="px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] border-b-2 transition-colors
					{activeTab === 'reports'
						? 'border-red-500 text-red-700'
						: 'border-transparent text-surface-500 hover:text-surface-700 hover:border-surface-300'}"
			>
				Reports
				{#if reports.length > 0}
					<span class="ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full bg-red-100 text-red-700 text-[10px] font-bold">{reports.length}</span>
				{/if}
			</button>
		</div>

		<!-- Tab: Pending Review -->
		{#if activeTab === 'review'}
			{#if pending.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No markets pending review.</div>
			{:else}
				<div class="space-y-3">
					{#each pending as market}
						<div class="sc-card p-5">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-2">
										<span class="sc-tag">{market.category.replace('_', ' ')}</span>
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

		<!-- Tab: Reports -->
		{#if activeTab === 'reports'}
			{#if reports.length === 0}
				<div class="sc-card p-8 text-center text-surface-400 text-sm">No pending reports.</div>
			{:else}
				<div class="space-y-3">
					{#each reports as report}
						<div class="sc-card p-5 border-l-4 border-l-red-400">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<p class="text-surface-500 text-xs mb-1">
										Reported by <span class="font-semibold text-surface-700">{report.reporter_name}</span>
										&middot; {new Date(report.created_at).toLocaleString()}
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
			{:else}
				<div class="space-y-3">
					{#each resolutionRequests as rr}
						<div class="sc-card p-5 border-l-4 border-l-amber-400 bg-amber-50/20">
							<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-2">
										<span class="sc-tag">{rr.category.replace('_', ' ')}</span>
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
									<button
										onclick={() => doResolve(rr.id, 'yes')}
										disabled={actingId === rr.id}
										class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider"
									>
										Resolve YES
									</button>
									<button
										onclick={() => doResolve(rr.id, 'no')}
										disabled={actingId === rr.id}
										class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider"
									>
										Resolve NO
									</button>
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


