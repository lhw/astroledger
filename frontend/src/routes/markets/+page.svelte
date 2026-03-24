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
		{ value: 'bug_fixes', label: '🐛 Bug Fixes' },
		{ value: 'feature_delivery', label: '🚀 Feature Delivery' },
		{ value: 'patch_timing', label: '⏰ Patch Timing' },
		{ value: 'cig_drama', label: '🎭 CIG Drama' },
		{ value: 'community_events', label: '🎉 Community Events' },
		{ value: 'meta', label: '🤔 Meta' }
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

	function prev() { offset = Math.max(0, offset - 20); load(); }
	function next() { offset += 20; load(); }
</script>

<div class="container mx-auto px-4 max-w-4xl py-8">
	<div class="flex items-center justify-between mb-6">
		<h1 class="h2 text-surface-100">Markets</h1>
		<a href="/markets/new" class="btn btn-sm variant-filled-primary">+ Submit Market</a>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap gap-3 mb-6">
		<select
			bind:value={statusFilter}
			on:change={() => { offset = 0; load(); }}
			class="select variant-form-material w-auto text-sm"
		>
			<option value="active">Active</option>
			<option value="resolved">Resolved</option>
			<option value="pending_review">Pending Review</option>
		</select>

		<select
			bind:value={categoryFilter}
			on:change={() => { offset = 0; load(); }}
			class="select variant-form-material w-auto text-sm"
		>
			{#each CATEGORIES as cat}
				<option value={cat.value}>{cat.label}</option>
			{/each}
		</select>
	</div>

	{#if loading}
		<div class="text-surface-400 text-center py-16">Loading markets…</div>
	{:else if error}
		<div class="alert variant-filled-error">{error}</div>
	{:else if !markets || markets.markets.length === 0}
		<div class="card variant-ghost-surface p-12 text-center rounded-lg">
			<p class="text-surface-400">No markets found.</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each markets.markets as market}
				<a
					href="/markets/{market.id}"
					class="card variant-glass-surface p-4 rounded-lg block hover:variant-glass-primary transition-all"
				>
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1">
								<span class="badge variant-filled-surface text-xs">{market.category.replace('_', ' ')}</span>
								{#if market.status === 'resolved'}
									<span class="badge variant-filled-success text-xs">
										✓ {market.resolution?.toUpperCase()}
									</span>
								{/if}
							</div>
							<div class="text-surface-100 font-medium">
								{market.title}
							</div>
							<div class="text-surface-400 text-xs mt-1">
								by {market.creator_name} ·
								{market.status === 'resolved' ? 'resolved' : 'closes'}
								{new Date(market.resolution_deadline).toLocaleDateString()}
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>

		<!-- Pagination -->
		<div class="flex justify-between items-center mt-6">
			<button
				on:click={prev}
				disabled={offset === 0}
				class="btn btn-sm variant-ghost"
			>
				← Previous
			</button>
			<span class="text-surface-400 text-sm">
				{offset + 1}–{Math.min(offset + 20, markets.total)} of {markets.total}
			</span>
			<button
				on:click={next}
				disabled={offset + 20 >= markets.total}
				class="btn btn-sm variant-ghost"
			>
				Next →
			</button>
		</div>
	{/if}
</div>
