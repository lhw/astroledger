<script lang="ts">
	import { onMount } from 'svelte';
	import { listPendingMarkets, approveMarket, rejectMarket } from '$lib/api';
	import { isModerator } from '$lib/stores/auth';
	import type { Market } from '$lib/types';

	let markets: (Market & { creator_name: string })[] = [];
	let loading = true;
	let error = '';

	onMount(async () => {
		if (!$isModerator) return;
		try {
			markets = await listPendingMarkets();
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	});

	async function approve(id: number) {
		await approveMarket(id);
		markets = markets.filter((m) => m.id !== id);
	}

	async function reject(id: number) {
		await rejectMarket(id);
		markets = markets.filter((m) => m.id !== id);
	}
</script>

<div class="container mx-auto px-4 max-w-3xl py-8">
	<h1 class="h2 text-surface-100 mb-2">Mod Queue</h1>
	<p class="text-surface-400 text-sm mb-6">Markets awaiting review. Approve or reject each one.</p>

	{#if !$isModerator}
		<div class="alert variant-filled-error">You don't have moderator access.</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16">Loading…</div>
	{:else if error}
		<div class="alert variant-filled-error">{error}</div>
	{:else if markets.length === 0}
		<div class="card variant-glass-surface p-12 text-center rounded-lg">
			<p class="text-surface-400">Queue is clear. Nice work. ✓</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each markets as market}
				<div class="card variant-glass-surface p-5 rounded-lg">
					<div class="mb-3">
						<span class="badge variant-filled-surface text-xs mb-1">{market.category.replace('_', ' ')}</span>
						<h3 class="text-surface-100 font-semibold">{market.title}</h3>
						<p class="text-surface-400 text-xs mt-1">
							Submitted by {market.creator_name} ·
							Deadline: {new Date(market.resolution_deadline).toLocaleDateString()}
						</p>
					</div>

					{#if market.description}
						<p class="text-surface-300 text-sm mb-2">{market.description}</p>
					{/if}

					{#if market.resolution_criteria}
						<div class="bg-surface-700 rounded p-3 mb-4">
							<p class="text-surface-400 text-xs uppercase tracking-wide mb-1">Resolution Criteria</p>
							<p class="text-surface-200 text-sm">{market.resolution_criteria}</p>
						</div>
					{/if}

					<div class="flex gap-3">
						<button
							on:click={() => approve(market.id)}
							class="btn btn-sm variant-filled-success flex-1"
						>
							✓ Approve
						</button>
						<button
							on:click={() => reject(market.id)}
							class="btn btn-sm variant-filled-error flex-1"
						>
							✗ Reject
						</button>
						<a
							href="/markets/{market.id}"
							class="btn btn-sm variant-ghost"
						>
							View
						</a>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
