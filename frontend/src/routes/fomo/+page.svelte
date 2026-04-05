<script lang="ts">
	import { onMount } from 'svelte';
	import { getStoreBadges, purchaseBadge } from '$lib/api';
	import { formatBadgeInsurance, getBadgeTierLabel, getBadgeTierSymbol } from '$lib/badges';
	import Alert from '$lib/components/Alert.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { formatExpiry, formatSpend } from '$lib/format';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import type { StoreBadge } from '$lib/types';

	let badges = $state<StoreBadge[]>([]);
	let loading = $state(true);
	let purchasing = $state<string | null>(null);
	let error = $state('');
	let successKey = $state<string | null>(null);
	onMount(async () => {
		try {
			badges = await getStoreBadges();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});

	async function buy(badge: StoreBadge) {
		if (!$isLoggedIn) { error = 'You must be logged in to purchase badges.'; return; }
		if (badge.owned || badge.expired || purchasing) return;
		purchasing = badge.badge_key;
		error = '';
		successKey = null;
		try {
			await purchaseBadge(badge.badge_key, badge.insurance ?? '');
			badge.owned = true;
			successKey = badge.badge_key;
			if ($currentUser) {
				$currentUser = { ...$currentUser, balance: $currentUser.balance - badge.cost };
			}
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			purchasing = null;
		}
	}

	function isSoldOut(badge: StoreBadge): boolean {
		return badge.remaining_stock !== undefined && badge.remaining_stock <= 0;
	}

	function canBuy(badge: StoreBadge): boolean {
		if (badge.owned || badge.expired || isSoldOut(badge)) return false;
		if ($currentUser && $currentUser.balance < badge.cost) return false;
		return true;
	}
</script>

<svelte:head>
	<title>FOMO Store — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-4xl py-10">
	<!-- Header -->
	<div class="text-center mb-10">
		<p class="text-xs font-bold uppercase tracking-[0.2em] text-primary-500 mb-2">Cosmetic Badges</p>
		<h1 class="text-4xl font-bold text-surface-900 mb-3">FOMO Store</h1>
		<p class="text-surface-500 text-sm max-w-md mx-auto">
			Spend your hard-earned bUEC on exclusive cosmetic badges. Purely for clout. Absolutely no gameplay advantage. We promise.
		</p>
		{#if $isLoggedIn && $currentUser}
			<div class="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary-50 border border-primary-200">
				<span class="text-xs uppercase tracking-wider text-primary-600 font-semibold">Balance:</span>
				<span class="text-primary-700 font-bold">{formatSpend($currentUser.balance)}</span>
			</div>
		{/if}
	</div>

	{#if error}
		<Alert type="error" message={error} />
	{/if}

	{#if loading}
		<EmptyState message="Loading store…" card={false} padding="py-20" />
	{:else if badges.length === 0}
		<EmptyState message="No badges are currently available in the store." />
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each badges as badge}
				<!-- Badge card — tier determines visual treatment -->
				<div class="badge-card tier-{badge.tier}" class:owned={badge.owned} class:unavailable={badge.expired || isSoldOut(badge)}>

					<!-- Scarcity banners (time-limited / limited stock) -->
					{#if badge.expired}
						<div class="scarcity-banner expired">EXPIRED</div>
					{:else if isSoldOut(badge)}
						<div class="scarcity-banner soldout">SOLD OUT</div>
					{:else}
						{@const showExpiry = badge.available_until != null}
						{@const showStock = badge.remaining_stock != null && badge.stock != null}
						{#if showExpiry || showStock}
							<div class="scarcity-row">
								{#if showExpiry && badge.available_until}
									<span class="scarcity-chip time" title="Limited-time offer">
										⏳ {formatExpiry(badge.available_until)}
									</span>
								{/if}
								{#if showStock && badge.remaining_stock != null && badge.stock != null}
									{@const pct = badge.remaining_stock / badge.stock}
									<span class="scarcity-chip stock" class:low={pct <= 0.25} title="Limited stock">
										{#if pct <= 0.1}🔥{:else}📦{/if}
										{badge.remaining_stock} / {badge.stock} left
									</span>
								{/if}
							</div>
						{/if}
					{/if}

					<div class="badge-icon-wrap">
						<div class="badge-icon">
							<span class="badge-symbol">{getBadgeTierSymbol(badge.tier)}</span>
						</div>
					</div>

					<div class="badge-body">
						<div class="badge-tier-label">{getBadgeTierLabel(badge.tier)}</div>
						<h3 class="badge-title">{badge.title}</h3>
						<p class="badge-desc">{badge.description}</p>
						{#if badge.insurance}
							<div class="badge-ins-row">
								<span class="badge-ins-chip ins-{badge.insurance}">
									{formatBadgeInsurance(badge.insurance)}
								</span>
							</div>
						{/if}
					</div>

					<div class="badge-footer">
						<span class="badge-price">{formatSpend(badge.cost)}</span>
						{#if badge.owned}
							<span class="badge-owned-tag">Owned</span>
						{:else if badge.expired}
							<span class="badge-unavailable-tag">Expired</span>
						{:else if isSoldOut(badge)}
							<span class="badge-unavailable-tag">Sold Out</span>
						{:else if !$isLoggedIn}
							<a href="/auth/login" class="badge-btn">Login to Buy</a>
						{:else}
							<button
								class="badge-btn"
								disabled={purchasing === badge.badge_key || !canBuy(badge)}
								onclick={() => buy(badge)}
							>
								{purchasing === badge.badge_key ? 'Buying…' : 'Buy'}
							</button>
						{/if}
					</div>
					{#if successKey === badge.badge_key}
						<div class="badge-success">✓ Added to your profile!</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
/* ─── Base badge card ───────────────────────────────────────────────── */
.badge-card {
	position: relative;
	border-radius: 1rem;
	padding: 1.5rem 1.25rem 1.25rem;
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
	overflow: hidden;
	transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.badge-card:hover {
	transform: translateY(-3px);
}

/* ─── T1 Common: clean flat card, muted gold border ────────────────── */
.badge-card.tier-1 {
	background: #fafaf9;
	border: 1.5px solid #d4b896;
	box-shadow: 0 1px 4px rgba(0,0,0,0.06);
}
.badge-card.tier-1 .badge-icon {
	background: #f5ede0;
	color: #b08050;
	border: 2px solid #d4b896;
}
.badge-card.tier-1 .badge-tier-label { color: #b08050; }
.badge-card.tier-1 .badge-title { color: #3d2e1a; }
.badge-card.tier-1 .badge-btn {
	background: #b08050;
	color: #fff;
}
.badge-card.tier-1 .badge-btn:hover:not(:disabled) {
	background: #8a6138;
}

/* ─── T2 Uncommon: warm gradient, shadow ───────────────────────────── */
.badge-card.tier-2 {
	background: linear-gradient(135deg, #fffbf0 0%, #fef3d0 100%);
	border: 1.5px solid #e6c96b;
	box-shadow: 0 2px 10px rgba(200,160,40,0.15);
}
.badge-card.tier-2 .badge-icon {
	background: linear-gradient(135deg, #fde68a, #f59e0b);
	color: #78350f;
	border: 2px solid #f59e0b;
}
.badge-card.tier-2 .badge-tier-label { color: #b45309; }
.badge-card.tier-2 .badge-title { color: #78350f; }
.badge-card.tier-2 .badge-btn {
	background: linear-gradient(90deg, #f59e0b, #d97706);
	color: #fff;
}
.badge-card.tier-2 .badge-btn:hover:not(:disabled) {
	background: linear-gradient(90deg, #d97706, #b45309);
}

/* ─── T3 Rare: shimmering gold border, deeper shadow ───────────────── */
.badge-card.tier-3 {
	background: linear-gradient(135deg, #fffdf0 0%, #fef9e0 50%, #fef3c0 100%);
	border: 2px solid transparent;
	background-clip: padding-box;
	box-shadow: 0 4px 20px rgba(180,140,20,0.25), inset 0 0 0 2px #f0c040;
}
.badge-card.tier-3::before {
	content: '';
	position: absolute;
	inset: 0;
	border-radius: 1rem;
	padding: 2px;
	background: linear-gradient(135deg, #f7d060, #e8a020, #f7d060, #c8800a);
	mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
	-webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
	-webkit-mask-composite: xor;
	mask-composite: exclude;
	pointer-events: none;
}
.badge-card.tier-3 .badge-icon {
	background: linear-gradient(135deg, #fde047, #f59e0b, #ea580c);
	color: #fff;
	border: 2px solid #f59e0b;
	box-shadow: 0 0 10px rgba(245,158,11,0.4);
}
.badge-card.tier-3 .badge-tier-label { color: #92400e; }
.badge-card.tier-3 .badge-title { color: #6b2d06; }
.badge-card.tier-3 .badge-btn {
	background: linear-gradient(90deg, #f59e0b, #ea580c);
	color: #fff;
	box-shadow: 0 2px 8px rgba(245,158,11,0.4);
}
.badge-card.tier-3 .badge-btn:hover:not(:disabled) {
	background: linear-gradient(90deg, #d97706, #c2410c);
}

/* ─── T4 Epic: glowing card with animated pulse ─────────────────────── */
.badge-card.tier-4 {
	background: linear-gradient(135deg, #1c1008 0%, #2d1c00 50%, #3d2800 100%);
	border: 2px solid #fbbf24;
	box-shadow: 0 0 20px rgba(251,191,36,0.35), 0 0 60px rgba(251,191,36,0.1);
	animation: epic-glow 3s ease-in-out infinite alternate;
}
@keyframes epic-glow {
	from { box-shadow: 0 0 15px rgba(251,191,36,0.3), 0 0 40px rgba(251,191,36,0.08); }
	to   { box-shadow: 0 0 30px rgba(251,191,36,0.55), 0 0 80px rgba(251,191,36,0.18); }
}
.badge-card.tier-4 .badge-icon {
	background: linear-gradient(135deg, #fbbf24, #f59e0b, #d97706);
	color: #1c1008;
	border: 2px solid #fde68a;
	box-shadow: 0 0 18px rgba(251,191,36,0.6);
	animation: icon-pulse 2.5s ease-in-out infinite alternate;
}
@keyframes icon-pulse {
	from { box-shadow: 0 0 12px rgba(251,191,36,0.5); }
	to   { box-shadow: 0 0 24px rgba(251,191,36,0.85); }
}
.badge-card.tier-4 .badge-tier-label { color: #fde68a; }
.badge-card.tier-4 .badge-title { color: #fef3c7; }
.badge-card.tier-4 .badge-desc { color: #d6a84a; }
.badge-card.tier-4 .badge-price { color: #fbbf24; }
.badge-card.tier-4 .badge-btn {
	background: linear-gradient(90deg, #fbbf24, #f59e0b);
	color: #1c1008;
	font-weight: 700;
	box-shadow: 0 0 10px rgba(251,191,36,0.5);
}
.badge-card.tier-4 .badge-btn:hover:not(:disabled) {
	background: linear-gradient(90deg, #fde68a, #fbbf24);
	box-shadow: 0 0 18px rgba(251,191,36,0.7);
}

/* ─── T5 Legendary: holographic rainbow animation ───────────────────── */
.badge-card.tier-5 {
	background: linear-gradient(135deg, #0d0d0d 0%, #1a1200 50%, #0d0a00 100%);
	border: 2px solid transparent;
	position: relative;
	box-shadow:
		0 0 0 2px rgba(255,215,0,0.8),
		0 0 40px rgba(255,215,0,0.3),
		0 0 80px rgba(255,100,0,0.15),
		0 0 120px rgba(100,0,255,0.1);
	animation: legendary-halo 4s linear infinite;
}
@keyframes legendary-halo {
	0%   { box-shadow: 0 0 0 2px rgba(255,215,0,0.8),  0 0 40px rgba(255,215,0,0.3),  0 0 80px rgba(255,100,0,0.15); }
	25%  { box-shadow: 0 0 0 2px rgba(255,150,0,0.9),  0 0 50px rgba(255,150,0,0.35), 0 0 90px rgba(255,50,0,0.2); }
	50%  { box-shadow: 0 0 0 2px rgba(200,100,255,0.8), 0 0 40px rgba(100,50,255,0.3), 0 0 80px rgba(50,0,200,0.2); }
	75%  { box-shadow: 0 0 0 2px rgba(50,200,255,0.8),  0 0 50px rgba(0,150,255,0.3),  0 0 90px rgba(0,100,200,0.2); }
	100% { box-shadow: 0 0 0 2px rgba(255,215,0,0.8),  0 0 40px rgba(255,215,0,0.3),  0 0 80px rgba(255,100,0,0.15); }
}
.badge-card.tier-5 .badge-icon {
	background: conic-gradient(from 0deg, #ff0080, #ff8c00, #ffd700, #00ff88, #00cfff, #8b00ff, #ff0080);
	color: #fff;
	border: 2px solid rgba(255,255,255,0.6);
	box-shadow: 0 0 30px rgba(255,215,0,0.7);
	animation: holo-spin 6s linear infinite;
	background-size: 400% 400%;
}
@keyframes holo-spin {
	from { filter: hue-rotate(0deg) brightness(1.1); }
	to   { filter: hue-rotate(360deg) brightness(1.4); }
}
.badge-card.tier-5 .badge-tier-label {
	background: linear-gradient(90deg, #ffd700, #ff8c00, #ff0080, #8b00ff, #00cfff, #ffd700);
	background-size: 400% 100%;
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
	animation: rainbow-slide 3s linear infinite;
	font-weight: 900;
	letter-spacing: 0.2em;
}
@keyframes rainbow-slide {
	from { background-position: 0% 50%; }
	to   { background-position: 400% 50%; }
}
.badge-card.tier-5 .badge-title {
	color: #ffd700;
	text-shadow: 0 0 10px rgba(255,215,0,0.5);
	font-size: 1.1rem;
}
.badge-card.tier-5 .badge-desc { color: #c8a44a; }
.badge-card.tier-5 .badge-price { color: #ffd700; }
.badge-card.tier-5 .badge-btn {
	background: conic-gradient(from 0deg, #ff0080, #ff8c00, #ffd700, #00ff88, #ff0080);
	color: #fff;
	font-weight: 800;
	text-shadow: 0 1px 3px rgba(0,0,0,0.5);
	box-shadow: 0 0 20px rgba(255,215,0,0.5);
	animation: btn-holo 4s linear infinite;
}
@keyframes btn-holo {
	from { filter: hue-rotate(0deg); }
	to   { filter: hue-rotate(360deg); }
}
.badge-card.tier-5 .badge-btn:hover:not(:disabled) {
	box-shadow: 0 0 30px rgba(255,215,0,0.8);
}

/* ─── Owned state overlay ────────────────────────────────────────────── */
.badge-card.owned {
	opacity: 0.75;
}
.badge-card.owned::after {
	content: '✓';
	position: absolute;
	top: 0.6rem;
	right: 0.75rem;
	font-size: 0.75rem;
	font-weight: 900;
	color: #22c55e;
}

/* ─── Unavailable (expired/sold-out) dim ─────────────────────────────── */
.badge-card.unavailable {
	opacity: 0.55;
	filter: grayscale(60%);
}

/* ─── Scarcity banners ───────────────────────────────────────────────── */
.scarcity-banner {
	text-align: center;
	font-size: 0.65rem;
	font-weight: 900;
	letter-spacing: 0.18em;
	text-transform: uppercase;
	padding: 0.25rem 0.5rem;
	border-radius: 0.4rem;
	margin-bottom: 0.25rem;
}
.scarcity-banner.expired {
	background: #fee2e2;
	color: #b91c1c;
	border: 1px solid #fca5a5;
}
.scarcity-banner.soldout {
	background: #f3f4f6;
	color: #6b7280;
	border: 1px solid #d1d5db;
}
.scarcity-row {
	display: flex;
	flex-wrap: wrap;
	gap: 0.3rem;
	margin-bottom: 0.5rem;
}
.scarcity-chip {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.2rem 0.55rem;
	border-radius: 9999px;
	font-size: 0.65rem;
	font-weight: 700;
	letter-spacing: 0.05em;
}
.scarcity-chip.time {
	background: #fffbeb;
	border: 1px solid #fcd34d;
	color: #92400e;
}
.scarcity-chip.stock {
	background: #f0fdf4;
	border: 1px solid #86efac;
	color: #14532d;
}
.scarcity-chip.stock.low {
	background: #fff7ed;
	border: 1px solid #fb923c;
	color: #9a3412;
	animation: stock-pulse 2s ease-in-out infinite alternate;
}
@keyframes stock-pulse {
	from { background: #fff7ed; }
	to   { background: #fee2e2; }
}

/* ─── Subcomponents ──────────────────────────────────────────────────── */
.badge-icon-wrap {
	display: flex;
	justify-content: center;
}
.badge-icon {
	width: 3.5rem;
	height: 3.5rem;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
}
.badge-symbol {
	font-size: 1.5rem;
	line-height: 1;
}
.badge-body {
	text-align: center;
	flex: 1;
}
.badge-tier-label {
	font-size: 0.6rem;
	font-weight: 800;
	text-transform: uppercase;
	letter-spacing: 0.18em;
	margin-bottom: 0.25rem;
}
.badge-title {
	font-size: 0.95rem;
	font-weight: 700;
	margin-bottom: 0.3rem;
}
.badge-desc {
	font-size: 0.72rem;
	color: #6b7280;
	line-height: 1.4;
}
.badge-footer {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 0.5rem;
	margin-top: 0.25rem;
}
.badge-price {
	font-size: 0.8rem;
	font-weight: 700;
	color: #374151;
}
.badge-btn {
	padding: 0.35rem 0.9rem;
	border-radius: 0.5rem;
	font-size: 0.72rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.08em;
	border: none;
	cursor: pointer;
	transition: all 0.18s ease;
	text-decoration: none;
	display: inline-block;
}
.badge-btn:disabled {
	opacity: 0.45;
	cursor: not-allowed;
}
.badge-owned-tag {
	padding: 0.3rem 0.8rem;
	border-radius: 0.5rem;
	font-size: 0.7rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.08em;
	background: #dcfce7;
	color: #16a34a;
	border: 1px solid #86efac;
}
.badge-unavailable-tag {
	padding: 0.3rem 0.8rem;
	border-radius: 0.5rem;
	font-size: 0.7rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.08em;
	background: #f3f4f6;
	color: #6b7280;
	border: 1px solid #d1d5db;
}
.badge-success {
	text-align: center;
	font-size: 0.7rem;
	font-weight: 700;
	color: #16a34a;
	animation: fade-in 0.3s ease;
}
@keyframes fade-in {
	from { opacity: 0; transform: translateY(4px); }
	to   { opacity: 1; transform: translateY(0); }
}

/* ─── Insurance display chip ─────────────────────────────────────────── */
.badge-ins-row {
	display: flex;
	justify-content: center;
	margin-top: 0.4rem;
}
.badge-ins-chip {
	padding: 0.18rem 0.6rem;
	border-radius: 9999px;
	font-size: 0.6rem;
	font-weight: 700;
	letter-spacing: 0.06em;
	text-transform: uppercase;
	border: 1.5px solid;
}
.badge-ins-chip.ins-6w { background: #f3e8ff; color: #7e22ce; border-color: #c084fc; }
.badge-ins-chip.ins-120w { background: #fff7ed; color: #c2410c; border-color: #fb923c; }
.badge-ins-chip.ins-lti { background: #fef2f2; color: #b91c1c; border-color: #f87171; }

/* ─── Dark mode overrides ─────────────────────────────────────────────── */
:root[data-theme='dark'] .badge-card.tier-1 {
	background: #1a1510;
	border-color: #4a3820;
	box-shadow: 0 1px 4px rgba(0,0,0,0.4);
}
:root[data-theme='dark'] .badge-card.tier-1 .badge-icon {
	background: #2a1e10;
	color: #c8904a;
	border-color: #5a3e20;
}
:root[data-theme='dark'] .badge-card.tier-1 .badge-tier-label { color: #c8904a; }
:root[data-theme='dark'] .badge-card.tier-1 .badge-title { color: #e8d5b8; }
:root[data-theme='dark'] .badge-card.tier-2 {
	background: linear-gradient(135deg, #1a1508 0%, #201a00 100%);
	border-color: #5a4010;
	box-shadow: 0 2px 10px rgba(100,70,0,0.3);
}
:root[data-theme='dark'] .badge-card.tier-2 .badge-tier-label { color: #d4a020; }
:root[data-theme='dark'] .badge-card.tier-2 .badge-title { color: #e8c060; }
:root[data-theme='dark'] .badge-card.tier-3 {
	background: linear-gradient(135deg, #1a1500 0%, #201800 50%, #201400 100%);
	box-shadow: 0 4px 20px rgba(120,90,0,0.35), inset 0 0 0 2px #6a5010;
}
:root[data-theme='dark'] .badge-card.tier-3 .badge-tier-label { color: #c89030; }
:root[data-theme='dark'] .badge-card.tier-3 .badge-title { color: #e0b040; }
:root[data-theme='dark'] .badge-desc { color: #a08870; }
:root[data-theme='dark'] .badge-price { color: #c8a060; }
:root[data-theme='dark'] .badge-owned-tag {
	background: #07200f;
	color: #4ade80;
	border-color: #166534;
}
:root[data-theme='dark'] .badge-unavailable-tag {
	background: #1c1c1c;
	color: #9ca3af;
	border-color: #374151;
}
:root[data-theme='dark'] .scarcity-banner.expired {
	background: #200808;
	color: #f87171;
	border-color: #7f1d1d;
}
:root[data-theme='dark'] .scarcity-banner.soldout {
	background: #1c1c1c;
	color: #9ca3af;
	border-color: #374151;
}
:root[data-theme='dark'] .scarcity-chip.time {
	background: #201800;
	border-color: #78400a;
	color: #fcd34d;
}
:root[data-theme='dark'] .scarcity-chip.stock {
	background: #061808;
	border-color: #14532d;
	color: #86efac;
}
:root[data-theme='dark'] .scarcity-chip.stock.low {
	background: #1a0c00;
	border-color: #9a3412;
	color: #fb923c;
}
:root[data-theme='dark'] .badge-ins-chip.ins-6w { background: #1e0a38; color: #c084fc; border-color: #7e22ce; }
:root[data-theme='dark'] .badge-ins-chip.ins-120w { background: #200c00; color: #fb923c; border-color: #c2410c; }
:root[data-theme='dark'] .badge-ins-chip.ins-lti { background: #200808; color: #f87171; border-color: #b91c1c; }
</style>
