<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getMyPositions,
		getMyTrades,
		getMyBadges,
		logout,
		setActiveBadge,
		listBotTokens,
		createBotToken,
		revokeBotToken,
		ApiClientError
	} from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import Tooltip from '$lib/components/Tooltip.svelte';
	import BadgePill from '$lib/components/BadgePill.svelte';
	import type { Position, TradeWithMarket, Badge, BotApiToken } from '$lib/types';

	let positions = $state<Position[]>([]);
	let trades = $state<TradeWithMarket[]>([]);
	let badges = $state<Badge[]>([]);
	let botTokens = $state<BotApiToken[]>([]);
	let activeBadgeKey = $state<string>('');
	let badgeSaving = $state(false);
	let loading = $state(true);
	let positionsLoading = $state(false);
	let tradesLoading = $state(false);
	let badgesLoading = $state(false);
	let botLoading = $state(false);
	let botCreateLoading = $state(false);
	let botError = $state('');
	let botName = $state('');
	let botCanTrade = $state(true);
	let createdTokenValue = $state('');
	let error = $state('');

	onMount(async () => {
		if (!$isLoggedIn) {
			loading = false;
			return;
		}

		activeBadgeKey = $currentUser?.active_badge_key ?? '';
		loading = false;
		error = '';

		positionsLoading = true;
		tradesLoading = true;
		badgesLoading = true;
		botLoading = true;

		const jobs = [
			(async () => {
				try {
					positions = await getMyPositions();
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					positionsLoading = false;
				}
			})(),
			(async () => {
				try {
					trades = await getMyTrades(0);
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					tradesLoading = false;
				}
			})(),
			(async () => {
				try {
					badges = await getMyBadges();
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					badgesLoading = false;
				}
			})(),
			(async () => {
				try {
					botTokens = await listBotTokens();
				} catch (e) {
					botError = e instanceof Error ? e.message : String(e);
				} finally {
					botLoading = false;
				}
			})()
		];

		await Promise.allSettled(jobs);
	});

	const openPositions = $derived(positions.filter(p => p.market_status !== 'resolved'));
	const resolvedPositions = $derived(positions.filter(p => p.market_status === 'resolved'));

	async function pickBadge(key: string) {
		// Toggle off if already active.
		const newKey = activeBadgeKey === key ? '' : key;
		badgeSaving = true;
		try {
			const res = await setActiveBadge(newKey);
			activeBadgeKey = res.active_badge_key;
		} catch {
			// ignore — UI stays consistent on failure
		} finally {
			badgeSaving = false;
		}
	}

	async function createToken() {
		botError = '';
		createdTokenValue = '';
		const name = botName.trim();
		if (name.length < 3 || name.length > 64) {
			botError = 'Token name must be between 3 and 64 characters.';
			return;
		}

		botCreateLoading = true;
		try {
			const created = await createBotToken({
				name,
				can_trade: botCanTrade
			});
			createdTokenValue = created.token;
			botName = '';
			botTokens = await listBotTokens();
		} catch (e) {
			botError = e instanceof ApiClientError ? e.message : e instanceof Error ? e.message : String(e);
		} finally {
			botCreateLoading = false;
		}
	}

	async function removeToken(id: number) {
		botError = '';
		try {
			await revokeBotToken(id);
			botTokens = botTokens.filter((t) => t.id !== id);
		} catch (e) {
			botError = e instanceof ApiClientError ? e.message : e instanceof Error ? e.message : String(e);
		}
	}

	async function copyCreatedToken() {
		if (!createdTokenValue) return;
		try {
			await navigator.clipboard.writeText(createdTokenValue);
		} catch {
			// Clipboard errors are non-fatal; token remains visible.
		}
	}
</script>

<svelte:head>
	<title>My Profile — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-10">
	{#if !$isLoggedIn}
		<div class="text-center py-16">
			<h1 class="text-2xl font-bold text-surface-900 mb-4">Not logged in</h1>
			<a href="/auth/login" class="btn preset-filled-primary-500 uppercase tracking-wider text-xs">Login with SCID</a>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading profile…</div>
	{:else}
		{@const user = $currentUser}

		<!-- Profile header -->
		{#if user}
			<div class="flex items-start justify-between mb-8 pb-6 border-b border-surface-200">
				<div class="flex items-start gap-4">
					<UserAvatar src={user.avatar_url} name={user.display_name} size={56} />
					<div>
						<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Profile</p>
						<h1 class="text-2xl font-bold text-surface-900">{user.display_name}</h1>
						<p class="text-surface-500 text-sm mt-1">
							Joined {new Date(user.created_at).toLocaleDateString()}
						</p>
						{#if user.is_moderator || user.is_admin}
							<span class="sc-tag mt-2 inline-flex">
								{user.is_admin ? 'Admin' : 'Moderator'}
							</span>
						{/if}
					</div>
				</div>
				<div class="text-right">
					<div class="text-3xl font-bold text-primary-600">{user.balance.toLocaleString()}</div>
					<div class="text-surface-500 text-xs uppercase tracking-widest font-semibold">bUEC</div>
					<button onclick={logout} class="border border-surface-300 text-surface-500 hover:border-surface-500 hover:text-surface-700 transition-colors rounded px-3 py-1 text-xs uppercase tracking-wider mt-3">
						Logout
					</button>
				</div>
			</div>

			<!-- RSI Identity -->
			{#if user.rsi_handle || user.is_rsi_verified}
				<section class="mb-8">
					<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">RSI Identity</h2>
					<div class="sc-card px-4 py-4 flex flex-wrap gap-6 items-center">
						{#if user.rsi_handle}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Handle</p>
								<a
									href="https://robertsspaceindustries.com/citizens/{user.rsi_handle}"
									target="_blank"
									rel="noopener noreferrer"
									class="text-primary-600 font-semibold hover:underline"
								>
									{user.rsi_handle}
								</a>
							</div>
						{/if}
						{#if user.rsi_citizen_record}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Citizen Record</p>
								<p class="text-surface-800 font-medium">#{user.rsi_citizen_record}</p>
							</div>
						{/if}
						{#if user.rsi_enlisted}
							<div>
								<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Enlisted</p>
								<p class="text-surface-800 font-medium">{user.rsi_enlisted}</p>
							</div>
						{/if}
						{#if user.is_rsi_verified}
							<div class="ml-auto">
								<span class="sc-tag text-green-700 bg-green-50 border-green-200">✓ RSI Verified</span>
							</div>
						{/if}
					</div>
				</section>
			{/if}
		{/if}

		{#if error}
			<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-6">
				{error}
			</div>
		{/if}

		<!-- Badges -->
		<section class="mb-8">
			<div class="flex items-center justify-between mb-1">
				<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">Badges</h2>
				<a href="/fomo" class="text-xs text-primary-600 hover:text-primary-800 font-semibold uppercase tracking-wider">FOMO Store →</a>
			</div>
			{#if badgesLoading}
				<div class="text-surface-400 text-sm py-4">Loading badges…</div>
			{:else if badges.length === 0}
				<div class="text-surface-400 text-sm py-4">
					No badges yet. Trade more to earn them — or visit the <a href="/fomo" class="text-primary-600 hover:underline">FOMO Store</a>!
				</div>
			{:else}
				<p class="text-[11px] text-surface-400 mb-3">Click a badge to display it on your comments. Click again to unset.</p>
				<div class="flex flex-wrap gap-3">
					{#each badges as badge}
						{@const isActive = activeBadgeKey === badge.badge_key}
						<Tooltip text={isActive ? 'Active — click to unset' : badge.description}>
							<button
								class="badge-btn"
								class:opacity-40={badgeSaving && !isActive}
								onclick={() => pickBadge(badge.badge_key)}
								disabled={badgeSaving}
							>
								<BadgePill tier={badge.tier} title={badge.title} active={isActive} showCheck={isActive} />
							</button>
						</Tooltip>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Bot API Tokens -->
		<section class="mb-8">
			<div class="flex items-center justify-between mb-2">
				<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">Bot API Tokens</h2>
			</div>
			<p class="text-[11px] text-surface-500 mb-3">
				Create scoped tokens for read/trade bot access. Tokens are shown once at creation.
			</p>

			<div class="sc-card p-4 mb-3 space-y-3">
				<div>
					<label class="block text-[10px] uppercase tracking-wider text-surface-500 mb-1" for="bot-token-name">Token Name</label>
					<input id="bot-token-name" bind:value={botName} class="sc-input text-sm" placeholder="e.g. Backtesting Bot" maxlength="64" />
				</div>

				<label class="flex items-center gap-2 text-sm text-surface-600">
					<input type="checkbox" bind:checked={botCanTrade} />
					Allow trading scope
				</label>

				<div class="flex items-center gap-2">
					<button onclick={createToken} disabled={botCreateLoading} class="btn btn-sm preset-filled-primary-500 uppercase tracking-wider text-xs">
						{botCreateLoading ? 'Creating…' : 'Create Token'}
					</button>
				</div>

				{#if createdTokenValue}
					<div class="rounded border border-primary-300 bg-primary-100/40 p-3 space-y-2">
						<p class="text-[11px] font-semibold text-primary-700 uppercase tracking-wider">Copy now - this value will not be shown again</p>
						<div class="text-xs break-all font-mono text-surface-800">{createdTokenValue}</div>
						<button onclick={copyCreatedToken} class="btn btn-xs preset-outlined uppercase tracking-wider text-[10px]">Copy Token</button>
					</div>
				{/if}
			</div>

			{#if botError}
				<div class="p-3 rounded border border-red-500/40 bg-red-500/10 text-red-400 text-sm mb-3">{botError}</div>
			{/if}

			{#if botLoading}
				<div class="text-surface-400 text-sm py-2">Loading tokens…</div>
			{:else if botTokens.length === 0}
				<div class="text-surface-400 text-sm py-2">No active bot tokens yet.</div>
			{:else}
				<div class="space-y-2">
					{#each botTokens as token}
						<div class="sc-card p-3 flex items-center justify-between gap-3">
							<div class="min-w-0">
								<p class="text-sm font-semibold text-surface-800 truncate">{token.name}</p>
								<p class="text-[11px] text-surface-500 font-mono">{token.token_prefix}...</p>
								<p class="text-[11px] text-surface-500 mt-1">
									{token.can_trade ? 'read + trade' : 'read only'} · created {new Date(token.created_at).toLocaleDateString()}
									{#if token.last_used_at} · used {new Date(token.last_used_at).toLocaleDateString()}{/if}
								</p>
							</div>
							<button onclick={() => removeToken(token.id)} class="btn btn-xs preset-outlined text-red-500 border-red-300 hover:bg-red-50 uppercase tracking-wider text-[10px]">
								Revoke
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</section>


		<!-- Positions -->
		<section class="mb-8">
			<!-- Open positions -->
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Open Positions</h2>
			{#if positionsLoading}
				<div class="text-surface-400 text-sm py-4">Loading positions…</div>
			{:else if openPositions.length === 0}
				<div class="text-surface-400 text-sm py-4">No open positions yet. Go make some bets!</div>
			{:else}
				<div class="space-y-2 mb-6">
					{#each openPositions as pos}
						<a
							href="/markets/{pos.market_id}"
							class="sc-card p-4 flex justify-between items-start gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block"
						>
							<div class="flex-1 min-w-0">
								<p class="text-surface-800 text-sm font-medium truncate">{pos.market_title}</p>
								<p class="text-surface-500 text-xs mt-0.5">
									<span class="font-semibold text-primary-700">{pos.shares}</span>
									<span class="text-surface-400 ml-1">{pos.outcome_label}</span>
								</p>
							</div>
							<div class="flex flex-col items-end gap-1 shrink-0">
								<span class="sc-tag-status">{pos.market_status}</span>
								{#if pos.cost_basis > 0}
									<span class="text-[10px] text-surface-400">Cost {pos.cost_basis.toLocaleString()} bUEC</span>
								{/if}
							</div>
						</a>
					{/each}
				</div>
			{/if}

			<!-- Resolved positions -->
			{#if resolvedPositions.length > 0}
				<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4 mt-2">Resolved Positions</h2>
				<div class="space-y-2">
					{#each resolvedPositions as pos}
						{@const won = pos.resolved_outcome_id !== null && pos.outcome_id === pos.resolved_outcome_id}
						{@const payout = won ? pos.shares * 100 : 0}
						{@const gain = payout - pos.cost_basis}
						<a
							href="/markets/{pos.market_id}"
							class="sc-card p-4 flex justify-between items-start gap-4 hover:border-primary-300 hover:shadow-md transition-all no-underline block {won ? 'border-l-2 border-l-green-500' : 'border-l-2 border-l-red-700'}"
						>
							<div class="flex-1 min-w-0">
								<p class="text-surface-800 text-sm font-medium truncate">{pos.market_title}</p>
								<p class="text-surface-500 text-xs mt-0.5">
									<span class="font-semibold text-primary-700">{pos.shares}</span>
									<span class="text-surface-400 ml-1">{pos.outcome_label}</span>
									<span class="ml-2 text-surface-500">· Cost {pos.cost_basis.toLocaleString()} bUEC</span>
								</p>
							</div>
							<div class="flex flex-col items-end gap-1 shrink-0">
								{#if won}
									<span class="text-xs font-bold text-green-400">+{payout.toLocaleString()} bUEC</span>
									<span class="text-[10px] {gain >= 0 ? 'text-green-600' : 'text-red-400'}">{gain >= 0 ? '+' : ''}{gain.toLocaleString()} net</span>
								{:else}
									<span class="text-xs font-bold text-red-400">−{pos.cost_basis.toLocaleString()} bUEC</span>
									<span class="text-[10px] text-surface-500">lost</span>
								{/if}
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Recent trades -->
		<section>
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Recent Trades</h2>
			{#if tradesLoading}
				<div class="text-surface-400 text-sm py-4">Loading trades…</div>
			{:else if trades.length === 0}
				<div class="text-surface-400 text-sm py-4">No trades yet.</div>
			{:else}
				<div class="sc-card overflow-hidden">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-surface-200">
								<th class="px-4 py-3 text-left text-surface-500 font-bold text-xs uppercase tracking-wider">Market</th>
								<th class="px-4 py-3 text-center text-surface-500 font-bold text-xs uppercase tracking-wider">Outcome</th>
								<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Shares</th>
								<th class="px-4 py-3 text-right text-surface-500 font-bold text-xs uppercase tracking-wider">Cost</th>
							</tr>
						</thead>
						<tbody>
							{#each trades as trade}
								<tr class="border-b border-surface-100 hover:bg-surface-50 transition-colors">
									<td class="px-4 py-3">
										<a
											href="/markets/{trade.market_id}"
											class="text-surface-800 hover:text-primary-600 truncate block max-w-[200px] transition-colors"
										>
											{trade.market_title}
										</a>
									</td>
									<td class="px-4 py-3 text-center">
										<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider bg-primary-100 text-primary-700 border border-primary-200">
											{trade.outcome_label}
										</span>
									</td>
									<td class="px-4 py-3 text-right text-surface-700 font-mono text-sm">{trade.shares}</td>
									<td class="px-4 py-3 text-right text-surface-700 font-mono text-sm">{trade.cost}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}
</div>

<style>
/* ─── Badge button wrapper (visual handled by BadgePill component) ─ */
.badge-btn {
	cursor: pointer;
	border: none;
	background: transparent;
	padding: 0;
	display: inline-flex;
	align-items: center;
	transition: transform 0.15s ease, filter 0.15s ease, opacity 0.15s ease;
}
.badge-btn:hover:not(:disabled) {
	transform: scale(1.06);
	filter: brightness(0.92);
}
</style>
