<script lang="ts">
	import { onMount } from 'svelte';
	import { createMarket } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import type { MarketCategory } from '$lib/types';

	let title = '';
	let description = '';
	let category: MarketCategory = 'bug_fixes';
	let resolutionCriteria = '';
	let deadline = '';
	let submitting = false;
	let error = '';

	const CATEGORIES: { value: MarketCategory; label: string }[] = [
		{ value: 'bug_fixes', label: '🐛 Bug Fixes' },
		{ value: 'feature_delivery', label: '🚀 Feature Delivery' },
		{ value: 'patch_timing', label: '⏰ Patch Timing' },
		{ value: 'cig_drama', label: '🎭 CIG Drama' },
		{ value: 'community_events', label: '🎉 Community Events' },
		{ value: 'meta', label: '🤔 Meta' }
	];

	// Set minimum deadline to tomorrow
	const tomorrow = new Date();
	tomorrow.setDate(tomorrow.getDate() + 1);
	const minDeadline = tomorrow.toISOString().slice(0, 16);

	async function submit() {
		if (!title.trim() || !resolutionCriteria.trim() || !deadline) return;
		submitting = true;
		error = '';
		try {
			const market = await createMarket({
				title: title.trim(),
				description: description.trim(),
				category,
				resolution_criteria: resolutionCriteria.trim(),
				deadline: new Date(deadline).toISOString()
			});
			await goto(`/markets/${market.id}`);
		} catch (e) {
			error = String(e);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="container mx-auto px-4 max-w-2xl py-8">
	<div class="mb-6">
		<a href="/markets" class="text-surface-400 hover:text-primary-400 text-sm">← Back to Markets</a>
		<h1 class="h2 text-surface-100 mt-2">Submit a Market</h1>
		<p class="text-surface-400 text-sm mt-1">
			Markets go through mod review before going live. Keep it civilised.
		</p>
	</div>

	{#if !$isLoggedIn}
		<div class="alert variant-filled-warning">
			You must be logged in to submit a market.
		</div>
	{:else}
		<form on:submit|preventDefault={submit} class="card variant-glass-surface p-6 rounded-lg space-y-5">
			<label class="label">
				<span class="text-surface-300 text-sm font-medium">Question Title *</span>
				<input
					type="text"
					bind:value={title}
					placeholder="Will the Pyro system ship in 2025?"
					maxlength="200"
					required
					class="input mt-1"
				/>
				<span class="text-surface-500 text-xs">{title.length}/200</span>
			</label>

			<label class="label">
				<span class="text-surface-300 text-sm font-medium">Category *</span>
				<select bind:value={category} class="select mt-1">
					{#each CATEGORIES as cat}
						<option value={cat.value}>{cat.label}</option>
					{/each}
				</select>
			</label>

			<label class="label">
				<span class="text-surface-300 text-sm font-medium">Description</span>
				<textarea
					bind:value={description}
					placeholder="Optional context or background information…"
					rows="3"
					class="textarea mt-1"
				></textarea>
			</label>

			<label class="label">
				<span class="text-surface-300 text-sm font-medium">Resolution Criteria *</span>
				<textarea
					bind:value={resolutionCriteria}
					placeholder="This resolves YES if CIG officially releases Pyro in the patch notes before 2026-01-01."
					rows="3"
					required
					class="textarea mt-1"
				></textarea>
			</label>

			<label class="label">
				<span class="text-surface-300 text-sm font-medium">Resolution Deadline *</span>
				<input
					type="datetime-local"
					bind:value={deadline}
					min={minDeadline}
					required
					class="input mt-1"
				/>
			</label>

			{#if error}
				<div class="alert variant-filled-error text-sm">{error}</div>
			{/if}

			<div class="flex gap-3">
				<button
					type="submit"
					disabled={submitting || !title.trim() || !resolutionCriteria.trim() || !deadline}
					class="btn variant-filled-primary flex-1"
				>
					{submitting ? 'Submitting…' : 'Submit for Review'}
				</button>
				<a href="/markets" class="btn variant-ghost">Cancel</a>
			</div>

			<p class="text-surface-500 text-xs">
				Markets about real money, harassment, doxxing, or player targeting will be automatically rejected.
			</p>
		</form>
	{/if}
</div>
