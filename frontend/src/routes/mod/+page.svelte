<script lang="ts">
	import { onMount } from 'svelte';
	import { listPendingMarkets, approveMarket, rejectMarket, resolveMarket } from '$lib/api';
	import { isModerator } from '$lib/stores/auth';
	import type { Market } from '$lib/types';

	let pending = $state<(Market & { creator_name: string })[]>([]);
	let loading = $state(true);
	let error = $state('');
	let actionError = $state('');
	let actingId = $state<number | null>(null);

	onMount(async () => {
		await load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			pending = await listPendingMarkets();
		} catch (e) {
			error = String(e);
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
			actionError = String(e);
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
		await withAction(id, () => resolveMarket(id, resolution));
	}
</script>

<svelte:head>
	<title>Mod Queue — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-8">
	<h1 class="text-3xl font-bold text-surface-100 mb-2">Mod Queue</h1>
	<p class="text-surface-400 text-sm mb-8">Review and manage markets pending approval or resolution.</p>

	{#if !$isModerator}
		<div class="card preset-tonal-surface p-8 rounded-lg text-center">
			<p class="text-surface-300">You must be a moderator to access this page.</p>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16">Loading queue…</div>
	{:else if error}
		<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mb-4">{error}</div>
	{:else}
		{#if actionError}
			<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mb-4">{actionError}</div>
		{/if}

		{#if pending.length === 0}
			<div class="text-surface-500 text-center py-16">No markets pending review.</div>
		{:else}
			<div class="space-y-4">
				{#each pending as market}
					<div class="card preset-tonal-surface p-5 rounded-lg">
						<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="badge preset-tonal-surface text-xs">{market.category.replace('_', ' ')}</span>
									<span class="badge preset-tonal-surface text-xs">{market.status}</span>
								</div>
								<h2 class="text-surface-100 font-semibold text-base">{market.title}</h2>
								{#if market.description}
									<p class="text-surface-400 text-sm mt-1 line-clamp-2">{market.description}</p>
								{/if}
								{#if market.resolution_criteria}
									<p class="text-surface-500 text-xs mt-1 italic">Criteria: {market.resolution_criteria}</p>
								{/if}
								<p class="text-surface-500 text-xs mt-2">
									by {market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
								</p>
							</div>

							<div class="flex flex-col gap-2 shrink-0">
								{#if market.status === 'pending_review'}
									<button
										onclick={() => doApprove(market.id)}
										disabled={actingId === market.id}
										class="btn btn-sm preset-filled-success-500"
									>
										Approve
									</button>
									<button
										onclick={() => doReject(market.id)}
										disabled={actingId === market.id}
										class="btn btn-sm preset-filled-error-500"
									>
										Reject
									</button>
								{:else if market.status === 'active'}
									<button
										onclick={() => doResolve(market.id, 'yes')}
										disabled={actingId === market.id}
										class="btn btn-sm preset-filled-success-500"
									>
										Resolve YES
									</button>
									<button
										onclick={() => doResolve(market.id, 'no')}
										disabled={actingId === market.id}
										class="btn btn-sm preset-filled-error-500"
									>
										Resolve NO
									</button>
								{/if}
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>
