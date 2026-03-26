<script lang="ts">
	import { goto } from '$app/navigation';
	import { createMarket } from '$lib/api';
	import { isLoggedIn } from '$lib/stores/auth';
	import type { MarketCategory, MarketResolutionType } from '$lib/types';
	import { CATEGORY_CREATE_OPTIONS } from '$lib/categories';

	let title = $state('');
	let description = $state('');
	let resolutionCriteria = $state('');
	let category = $state<MarketCategory>('bug_fixes');
	let deadlineType = $state<'date' | 'patch'>('date');
	let deadlineDate = $state('');
	let patchTarget = $state('');
	let resolutionType = $state<MarketResolutionType>('binary');
	let resolutionThreshold = $state('');
	let submitting = $state(false);
	let error = $state('');

	const minDate = $derived(() => {
		const d = new Date();
		d.setDate(d.getDate() + 1);
		return d.toISOString().slice(0, 10);
	});

	const isValid = $derived(
		title.trim().length > 0 &&
			(deadlineType === 'date' ? deadlineDate.length > 0 : patchTarget.trim().length > 0) &&
			(resolutionType === 'binary' || resolutionThreshold.trim().length > 0)
	);

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
				deadline,
				resolution_type: resolutionType,
				resolution_threshold: resolutionType !== 'binary' ? resolutionThreshold.trim() : undefined
			});
			goto(`/markets/${market.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Create Market — ScolyMarket</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-2xl py-10">
	<div class="mb-7">
		<a href="/markets" class="text-surface-500 hover:text-primary-600 text-xs uppercase tracking-wider no-underline transition-colors">← Markets</a>
		<h1 class="text-2xl font-bold text-surface-900 mt-3 tracking-tight">Create a Market</h1>
		<p class="text-surface-500 text-sm mt-1">
			Predict something Star Citizen-related. Be specific. No personal attacks.
		</p>
	</div>

	{#if !$isLoggedIn}
		<div class="sc-card p-6 text-center">
			<p class="text-surface-600 mb-4 text-sm">You must be logged in to create a market.</p>
			<a href="/auth/login" class="btn preset-filled-primary-500 text-xs uppercase tracking-wider">Login with SCID</a>
		</div>
	{:else}
		<form onsubmit={handleSubmit} class="space-y-5">
			{#if error}
				<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
					{error}
				</div>
			{/if}

			<label class="block">
				<span class="text-surface-700 text-xs font-bold uppercase tracking-wider">Question <span class="text-red-500">*</span></span>
				<input
					type="text"
					bind:value={title}
					required
					maxlength="200"
					placeholder="Will [bug] be fixed before [date]?"
					class="sc-input mt-1.5"
				/>
				<span class="text-surface-400 text-xs">{title.length}/200</span>
			</label>

			<label class="block">
				<span class="text-surface-700 text-xs font-bold uppercase tracking-wider">Category <span class="text-red-500">*</span></span>
				<select
					bind:value={category}
					class="sc-input mt-1.5"
				>
					{#each CATEGORY_CREATE_OPTIONS as cat}
						<option value={cat.value}>{cat.label}</option>
					{/each}
				</select>
			</label>

			<!-- Market type / resolution type -->
			<fieldset class="block">
				<legend class="text-surface-700 text-xs font-bold uppercase tracking-wider mb-2.5">Market Type <span class="text-red-500">*</span></legend>
				<div class="flex gap-2 mb-3">
					<button type="button" onclick={() => { resolutionType = 'binary'; resolutionThreshold = ''; }}
						class="flex-1 btn btn-sm {resolutionType === 'binary' ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400'} transition-colors text-xs uppercase tracking-wider">
						Yes / No
					</button>
					<button type="button" onclick={() => { resolutionType = 'date'; resolutionThreshold = ''; }}
						class="flex-1 btn btn-sm {resolutionType === 'date' ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400'} transition-colors text-xs uppercase tracking-wider">
						Date Prediction
					</button>
					<button type="button" onclick={() => { resolutionType = 'numeric'; resolutionThreshold = ''; }}
						class="flex-1 btn btn-sm {resolutionType === 'numeric' ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400'} transition-colors text-xs uppercase tracking-wider">
						Numeric ($)
					</button>
				</div>
				{#if resolutionType === 'binary'}
					<p class="text-surface-400 text-xs">Standard yes/no question. Resolves YES or NO based on your criteria.</p>
				{:else if resolutionType === 'date'}
					<label class="block">
						<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Will the event happen before… <span class="text-red-500">*</span></span>
						<input type="date" bind:value={resolutionThreshold} required class="sc-input mt-1.5" />
						<span class="text-surface-400 text-xs mt-1 block">Resolves YES if the event happens before this date, NO otherwise.</span>
					</label>
				{:else if resolutionType === 'numeric'}
					<label class="block">
						<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Threshold value (USD or units) <span class="text-red-500">*</span></span>
						<div class="flex items-center gap-1 mt-1.5">
							<span class="text-surface-500 font-semibold pl-2">$</span>
							<input type="number" bind:value={resolutionThreshold} required min="0" step="any" placeholder="e.g. 45" class="sc-input flex-1" />
						</div>
						<span class="text-surface-400 text-xs mt-1 block">Resolves YES if the value reaches or exceeds this threshold.</span>
					</label>
				{/if}
			</fieldset>
			<fieldset class="block">
				<legend class="text-surface-700 text-xs font-bold uppercase tracking-wider mb-2.5">
					Resolution Timing <span class="text-red-500">*</span>
				</legend>
				<div class="flex gap-2 mb-3">
					<button
						type="button"
						onclick={() => (deadlineType = 'date')}
						class="flex-1 btn btn-sm {deadlineType === 'date' ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400'} transition-colors text-xs uppercase tracking-wider"
					>
						Specific Date
					</button>
					<button
						type="button"
						onclick={() => (deadlineType = 'patch')}
						class="flex-1 btn btn-sm {deadlineType === 'patch' ? 'preset-filled-primary-500' : 'border border-surface-300 text-surface-600 hover:border-primary-400'} transition-colors text-xs uppercase tracking-wider"
					>
						Patch Release
					</button>
				</div>

				{#if deadlineType === 'date'}
					<label class="block">
						<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Closes on</span>
						<input
							type="date"
							bind:value={deadlineDate}
							required
							min={minDate()}
							class="sc-input mt-1.5"
						/>
					</label>
				{:else}
					<label class="block">
						<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Target patch version</span>
						<input
							type="text"
							bind:value={patchTarget}
							required
							placeholder="e.g. 4.1.0, 4.2.0-PTU"
							class="sc-input mt-1.5"
						/>
						<span class="text-surface-400 text-xs mt-1 block">
							Market resolves when this patch ships. Moderators resolve it manually.
						</span>
					</label>
				{/if}
			</fieldset>

			<label class="block">
				<span class="text-surface-700 text-xs font-bold uppercase tracking-wider">Description</span>
				<textarea
					bind:value={description}
					rows="3"
					maxlength="2000"
					placeholder="Provide context for your question…"
					class="sc-input mt-1.5 resize-y"
				></textarea>
			</label>

			<label class="block">
				<span class="text-surface-700 text-xs font-bold uppercase tracking-wider">Resolution Criteria</span>
				<textarea
					bind:value={resolutionCriteria}
					rows="3"
					maxlength="2000"
					placeholder="How will this market be resolved? What counts as YES vs NO?"
					class="sc-input mt-1.5 resize-y"
				></textarea>
			</label>

			<div class="pt-2">
				<button type="submit" disabled={submitting || !isValid} class="btn preset-filled-primary-500 w-full uppercase tracking-wider text-sm">
					{submitting ? 'Submitting…' : 'Submit for Review'}
				</button>
				<p class="text-surface-400 text-xs text-center mt-2">
					Markets are reviewed before going live. No harassment, no player targeting.
				</p>
			</div>
		</form>
	{/if}
</div>
