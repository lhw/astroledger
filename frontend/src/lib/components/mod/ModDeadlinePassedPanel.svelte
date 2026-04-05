<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { CATEGORY_LABELS } from '$lib/categories';
	import { formatDate } from '$lib/format';
	import type { Market } from '$lib/types';

	type PendingMarket = Market & { creator_name: string };

	let {
		deadlinePassed,
		filteredDeadlinePassed,
		resolveEvidence = $bindable(),
		actingId,
		onResolve
	}: {
		deadlinePassed: PendingMarket[];
		filteredDeadlinePassed: PendingMarket[];
		resolveEvidence: Record<number, string>;
		actingId: number | null;
		onResolve: (id: number, outcomeId: number) => Promise<void>;
	} = $props();
</script>

{#if deadlinePassed.length === 0}
	<EmptyState message="No deadline-passed markets awaiting resolution." />
{:else if filteredDeadlinePassed.length === 0}
	<EmptyState message="No deadline-passed markets match the current filters." />
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
						<h2 class="text-surface-800 font-semibold text-base"><a href="/markets/{market.id}" class="hover:text-primary-600 transition-colors">{market.title}</a></h2>
						{#if market.description}
							<p class="text-surface-600 text-sm mt-1 line-clamp-2">{market.description}</p>
						{/if}
						{#if market.resolution_criteria}
							<p class="text-surface-500 text-xs mt-1 italic">Criteria: {market.resolution_criteria}</p>
						{/if}
						<p class="text-surface-400 text-xs mt-2">{market.creator_name} · closed {formatDate(market.resolution_deadline)}</p>
					</div>
					<div class="flex flex-col gap-2 shrink-0">
						<input type="url" bind:value={resolveEvidence[market.id]} placeholder="Evidence link (optional)" class="sc-input text-xs w-44" />
						{#each (market.outcomes ?? []) as outcome, index}
							<button onclick={() => onResolve(market.id, outcome.id)} disabled={actingId === market.id} class="btn btn-sm text-white text-xs uppercase tracking-wider {index === 0 ? 'bg-green-600 hover:bg-green-700' : index === 1 ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'}">
								Resolve {outcome.label}
							</button>
						{/each}
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}