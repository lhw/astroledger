<script lang="ts">
	import { onMount } from 'svelte';
	import { getMe, getMyPositions, getMyTrades, getMyBadges, logout, setActiveBadge } from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import type { Position, TradeWithMarket, Badge } from '$lib/types';

	let positions = $state<Position[]>([]);
	let trades = $state<TradeWithMarket[]>([]);
	let badges = $state<Badge[]>([]);
	let activeBadgeKey = $state<string>('');
	let badgeSaving = $state(false);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		if (!$isLoggedIn) { loading = false; return; }
		try {
			const [, posRes, tradeRes, badgeRes] = await Promise.all([
				Promise.resolve(), // placeholder
				getMyPositions(),
				getMyTrades(0),
				getMyBadges()
			]);
			positions = posRes;
			trades = tradeRes;
			badges = badgeRes;
			activeBadgeKey = $currentUser?.active_badge_key ?? '';
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
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
</script>

<svelte:head>
	<title>My Profile — ScolyMarket</title>
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
			{#if badges.length === 0}
				<div class="text-surface-400 text-sm py-4">
					No badges yet. Trade more to earn them — or visit the <a href="/fomo" class="text-primary-600 hover:underline">FOMO Store</a>!
				</div>
			{:else}
				<p class="text-[11px] text-surface-400 mb-3">Click a badge to display it on your comments. Click again to unset.</p>
				<div class="flex flex-wrap gap-3">
					{#each badges as badge}
						{@const isActive = activeBadgeKey === badge.badge_key}
						<button
							class="badge-pill tier-{badge.tier} {isActive ? 'badge-active' : ''} badge-btn"
							class:opacity-40={badgeSaving && !isActive}
							title={isActive ? `Active — click to unset` : badge.description}
							onclick={() => pickBadge(badge.badge_key)}
							disabled={badgeSaving}
						>
							<span class="badge-pip">
								{#if badge.tier === 5}★{:else if badge.tier === 4}◈{:else if badge.tier === 3}◆{:else if badge.tier === 2}●{:else}▲{/if}
							</span>
							<span class="badge-pill-title">{badge.title}</span>
							{#if isActive}
								<span class="badge-active-dot" title="Active">✓</span>
							{/if}
						</button>
					{/each}
				</div>
			{/if}
		</section>


		<!-- Positions -->
		<section class="mb-8">
			<!-- Open positions -->
			<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">Open Positions</h2>
			{#if openPositions.length === 0}
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
			{#if trades.length === 0}
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
/* ─── Badge pills for profile page ──────────────────────────────── */
.badge-pill {
	display: inline-flex;
	align-items: center;
	gap: 0.4rem;
	padding: 0.35rem 0.85rem 0.35rem 0.65rem;
	border-radius: 9999px;
	font-size: 0.75rem;
	font-weight: 700;
	transition: transform 0.15s ease, box-shadow 0.15s ease;
	cursor: default;
}
.badge-btn {
	cursor: pointer;
	border: none;
	background: inherit;
}
.badge-btn:hover { transform: scale(1.06); }
.badge-btn:not(.badge-active):hover { filter: brightness(0.92); }
.badge-active {
	outline: 2px solid currentColor;
	outline-offset: 2px;
	box-shadow: 0 0 0 4px rgba(99,102,241,0.15);
}
.badge-active-dot {
	font-size: 0.6rem;
	font-weight: 900;
	background: #22c55e;
	color: #fff;
	border-radius: 50%;
	width: 1rem;
	height: 1rem;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	margin-left: 0.1rem;
}
.badge-pip {
	font-size: 0.75rem;
	line-height: 1;
}
.badge-pill-title {
	letter-spacing: 0.02em;
}

/* T1 Common */
.badge-pill.tier-1 {
	background: #f5ede0;
	border: 1.5px solid #d4b896;
	color: #7a5030;
}
/* T2 Uncommon */
.badge-pill.tier-2 {
	background: linear-gradient(90deg, #fef3d0, #fde68a);
	border: 1.5px solid #e6c96b;
	color: #92400e;
}
/* T3 Rare */
.badge-pill.tier-3 {
	background: linear-gradient(90deg, #fef9e0, #fef3c0);
	border: 1.5px solid #f0c040;
	color: #6b2d06;
	box-shadow: 0 0 6px rgba(240,192,64,0.3);
}
/* T4 Epic */
.badge-pill.tier-4 {
	background: linear-gradient(90deg, #1c1008, #2d1c00);
	border: 1.5px solid #fbbf24;
	color: #fef3c7;
	box-shadow: 0 0 8px rgba(251,191,36,0.4);
	animation: pill-glow 3s ease-in-out infinite alternate;
}
@keyframes pill-glow {
	from { box-shadow: 0 0 6px rgba(251,191,36,0.35); }
	to   { box-shadow: 0 0 14px rgba(251,191,36,0.65); }
}
/* T5 Legendary */
.badge-pill.tier-5 {
	background: linear-gradient(90deg, #0d0d0d, #1a1200);
	border: 1.5px solid #ffd700;
	color: #ffd700;
	text-shadow: 0 0 6px rgba(255,215,0,0.5);
	box-shadow: 0 0 12px rgba(255,215,0,0.35);
	animation: pill-legendary 4s linear infinite;
}
@keyframes pill-legendary {
	from { box-shadow: 0 0 10px rgba(255,215,0,0.3), 0 0 0 1.5px #ffd700; }
	50%  { box-shadow: 0 0 18px rgba(200,100,255,0.4), 0 0 0 1.5px #c880ff; }
	to   { box-shadow: 0 0 10px rgba(255,215,0,0.3), 0 0 0 1.5px #ffd700; }
}
</style>
