<script lang="ts">
	import type { MarketStatus } from '$lib/types';

	let { status }: { status: MarketStatus } = $props();

	const labels: Record<MarketStatus, string> = {
		pending_review: 'Pending Review',
		active: 'Active',
		resolved: 'Resolved',
		cancelled: 'Cancelled',
		resolution_requested: 'Resolution Requested',
		deadline_passed: 'Deadline Passed'
	};

	const styles: Record<MarketStatus, string> = {
		pending_review: 'bg-primary-50 text-primary-700 border-primary-200',
		active: 'bg-primary-50 text-primary-700 border-primary-200',
		resolved: 'bg-green-100 text-green-700 border-green-200',
		cancelled: 'bg-surface-100 text-surface-500 border-surface-200',
		resolution_requested: 'bg-amber-100 text-amber-700 border-amber-200',
		deadline_passed: 'bg-amber-100 text-amber-700 border-amber-200'
	};
</script>

<span class={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border ${styles[status]}`}>
	{labels[status]}
</span>