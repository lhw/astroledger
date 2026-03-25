<script lang="ts">
	import { onMount } from 'svelte';
	import { listMarkets } from '$lib/api';
	import type { MarketList, MarketCategory } from '$lib/types';

	let markets: MarketList | null = null;
	let loading = true;
	let error = '';

	let statusFilter = 'active';
	let categoryFilter: MarketCategory | '' = '';
	let offset = 0;

	const CATEGORIES: { value: MarketCategory | ''; label: string }[] = [
		{ value: '', label: 'All Categories' },
		{ value: 'bug_fixes', label: 'Bug Fixes' },
		{ value: 'feature_delivery', label: 'Feature Delivery' },
		{ value: 'patch_timing', label: 'Patch Timing' },
		{ value: 'community_events', label: 'Community Events' },
		{ value: 'meta', label: 'Meta' }
	];

	async function load() {
		loading = true;
		error = '';
		try {
			markets = await listMarkets(statusFilter, categoryFilter, offset);
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	}

	onMount(load);

	function prev() {
		offset = Math.max(0, offset - 20);
		load();
	}
	function next() {
		offset += 20;
		load();
	}
</script>

<svelte:head>
	<title>Markets — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-8">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-3xl font-bold text-surface-100">Markets</h1>
		<a href="/markets/new" class="btn btn-sm preset-filled-primary-500 no-underline">+ Submit Market</a>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap gap-3 mb-6">
		<select
			bind:value={statusFilter}
			onchange={() => { offset = 0; load(); }}
			class="bg-surface-800 border border-surface-600 rounded-lg px-3 py-1.5 text-surface-100 text-sm"
		>
			<option value="active">Active</option>
			<option value="resolved">Resolved</option>
			<option value="pending_review">Pending Review</option>
		</select>

		<select
			bind:value={categoryFilter}
			onchange={() => { offset = 0; load(); }}
			class="bg-surface-800 border border-surface-600 rounded-lg px-3 py-1.5 text-surface-100 text-sm"
		>
			{#each CATEGORIES as cat}
				<option value={cat.value}>{cat.label}</option>
			{/each}
		</select>
	</div>

	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading markets…</div>
	{:else if error}
		<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 mb-4">{error}</div>
	{:else if !markets || markets.markets.length === 0}
		<div class="card preset-tonal-surface p-8 text-center rounded-lg">
			<p class="text-surface-400">No markets found.</p>
		</div>
	{:else}
		<div class="space-y-3 mb-6">
			{#each markets.markets as market}
				<a
					href="/markets/{market.id}"
					class="card preset-tonal-surface p-4 rounded-lg flex items-start justify-between gap-4 hover:preset-tonal-primary transition-all no-underline block"
				>
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-1">
							<span class="badge preset-tonal-surface text-xs">{market.category.replace('_', ' ')}</span>
							{#if market.status === 'resolved'}
								<span class="badge preset-filled-success-500 text-xs">Resolved</span>
							{/if}
						</div>
						<div class="text-surface-100 font-medium">{market.title}</div>
						<div class="text-surface-400 text-xs mt-1">
							by {market.creator_name} · closes {new Date(market.resolution_deadline).toLocaleDateString()}
						</div>
					</div>
				</a>
			{/each}
		</div>

		<!-- Pagination -->
		<div class="flex items-center justify-between">
			<button onclick={prev} disabled={offset === 0} class="btn btn-sm preset-outlined">← Prev</button>
			<span class="text-surface-400 text-sm">{offset + 1}–{offset + markets.markets.length} of {markets.total}</span>
			<button
				onclick={next}
				disabled={offset + markets.markets.length >= markets.total}
				class="btn btn-sm preset-outlined"
			>
				Next →
			</button>
		</div>
	{/if}
</div>
