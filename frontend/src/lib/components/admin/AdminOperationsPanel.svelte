<script lang="ts">
	import {
		adminTriggerWeeklyPayout,
		adminAdjustBalance,
		adminSearchUsers,
		adminBanUser,
		adminShadowBanUser,
		ApiClientError
	} from '$lib/api';
	import type { UserSearchResult } from '$lib/types';

	let payoutLoading = $state(false);
	let payoutResult = $state<string | null>(null);
	let payoutError = $state<string | null>(null);

	let adjSearchQuery = $state('');
	let adjSearchResults = $state<UserSearchResult[]>([]);
	let adjSearchLoading = $state(false);
	let adjSelectedUser = $state<UserSearchResult | null>(null);
	let adjAmount = $state('');
	let adjReason = $state('');
	let adjLoading = $state(false);
	let adjResult = $state<string | null>(null);
	let adjError = $state<string | null>(null);
	let adjIsBanned = $state(false);
	let adjIsShadowBanned = $state(false);
	let adjBanLoading = $state(false);
	let adjShadowBanLoading = $state(false);
	let adjModError = $state<string | null>(null);

	let searchTimer: ReturnType<typeof setTimeout> | null = null;

	function resetSelectedUser() {
		adjSelectedUser = null;
		adjAmount = '';
		adjReason = '';
		adjResult = null;
		adjError = null;
		adjModError = null;
		adjIsBanned = false;
		adjIsShadowBanned = false;
	}

	function onSearchInput() {
		resetSelectedUser();
		adjSearchResults = [];
		if (searchTimer) clearTimeout(searchTimer);
		const query = adjSearchQuery.trim();
		if (query.length < 2) return;
		searchTimer = setTimeout(async () => {
			adjSearchLoading = true;
			try {
				adjSearchResults = await adminSearchUsers(query);
			} catch {
				adjSearchResults = [];
			} finally {
				adjSearchLoading = false;
			}
		}, 300);
	}

	function selectUser(user: UserSearchResult) {
		adjSelectedUser = user;
		adjIsBanned = user.is_banned === 1;
		adjIsShadowBanned = user.is_shadow_banned === 1;
		adjSearchQuery = '';
		adjSearchResults = [];
		adjModError = null;
	}

	async function triggerPayout() {
		payoutLoading = true;
		payoutResult = null;
		payoutError = null;
		try {
			const result = await adminTriggerWeeklyPayout();
			payoutResult = `${result.message} — ${result.users_paid} users received ${result.credits_per_user} bUEC each.`;
		} catch (err) {
			payoutError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			payoutLoading = false;
		}
	}

	async function submitAdjustment() {
		adjResult = null;
		adjError = null;
		if (!adjSelectedUser) {
			adjError = 'Select a user first.';
			return;
		}
		const amount = parseInt(adjAmount, 10);
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
			const result = await adminAdjustBalance(adjSelectedUser.id, amount, adjReason.trim());
			adjResult = `Done. ${adjSelectedUser.display_name} new balance: ${result.new_balance.toLocaleString()} bUEC.`;
			resetSelectedUser();
		} catch (err) {
			adjError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjLoading = false;
		}
	}

	async function toggleBan() {
		if (!adjSelectedUser) return;
		adjBanLoading = true;
		adjModError = null;
		try {
			await adminBanUser(adjSelectedUser.id, !adjIsBanned);
			adjIsBanned = !adjIsBanned;
		} catch (err) {
			adjModError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjBanLoading = false;
		}
	}

	async function toggleShadowBan() {
		if (!adjSelectedUser) return;
		adjShadowBanLoading = true;
		adjModError = null;
		try {
			await adminShadowBanUser(adjSelectedUser.id, !adjIsShadowBanned);
			adjIsShadowBanned = !adjIsShadowBanned;
		} catch (err) {
			adjModError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			adjShadowBanLoading = false;
		}
	}
</script>

<div class="space-y-8 max-w-2xl">
	<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 space-y-4">
		<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">
			Weekly Payout
		</h2>
		<p class="text-surface-400 text-sm leading-relaxed">
			Manually trigger the 200 bUEC weekly credit payout for all users. Idempotent — safe to
			call even if the cron already ran this week.
		</p>
		<button
			onclick={triggerPayout}
			disabled={payoutLoading}
			class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50"
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

	<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 space-y-4">
		<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">
			User Management
		</h2>
		<p class="text-surface-400 text-sm leading-relaxed">
			Search for a user to adjust their balance, ban them from logging in, or shadow-ban them to silently hide all their comments.
		</p>
		<div class="space-y-3">
			{#if adjSelectedUser}
				<div class="bg-surface-700 border border-primary-600 rounded-lg px-3 py-2.5 space-y-2.5">
					<div class="flex items-center justify-between gap-3">
						<div class="flex items-center gap-3 min-w-0">
							<div class="w-2 h-2 rounded-full bg-primary-400 shrink-0"></div>
							<div class="min-w-0">
								<p class="text-surface-100 text-sm font-medium truncate">{adjSelectedUser.display_name}</p>
								<p class="text-surface-400 text-xs">
									{#if adjSelectedUser.rsi_handle}{adjSelectedUser.rsi_handle} · {/if}ID {adjSelectedUser.id} · {adjSelectedUser.balance.toLocaleString()} bUEC
								</p>
							</div>
						</div>
						<button
							onclick={resetSelectedUser}
							class="text-surface-500 hover:text-surface-200 text-lg leading-none shrink-0"
							aria-label="Clear selection"
						>×</button>
					</div>
					<div class="flex flex-wrap items-center gap-2 pt-1 border-t border-surface-600">
						<span class="text-surface-400 text-xs uppercase tracking-wide">Moderation:</span>
						{#if adjIsBanned}
							<span class="text-xs font-semibold text-red-300 bg-red-900/40 border border-red-700 rounded px-2 py-0.5">Banned</span>
						{/if}
						{#if adjIsShadowBanned}
							<span class="text-xs font-semibold text-amber-300 bg-amber-900/40 border border-amber-700 rounded px-2 py-0.5">Shadow Banned</span>
						{/if}
						{#if !adjIsBanned && !adjIsShadowBanned}
							<span class="text-xs text-surface-500">No restrictions</span>
						{/if}
						<div class="flex gap-2 ml-auto">
							<button
								onclick={toggleBan}
								disabled={adjBanLoading || adjShadowBanLoading}
								class="text-xs px-2.5 py-1 rounded font-semibold uppercase tracking-wide transition-colors disabled:opacity-50 {adjIsBanned ? 'bg-green-800/60 hover:bg-green-700/60 text-green-300 border border-green-700' : 'bg-red-900/60 hover:bg-red-800/60 text-red-300 border border-red-700'}"
							>
								{adjBanLoading ? '…' : adjIsBanned ? 'Unban' : 'Ban'}
							</button>
							<button
								onclick={toggleShadowBan}
								disabled={adjBanLoading || adjShadowBanLoading}
								class="text-xs px-2.5 py-1 rounded font-semibold uppercase tracking-wide transition-colors disabled:opacity-50 {adjIsShadowBanned ? 'bg-surface-700 hover:bg-surface-600 text-surface-300 border border-surface-500' : 'bg-amber-900/60 hover:bg-amber-800/60 text-amber-300 border border-amber-700'}"
							>
								{adjShadowBanLoading ? '…' : adjIsShadowBanned ? 'Un-shadow-ban' : 'Shadow Ban'}
							</button>
						</div>
					</div>
					{#if adjModError}
						<p class="text-red-400 text-xs">Error: {adjModError}</p>
					{/if}
				</div>
			{:else}
				<div class="relative">
					<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-user-search">Search User</label>
					<div class="relative">
						<input
							id="adj-user-search"
							type="text"
							bind:value={adjSearchQuery}
							oninput={onSearchInput}
							placeholder="Type display name or RSI handle…"
							autocomplete="off"
							class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none pr-8"
						/>
						{#if adjSearchLoading}
							<span class="absolute right-2.5 top-1/2 -translate-y-1/2 text-surface-400 text-xs">…</span>
						{/if}
					</div>
					{#if adjSearchResults.length > 0}
						<ul class="absolute z-10 left-0 right-0 top-full mt-1 bg-surface-800 border border-surface-600 rounded-lg overflow-hidden shadow-xl">
							{#each adjSearchResults as user}
								<li>
									<button
										onclick={() => selectUser(user)}
										class="w-full text-left px-3 py-2.5 hover:bg-surface-700 transition-colors flex items-center justify-between gap-3"
									>
										<div class="min-w-0">
											<p class="text-surface-100 text-sm font-medium truncate">{user.display_name}</p>
											<p class="text-surface-400 text-xs">{#if user.rsi_handle}{user.rsi_handle} · {/if}ID {user.id}</p>
										</div>
										<span class="text-surface-400 text-xs tabular-nums shrink-0">{user.balance.toLocaleString()} bUEC</span>
									</button>
								</li>
							{/each}
						</ul>
					{:else if adjSearchQuery.trim().length >= 2 && !adjSearchLoading}
						<div class="absolute z-10 left-0 right-0 top-full mt-1 bg-surface-800 border border-surface-600 rounded-lg px-3 py-2 text-surface-500 text-xs">
							No users found.
						</div>
					{/if}
				</div>
			{/if}

			{#if adjSelectedUser}
				<div>
					<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-amount">Amount (bUEC)</label>
					<input
						id="adj-amount"
						type="number"
						bind:value={adjAmount}
						placeholder="e.g. 500 or -100"
						class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none"
					/>
				</div>
				<div>
					<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="adj-reason">Reason</label>
					<textarea
						id="adj-reason"
						bind:value={adjReason}
						rows={3}
						maxlength={200}
						placeholder="E.g. compensation for bug, contest prize…"
						class="textarea w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm resize-none focus:border-primary-500 outline-none"
					></textarea>
					<p class="text-surface-500 text-xs mt-1">{adjReason.length}/200</p>
				</div>
				<button
					onclick={submitAdjustment}
					disabled={adjLoading}
					class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50"
				>
					{adjLoading ? 'Adjusting…' : 'Apply Adjustment'}
				</button>
			{/if}
		</div>
		{#if adjResult}
			<p class="text-green-400 text-sm">{adjResult}</p>
		{/if}
		{#if adjError}
			<p class="text-red-400 text-sm">Error: {adjError}</p>
		{/if}
	</section>
</div>