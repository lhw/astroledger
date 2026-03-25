<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMarket } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import type { MarketCategory } from '$lib/types';

	let title = $state('');
	let description = $state('');
	let resolutionCriteria = $state('');
	let category = $state<MarketCategory>('bug_fixes');
	let deadlineType = $state<'date' | 'patch'>('date');
	let deadlineDate = $state('');
	let patchTarget = $state('');
	let submitting = $state(false);
	let error = $state('');

	const minDate = $derived(() => {
		const d = new Date();
		d.setDate(d.getDate() + 1);
		return d.toISOString().slice(0, 10);
	});

	const isValid = $derived(
		title.trim().length > 0 &&
			(deadlineType === 'date' ? deadlineDate.length > 0 : patchTarget.trim().length > 0)
	);

	const categories: { value: MarketCategory; label: string }[] = [
		{ value: 'bug_fixes', label: 'Bug Fix' },
		{ value: 'feature_delivery', label: 'Feature / Patch' },
		{ value: 'patch_timing', label: 'Patch Timing' },
		{ value: 'community_events', label: 'Community Event' },
		{ value: 'meta', label: 'Meta' }
	];

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!isValid) return;
		submitting = true;
		error = '';
		try {
			let deadline: string;
			let criteria = resolutionCriteria.trim();

			if (deadlineType === 'patch') {
				// Set deadline 2 years out; resolution criteria records the target patch.
				const far = new Date();
				far.setFullYear(far.getFullYear() + 2);
				deadline = far.toISOString();
				const prefix = `Resolves when patch ${patchTarget.trim()} ships.`;
				criteria = criteria ? `${prefix} ${criteria}` : prefix;
			} else {
				deadline = new Date(deadlineDate + 'T23:59:59Z').toISOString();
			}

			const market = await createMarket({
				title: title.trim(),
				description: description.trim(),
				resolution_criteria: criteria,
				category,
				deadline
			});
			goto(`/markets/${market.id}`);
		} catch (e) {
			error = String(e);
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Create Market — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-2xl py-8">
	<div class="mb-6">
		<a href="/markets" class="text-surface-400 hover:text-primary-400 text-sm no-underline">← Markets</a>
		<h1 class="text-3xl font-bold text-surface-100 mt-2">Create a Market</h1>
		<p class="text-surface-400 text-sm mt-1">
			Predict something Star Citizen-related. Be specific. No personal attacks.
		</p>
	</div>

	{#if !$isLoggedIn}
		<div class="card preset-tonal-surface p-6 rounded-lg text-center">
			<p class="text-surface-300 mb-4">You must be logged in to create a market.</p>
			<a href="/auth/login" class="btn preset-filled-primary-500">Login with SCID</a>
		</div>
	{:else}
		<form onsubmit={handleSubmit} class="space-y-5">
			{#if error}
				<div class="p-4 bg-error-500/20 border border-error-500 rounded-lg text-error-300 text-sm">
					{error}
				</div>
			{/if}

			<label class="block">
				<span class="text-surface-300 text-sm font-medium">Question <span class="text-error-400">*</span></span>
				<input
					type="text"
					bind:value={title}
					required
					maxlength="200"
					placeholder="Will [bug] be fixed before [date]?"
					class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 focus:border-primary-500 outline-none"
				/>
				<span class="text-surface-500 text-xs">{title.length}/200</span>
			</label>

			<label class="block">
				<span class="text-surface-300 text-sm font-medium">Category <span class="text-error-400">*</span></span>
				<select
					bind:value={category}
					class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 focus:border-primary-500 outline-none"
				>
					{#each categories as cat}
						<option value={cat.value}>{cat.label}</option>
					{/each}
				</select>
			</label>

			<!-- Deadline type toggle -->
			<fieldset class="block">
				<legend class="text-surface-300 text-sm font-medium mb-2">
					Resolution Timing <span class="text-error-400">*</span>
				</legend>
				<div class="flex gap-2 mb-3">
					<button
						type="button"
						onclick={() => (deadlineType = 'date')}
						class="flex-1 btn btn-sm {deadlineType === 'date' ? 'preset-filled-primary-500' : 'preset-outlined'}"
					>
						Specific Date
					</button>
					<button
						type="button"
						onclick={() => (deadlineType = 'patch')}
						class="flex-1 btn btn-sm {deadlineType === 'patch' ? 'preset-filled-primary-500' : 'preset-outlined'}"
					>
						Patch Release
					</button>
				</div>

				{#if deadlineType === 'date'}
					<label class="block">
						<span class="text-surface-400 text-xs">Closes on</span>
						<input
							type="date"
							bind:value={deadlineDate}
							required
							min={minDate()}
							class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 focus:border-primary-500 outline-none"
						/>
					</label>
				{:else}
					<label class="block">
						<span class="text-surface-400 text-xs">Target patch version</span>
						<input
							type="text"
							bind:value={patchTarget}
							required
							placeholder="e.g. 4.1.0, 4.2.0-PTU"
							class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 focus:border-primary-500 outline-none"
						/>
						<span class="text-surface-500 text-xs">
							Market resolves when this patch ships. Moderators resolve it manually.
						</span>
					</label>
				{/if}
			</fieldset>

			<label class="block">
				<span class="text-surface-300 text-sm font-medium">Description</span>
				<textarea
					bind:value={description}
					rows="3"
					maxlength="2000"
					placeholder="Provide context for your question…"
					class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 focus:border-primary-500 outline-none resize-y"
				></textarea>
			</label>

			<label class="block">
				<span class="text-surface-300 text-sm font-medium">Resolution Criteria</span>
				<textarea
					bind:value={resolutionCriteria}
					rows="3"
					maxlength="2000"
					placeholder="How will this market be resolved? What counts as YES vs NO?"
					class="mt-1 w-full bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 focus:border-primary-500 outline-none resize-y"
				></textarea>
			</label>

			<div class="pt-2">
				<button type="submit" disabled={submitting || !isValid} class="btn preset-filled-primary-500 w-full">
					{submitting ? 'Submitting…' : 'Submit for Review'}
				</button>
				<p class="text-surface-500 text-xs text-center mt-2">
					Markets are reviewed before going live. No harassment, no player targeting.
				</p>
			</div>
		</form>
	{/if}
</div>
