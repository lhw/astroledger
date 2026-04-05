<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { CATEGORY_LABELS } from '$lib/categories';
	import { formatDateTime } from '$lib/format';
	import type { Report } from '$lib/types';

	let {
		reports,
		filteredReports,
		actingId,
		onReview,
		onDismiss
	}: {
		reports: Report[];
		filteredReports: Report[];
		actingId: number | null;
		onReview: (id: number) => Promise<void>;
		onDismiss: (id: number) => Promise<void>;
	} = $props();
</script>

{#if reports.length === 0}
	<EmptyState message="No pending reports." />
{:else if filteredReports.length === 0}
	<EmptyState message="No reports match the current filters." />
{:else}
	<div class="space-y-3">
		{#each filteredReports as report}
			<div class="sc-card p-5 border-l-4 border-l-red-400">
				<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
					<div class="flex-1 min-w-0">
						<p class="text-surface-500 text-xs mb-1">Reported by <span class="font-semibold text-surface-700">{report.reporter_name}</span> · {formatDateTime(report.created_at)}</p>
						<p class="text-surface-400 text-[11px] mb-1 uppercase tracking-wider">{CATEGORY_LABELS[report.category] ?? report.category}</p>
						<h2 class="text-surface-800 font-semibold text-base"><a href="/markets/{report.market_id}" class="hover:text-primary-600 transition-colors">{report.market_title}</a></h2>
						<p class="text-surface-600 text-sm mt-2 bg-surface-50 border border-surface-200 rounded p-2 italic">&ldquo;{report.reason}&rdquo;</p>
					</div>
					<div class="flex flex-col gap-2 shrink-0">
						<button onclick={() => onReview(report.id)} disabled={actingId === report.id} class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider" title="Mark as reviewed — take action on the market separately if needed">Reviewed</button>
						<button onclick={() => onDismiss(report.id)} disabled={actingId === report.id} class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider">Dismiss</button>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}