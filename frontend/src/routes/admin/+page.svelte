<script lang="ts">
	import { adminTriggerWeeklyPayout, adminAdjustBalance, ApiClientError } from '$lib/api';

	// Payout state
	let payoutLoading = $state(false);
	let payoutResult = $state<string | null>(null);
	let payoutError = $state<string | null>(null);

	// Balance adjustment state
	let adjUserId = $state('');
	let adjAmount = $state('');
	let adjReason = $state('');
	let adjLoading = $state(false);
	let adjResult = $state<string | null>(null);
	let adjError = $state<string | null>(null);

	async function triggerPayout() {
		payoutLoading = true;
		payoutResult = null;
		payoutError = null;
		try {
			const res = await adminTriggerWeeklyPayout();
			payoutResult = `${res.message} — ${res.users_paid} users received ${res.credits_per_user} bUEC each.`;
		} catch (err) {
			payoutError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			payoutLoading = false;
		}
	}

	async function submitAdjustment() {
		adjResult = null;
		adjError = null;

		const userId = parseInt(adjUserId, 10);
		const amount = parseInt(adjAmount, 10);

		if (!Number.isInteger(userId) || userId <= 0) {
			adjError = 'User ID must be a positive integer.';
			return;
		}
		if (!Number.isInteger(amount) || amount === 0) {
			adjError = 'Amount must be a non-zero integer.';
			return;
		}
		if (!adjReason.trim()) {
			adjError = 'Reason is required.';
			return;
		}
		if (adjReason.length > 200) {
			adjError = 'Reason must be at most 200 characters.';
			return;
		}

		adjLoading = true;
		try {
			const res = await adminAdjustBalance(userId, amount, adjReason.trim());
			adjResult = `Done. User ${userId} new balance: ${res.new_balance.toLocaleString()} bUEC.`;
			adjUserId = '';
			adjAmount = '';
			adjReason = '';
		} catch (err) {
			adjError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Admin Panel — ScolyMarket</title>
</svelte:head>

<div class="max-w-2xl mx-auto px-6 py-10 space-y-10">
	<h1 class="text-2xl font-bold text-primary-400 tracking-widest uppercase">Admin Panel</h1>

	<!-- Weekly Payout -->
	<section class="bg-surface-800 border border-surface-700 rounded-lg p-6 space-y-4">
		<h2 class="text-lg font-semibold text-surface-100 uppercase tracking-wide">Weekly Payout</h2>
		<p class="text-surface-400 text-sm">
			Manually trigger the 200 bUEC weekly credit payout for all users. Idempotent — safe to call
			even if the cron already ran this week (returns a 409 in that case).
		</p>

		<button
			onclick={triggerPayout}
			disabled={payoutLoading}
			class="btn preset-filled-primary-500 tracking-wider uppercase text-sm disabled:opacity-50"
		>
			{payoutLoading ? 'Running…' : 'Trigger Weekly Payout'}
		</button>

		{#if payoutResult}
			<p class="text-green-400 text-sm">{payoutResult}</p>
		{/if}
		{#if payoutError}
			<p class="text-red-400 text-sm">Error: {payoutError}</p>
		{/if}
	</section>

	<!-- Balance Adjustment -->
	<section class="bg-surface-800 border border-surface-700 rounded-lg p-6 space-y-4">
		<h2 class="text-lg font-semibold text-surface-100 uppercase tracking-wide">
			Adjust User Balance
		</h2>
		<p class="text-surface-400 text-sm">
			Add or remove bUEC from a user's balance. Provide a positive amount to add, negative to
			remove. The balance cannot go below 0.
		</p>

		<div class="space-y-3">
			<div>
				<label class="block text-surface-300 text-xs uppercase tracking-wide mb-1" for="adj-user-id"
					>User ID</label
				>
				<input
					id="adj-user-id"
					type="number"
					bind:value={adjUserId}
					placeholder="e.g. 42"
					class="input w-full bg-surface-700 border border-surface-600 rounded px-3 py-2 text-surface-100 text-sm"
				/>
			</div>

			<div>
				<label class="block text-surface-300 text-xs uppercase tracking-wide mb-1" for="adj-amount"
					>Amount (bUEC)</label
				>
				<input
					id="adj-amount"
					type="number"
					bind:value={adjAmount}
					placeholder="e.g. 500 or -100"
					class="input w-full bg-surface-700 border border-surface-600 rounded px-3 py-2 text-surface-100 text-sm"
				/>
			</div>

			<div>
				<label class="block text-surface-300 text-xs uppercase tracking-wide mb-1" for="adj-reason"
					>Reason</label
				>
				<textarea
					id="adj-reason"
					bind:value={adjReason}
					rows={3}
					maxlength={200}
					placeholder="E.g. compensation for bug, contest prize…"
					class="textarea w-full bg-surface-700 border border-surface-600 rounded px-3 py-2 text-surface-100 text-sm resize-none"
				></textarea>
				<p class="text-surface-500 text-xs mt-1">{adjReason.length}/200</p>
			</div>

			<button
				onclick={submitAdjustment}
				disabled={adjLoading}
				class="btn preset-filled-primary-500 tracking-wider uppercase text-sm disabled:opacity-50"
			>
				{adjLoading ? 'Adjusting…' : 'Apply Adjustment'}
			</button>
		</div>

		{#if adjResult}
			<p class="text-green-400 text-sm">{adjResult}</p>
		{/if}
		{#if adjError}
			<p class="text-red-400 text-sm">Error: {adjError}</p>
		{/if}
	</section>
</div>
