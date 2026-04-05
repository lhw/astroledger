<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { formatDateTime } from '$lib/format';
	import type { DetectedPatch } from '$lib/types';

	let {
		patches,
		unseenPatches,
		actingId,
		onMarkSeen
	}: {
		patches: DetectedPatch[];
		unseenPatches: DetectedPatch[];
		actingId: number | null;
		onMarkSeen: (id: number) => Promise<void>;
	} = $props();
</script>

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
		<EmptyState message="No patches detected yet — scraper runs every 30 minutes." padding="p-5" />
	{:else}
		<div class="space-y-2">
			{#each patches as patch}
				<div class="sc-card p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 {patch.notified === 0 ? 'border-l-4 border-l-green-400' : 'opacity-60'}">
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
						<p class="text-surface-400 text-xs mt-0.5">Detected {formatDateTime(patch.first_seen_at)}</p>
					</div>
					<div class="flex items-center gap-2 shrink-0">
						<a href={patch.thread_url} target="_blank" rel="noopener noreferrer" class="btn btn-sm border border-surface-300 text-surface-600 hover:border-surface-400 text-xs uppercase tracking-wider">
							View Thread
						</a>
						{#if patch.notified === 0}
							<button onclick={() => onMarkSeen(patch.id)} disabled={actingId === patch.id} class="btn btn-sm bg-green-600 hover:bg-green-700 text-white text-xs uppercase tracking-wider">
								Mark Seen
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>