<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { CATEGORY_LABELS } from '$lib/categories';
	import { formatDate } from '$lib/format';
	import type { Market } from '$lib/types';

	type PendingMarket = Market & { creator_name: string };

	let {
		pending,
		filteredPending,
		selectedPending,
		selectedPendingCount,
		actingId,
		bulkAction,
		onTogglePendingSelection,
		onToggleAllFilteredPending,
		onApprove,
		onReject,
		onBulkPending
	}: {
		pending: PendingMarket[];
		filteredPending: PendingMarket[];
		selectedPending: Record<number, boolean>;
		selectedPendingCount: number;
		actingId: number | null;
		bulkAction: 'approve' | 'reject' | null;
		onTogglePendingSelection: (id: number) => void;
		onToggleAllFilteredPending: () => void;
		onApprove: (id: number) => Promise<void>;
		onReject: (id: number) => Promise<void>;
		onBulkPending: (action: 'approve' | 'reject') => Promise<void>;
	} = $props();
</script>

{#if pending.length === 0}
	<EmptyState message="No markets pending review." />
{:else if filteredPending.length === 0}
	<EmptyState message="No pending markets match the current filters." />
{:else}
	<div class="mb-3 sc-card p-3 flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-3">
			<label class="inline-flex items-center gap-2 text-xs text-surface-600">
				<input type="checkbox" checked={filteredPending.length > 0 && selectedPendingCount === filteredPending.length} onchange={onToggleAllFilteredPending} />
				Select all filtered
			</label>
			<span class="text-xs text-surface-500">{selectedPendingCount} selected</span>
		</div>
		<div class="flex items-center gap-2">
			<button onclick={() => onBulkPending('approve')} disabled={selectedPendingCount === 0 || bulkAction !== null || actingId !== null} class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider disabled:opacity-50">
				{bulkAction === 'approve' ? 'Approving…' : 'Bulk Approve'}
			</button>
			<button onclick={() => onBulkPending('reject')} disabled={selectedPendingCount === 0 || bulkAction !== null || actingId !== null} class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider disabled:opacity-50">
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
							<input type="checkbox" checked={Boolean(selectedPending[market.id])} onchange={() => onTogglePendingSelection(market.id)} />
							<span class="sc-tag">{CATEGORY_LABELS[market.category] ?? market.category}</span>
						</div>
						<h2 class="text-surface-800 font-semibold text-base">{market.title}</h2>
						{#if market.description}
							<p class="text-surface-600 text-sm mt-1 line-clamp-2">{market.description}</p>
						{/if}
						{#if market.resolution_criteria}
							<p class="text-surface-500 text-xs mt-1 italic">Criteria: {market.resolution_criteria}</p>
						{/if}
						<p class="text-surface-400 text-xs mt-2">{market.creator_name} · closes {formatDate(market.resolution_deadline)}</p>
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
						<button onclick={() => onApprove(market.id)} disabled={actingId === market.id} class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider">Approve</button>
						<button onclick={() => onReject(market.id)} disabled={actingId === market.id} class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider">Reject</button>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}