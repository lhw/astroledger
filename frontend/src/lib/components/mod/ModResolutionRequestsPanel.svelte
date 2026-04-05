<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { CATEGORY_LABELS } from '$lib/categories';
	import { formatDate, formatDateTime } from '$lib/format';
	import type { ResolutionRequestMarket } from '$lib/types';

	let {
		resolutionRequests,
		filteredResolutionRequests,
		resolveEvidence = $bindable(),
		actingId,
		onResolve,
		onDeny
	}: {
		resolutionRequests: ResolutionRequestMarket[];
		filteredResolutionRequests: ResolutionRequestMarket[];
		resolveEvidence: Record<number, string>;
		actingId: number | null;
		onResolve: (id: number, outcomeId: number) => Promise<void>;
		onDeny: (id: number) => Promise<void>;
	} = $props();
</script>

{#if resolutionRequests.length === 0}
	<EmptyState message="No resolution requests." />
{:else if filteredResolutionRequests.length === 0}
	<EmptyState message="No resolution requests match the current filters." />
{:else}
	<div class="space-y-3">
		{#each filteredResolutionRequests as request}
			<div class="sc-card p-5 border-l-4 border-l-amber-400 bg-amber-50/20">
				<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-2">
							<span class="sc-tag">{CATEGORY_LABELS[request.category] ?? request.category}</span>
							<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-700 border border-amber-200">Resolution Requested</span>
						</div>
						<h2 class="text-surface-800 font-semibold text-base"><a href="/markets/{request.id}" class="hover:text-primary-600 transition-colors">{request.title}</a></h2>
						{#if request.resolution_criteria}
							<p class="text-surface-500 text-xs mt-1 italic">Criteria: {request.resolution_criteria}</p>
						{/if}
						<p class="text-surface-400 text-xs mt-1">Created by {request.creator_name} · closes {formatDate(request.resolution_deadline)}</p>
						<div class="mt-3 pt-3 border-t border-amber-100 space-y-1.5">
							<p class="text-xs text-surface-600"><span class="font-semibold">Requested by:</span> {request.requester_name} · {formatDateTime(request.requested_at)}</p>
							{#if request.request_link}
								<p class="text-xs"><span class="font-semibold text-surface-600">Evidence:</span><a href={request.request_link} target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline break-all ml-1">{request.request_link}</a></p>
							{/if}
							{#if request.request_note}
								<p class="text-xs text-surface-600"><span class="font-semibold">Note:</span><span class="ml-1 italic">{request.request_note}</span></p>
							{/if}
						</div>
					</div>
					<div class="flex flex-col gap-2 shrink-0">
						<input type="url" bind:value={resolveEvidence[request.id]} placeholder="Evidence link (optional)" class="sc-input text-xs w-44" />
						{#each (request.outcomes ?? []) as outcome, index}
							<button onclick={() => onResolve(request.id, outcome.id)} disabled={actingId === request.id} class="btn btn-sm text-white text-xs uppercase tracking-wider {index === 0 ? 'bg-green-600 hover:bg-green-700' : index === 1 ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'}">Resolve {outcome.label}</button>
						{/each}
						<button onclick={() => onDeny(request.id)} disabled={actingId === request.id} class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider">Deny</button>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}