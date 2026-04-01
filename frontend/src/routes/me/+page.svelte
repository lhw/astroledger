<script lang="ts">
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

	let dataLoaded = $state(false);

	$effect(() => {
		// Wait until auth state is resolved (not undefined) before fetching.
		if ($currentUser === undefined) return;

		if ($currentUser === null) {
			loading = false;
			return;
		}

		// Only fetch once per page load.
		if (dataLoaded) return;
		dataLoaded = true;

		activeBadgeKey = $currentUser.active_badge_key ?? '';
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

		Promise.allSettled(jobs);
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
				<div class="hangar-list">
					{#each badges as badge}
						{@const isActive = activeBadgeKey === badge.badge_key}
						<div class="hangar-row" class:active={isActive}>
							<!-- Tier icon -->
							<div class="hangar-icon tier-{badge.tier}">
								{#if badge.tier === 5}★{:else if badge.tier === 4}◈{:else if badge.tier === 3}◆{:else if badge.tier === 2}●{:else}▲{/if}
							</div>
							<!-- Info -->
							<div class="hangar-info">
								<div class="hangar-title">{badge.title}</div>
								<div class="hangar-desc">{badge.description}</div>
								<div class="hangar-date">{new Date(badge.awarded_at).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}</div>
							</div>
							<!-- Right: insurance pill + active toggle -->
							<div class="hangar-actions">
								{#if badge.insurance}
									<span class="ins-pip ins-{badge.insurance}" title="Insurance tier">
										{badge.insurance === '6w' ? '6 Weeks' : badge.insurance === '120w' ? '120 Weeks' : 'LTI'}
									</span>
								{:else if badge.purchasable}
									<span class="ins-pip ins-none" title="No insurance selected at purchase">No Ins.</span>

								{/if}
								<button
									class="hangar-btn"
									class:active={isActive}
									disabled={badgeSaving}
									onclick={() => pickBadge(badge.badge_key)}
									title={isActive ? 'Unset active badge' : 'Set as active badge'}
								>
									{isActive ? '✓ Active' : 'Set Active'}
								</button>
							</div>
						</div>
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
/* ─── Badge Hangar ────────────────────────────────────────── */
.hangar-list {
	display: flex;
	flex-direction: column;
	gap: 0;
	border-radius: 0.75rem;
	overflow: hidden;
	border: 1px solid #e5e0d8;
}
.hangar-row {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.75rem 1rem;
	background: #fafaf8;
	border-bottom: 1px solid #ede8e0;
	transition: background 0.15s ease;
}
.hangar-row:last-child { border-bottom: none; }
.hangar-row:hover { background: #f5f0e8; }
.hangar-row.active {
	background: #fffbf0;
	border-left: 3px solid #d4a017;
}
.hangar-icon {
	width: 2.5rem;
	height: 2.5rem;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.1rem;
	flex-shrink: 0;
	font-weight: 700;
}
.hangar-icon.tier-1 { background: #f5ede0; color: #b08050; border: 1.5px solid #d4b896; }
.hangar-icon.tier-2 { background: linear-gradient(135deg, #fde68a, #f59e0b); color: #78350f; border: 1.5px solid #f59e0b; }
.hangar-icon.tier-3 { background: linear-gradient(135deg, #fde047, #f59e0b, #ea580c); color: #fff; border: 1.5px solid #f59e0b; box-shadow: 0 0 8px rgba(245,158,11,0.3); }
.hangar-icon.tier-4 { background: linear-gradient(135deg, #fbbf24, #f59e0b); color: #1c1008; border: 1.5px solid #fde68a; box-shadow: 0 0 10px rgba(251,191,36,0.4); }
.hangar-icon.tier-5 { background: conic-gradient(from 0deg, #ff0080, #ff8c00, #ffd700, #00ff88, #ff0080); color: #fff; border: 1.5px solid rgba(255,255,255,0.5); box-shadow: 0 0 12px rgba(255,215,0,0.4); }
.hangar-info {
	flex: 1;
	min-width: 0;
}
.hangar-title {
	font-size: 0.82rem;
	font-weight: 700;
	color: #2d2620;
	line-height: 1.2;
}
.hangar-desc {
	font-size: 0.68rem;
	color: #8a7560;
	line-height: 1.4;
	margin-top: 0.1rem;
}
.hangar-date {
	font-size: 0.6rem;
	color: #b0a090;
	margin-top: 0.15rem;
	text-transform: uppercase;
	letter-spacing: 0.06em;
}
.hangar-actions {
	display: flex;
	align-items: center;
	gap: 0.45rem;
	flex-shrink: 0;
}
/* Insurance chips (owner-only cosmetic) */
.ins-pip {
	padding: 0.18rem 0.55rem;
	border-radius: 9999px;
	font-size: 0.58rem;
	font-weight: 700;
	letter-spacing: 0.04em;
	text-transform: uppercase;
	border: 1.5px solid;
	white-space: nowrap;
}
.ins-pip.ins-none { background: #f3f4f6; color: #9ca3af; border-color: #e5e7eb; }
.ins-pip.ins-earned { background: #f0fdf4; color: #16a34a; border-color: #86efac; }
.ins-pip.ins-6w { background: #f3e8ff; color: #7e22ce; border-color: #c084fc; }
.ins-pip.ins-120w { background: #fff7ed; color: #c2410c; border-color: #fb923c; }
.ins-pip.ins-lti { background: #fef2f2; color: #b91c1c; border-color: #f87171; }
/* Active toggle button */
.hangar-btn {
	padding: 0.28rem 0.65rem;
	border-radius: 0.4rem;
	font-size: 0.65rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.06em;
	border: 1.5px solid #d4b896;
	background: transparent;
	color: #8a7560;
	cursor: pointer;
	transition: all 0.15s ease;
	white-space: nowrap;
}
.hangar-btn:hover:not(:disabled) { background: #f5ede0; border-color: #b08050; color: #b08050; }
.hangar-btn.active { background: #d4a017; border-color: #d4a017; color: #fff; }
.hangar-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* ─── Dark mode overrides ─────────────────────────────────────────────── */
:root[data-theme='dark'] .hangar-list { border-color: #2e261c; }
:root[data-theme='dark'] .hangar-row {
	background: #161210;
	border-bottom-color: #2e261c;
}
:root[data-theme='dark'] .hangar-row:hover { background: #201a12; }
:root[data-theme='dark'] .hangar-row.active {
	background: #1e1800;
	border-left-color: #d4a017;
}
:root[data-theme='dark'] .hangar-icon.tier-1 { background: #2a1e10; color: #c8904a; border-color: #5a3e20; }
:root[data-theme='dark'] .hangar-title { color: #e8d5b8; }
:root[data-theme='dark'] .hangar-desc { color: #a08870; }
:root[data-theme='dark'] .hangar-date { color: #6e5e50; }
:root[data-theme='dark'] .ins-pip.ins-none { background: #1c1c1c; color: #9ca3af; border-color: #374151; }
:root[data-theme='dark'] .ins-pip.ins-earned { background: #061808; color: #4ade80; border-color: #166534; }
:root[data-theme='dark'] .ins-pip.ins-6w { background: #1e0a38; color: #c084fc; border-color: #7e22ce; }
:root[data-theme='dark'] .ins-pip.ins-120w { background: #200c00; color: #fb923c; border-color: #c2410c; }
:root[data-theme='dark'] .ins-pip.ins-lti { background: #200808; color: #f87171; border-color: #b91c1c; }
:root[data-theme='dark'] .hangar-btn {
	border-color: #4a3820;
	color: #a08870;
}
:root[data-theme='dark'] .hangar-btn:hover:not(:disabled) { background: #2a1e10; border-color: #c8904a; color: #c8904a; }
:root[data-theme='dark'] .hangar-btn.active { background: #d4a017; border-color: #d4a017; color: #fff; }
</style>
