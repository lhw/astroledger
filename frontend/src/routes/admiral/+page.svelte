<script lang="ts">
	import { onMount } from 'svelte';
	import { getAdmiralRanks } from '$lib/api';
	import Alert from '$lib/components/Alert.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { formatSpend } from '$lib/format';
	import { isLoggedIn } from '$lib/stores/auth';
	import type { AdmiralRank } from '$lib/types';

	let ranks = $state<AdmiralRank[]>([]);
	let lifetimeSpend = $state(0);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const data = await getAdmiralRanks();
			ranks = data.ranks;
			lifetimeSpend = data.lifetime_spend;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});

	// The highest rank threshold for the progress bar ceiling.
	const maxThreshold = $derived(ranks.length > 0 ? ranks[ranks.length - 1].spend_threshold : 1);

	// Current rank = highest owned rank.
	const currentRank = $derived(
		ranks.filter((r) => r.owned).at(-1) ?? null
	);

	// Next rank = first not yet owned.
	const nextRank = $derived(ranks.find((r) => !r.owned) ?? null);

	// Progress toward next rank (0–100).
	const progress = $derived(
		nextRank
			? Math.min(100, Math.round((lifetimeSpend / nextRank.spend_threshold) * 100))
			: 100
	);

	// Insignia symbols per tier (distinct from FOMO store).
	const insignia: Record<number, string> = {
		1: '⚓',
		2: '⚔',
		3: '🛡',
		4: '👑',
		5: '🌟'
	};

</script>

<svelte:head>
	<title>Rank — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-2xl py-10">
	<!-- Header -->
	<div class="text-center mb-10">
		<p class="text-xs font-bold uppercase tracking-[0.2em] text-primary-500 mb-2">Lifetime Spend</p>
		<h1 class="text-4xl font-bold text-surface-900 mb-3">Admiral Rank</h1>
		<p class="text-surface-500 text-sm max-w-md mx-auto">
			Spend bUEC in the FOMO Store to rise through the ranks. Ranks are awarded automatically — no purchase required beyond the badges themselves.
		</p>
	</div>

	{#if error}
		<Alert type="error" message={error} />
	{/if}

	{#if loading}
		<EmptyState message="Loading ranks…" card={false} padding="py-20" />
	{:else if ranks.length === 0}
		<EmptyState message="No admiral ranks are configured yet." />
	{:else}
		<!-- Current status card -->
		<div class="rank-status-card mb-10">
			<div class="flex items-center justify-between mb-2 flex-wrap gap-2">
				<div>
					{#if currentRank}
						<p class="rank-status-label">Current Rank</p>
						<p class="rank-status-title">{currentRank.title}</p>
					{:else}
						<p class="rank-status-label">Current Rank</p>
						<p class="rank-status-title rank-unranked">Unranked</p>
					{/if}
				</div>
				<div class="text-right">
					<p class="rank-status-label">Lifetime Spend</p>
					<p class="rank-status-spend">
						{#if $isLoggedIn}
							{formatSpend(lifetimeSpend)}
						{:else}
							<a href="/auth/login" class="text-primary-600 hover:underline text-sm">Log in to see your rank</a>
						{/if}
					</p>
				</div>
			</div>

			{#if $isLoggedIn}
				<!-- Progress bar toward next rank -->
				<div class="progress-track" title="{progress}% to {nextRank?.title ?? 'max rank'}">
					<div class="progress-fill" style="width: {progress}%"></div>
					<!-- Rank milestone ticks -->
					{#each ranks as rank}
						{@const pct = Math.min(100, (rank.spend_threshold / maxThreshold) * 100)}
						<div class="progress-tick" class:reached={rank.owned} style="left: {pct}%">
							<div class="progress-tick-dot"></div>
						</div>
					{/each}
				</div>
				<div class="flex justify-between text-[10px] text-surface-400 mt-1.5">
					<span>0</span>
					{#if nextRank && !currentRank}
						<span>{formatSpend(nextRank.spend_threshold)} to reach {nextRank.title}</span>
					{:else if nextRank && currentRank}
						<span>{formatSpend(nextRank.spend_threshold - lifetimeSpend)} more to {nextRank.title}</span>
					{:else}
						<span class="text-primary-600 font-bold">Maximum rank achieved!</span>
					{/if}
					<span>{formatSpend(maxThreshold)}</span>
				</div>
			{/if}
		</div>

		<!-- Rank cards — vertical timeline -->
		<div class="ranks-timeline">
			{#each ranks as rank, i}
				{@const isLast = i === ranks.length - 1}
				<div class="rank-row">
					<!-- Connector line -->
					{#if !isLast}
						<div class="rank-connector" class:reached={rank.owned}></div>
					{/if}

					<!-- Badge card -->
					<div class="rank-card tier-{rank.tier}" class:owned={rank.owned} class:locked={!rank.owned}>
						<!-- Left: insignia -->
						<div class="rank-insignia tier-{rank.tier}">
							<span class="rank-insignia-symbol">{insignia[rank.tier]}</span>
							{#if rank.owned}
								<div class="rank-check">✓</div>
							{/if}
						</div>

						<!-- Center: info -->
						<div class="rank-info">
							<div class="rank-tier-label tier-{rank.tier}">{rank.title}</div>
							<div class="rank-desc">{rank.description}</div>
						</div>

						<!-- Right: threshold -->
						<div class="rank-threshold">
							<span class="rank-threshold-amount">{formatSpend(rank.spend_threshold)}</span>
							<span class="rank-threshold-label">total spend</span>
						</div>
					</div>
				</div>
			{/each}
		</div>

		<!-- Footer note -->
		<p class="text-center text-surface-400 text-xs mt-10">
			Ranks are awarded instantly when you hit the spend threshold. Visit the
			<a href="/fomo" class="text-primary-600 hover:underline">FOMO Store</a> to start spending.
		</p>
	{/if}
</div>

<style>
/* ─── Status card ───────────────────────────────────────────────────── */
.rank-status-card {
	background: linear-gradient(135deg, #fff6df 0%, #f3e1b4 100%);
	border: 1.5px solid #d7b15d;
	border-radius: 1rem;
	padding: 1.5rem;
	color: #3f2f12;
}
.rank-status-label {
	font-size: 0.6rem;
	text-transform: uppercase;
	letter-spacing: 0.18em;
	color: #9a6f1f;
	font-weight: 700;
	margin-bottom: 0.15rem;
}
.rank-status-title {
	font-size: 1.4rem;
	font-weight: 800;
	color: #8c6210;
}
.rank-status-title.rank-unranked {
	color: #8a7a63;
	font-style: italic;
}
.rank-status-spend {
	font-size: 1.1rem;
	font-weight: 700;
	color: #4e3917;
}

/* ─── Progress bar ──────────────────────────────────────────────────── */
.progress-track {
	position: relative;
	height: 10px;
	background: #d9c39a;
	border-radius: 9999px;
	margin-top: 1rem;
	overflow: visible;
}
.progress-fill {
	height: 100%;
	border-radius: 9999px;
	background: linear-gradient(90deg, #c8880a, #f5d060, #ffd700);
	transition: width 0.6s ease;
	box-shadow: 0 0 10px rgba(245,208,96,0.4);
}
.progress-tick {
	position: absolute;
	top: 50%;
	transform: translate(-50%, -50%);
	display: flex;
	flex-direction: column;
	align-items: center;
}
.progress-tick-dot {
	width: 10px;
	height: 10px;
	border-radius: 50%;
	background: #d7bf8d;
	border: 2px solid #9f7a2c;
	transition: all 0.3s ease;
}
.progress-tick.reached .progress-tick-dot {
	background: #ffd700;
	border-color: #ffd700;
	box-shadow: 0 0 6px rgba(255,215,0,0.7);
}

/* ─── Rank timeline ─────────────────────────────────────────────────── */
.ranks-timeline {
	display: flex;
	flex-direction: column;
	gap: 0;
}
.rank-row {
	position: relative;
	padding-bottom: 0;
}
.rank-connector {
	position: absolute;
	left: 2.25rem;
	top: 5.5rem;
	bottom: 0;
	width: 2px;
	background: #e5e7eb;
	z-index: 0;
}
.rank-connector.reached {
	background: linear-gradient(to bottom, #f5d060, #c8880a);
}

/* ─── Rank card ─────────────────────────────────────────────────────── */
.rank-card {
	position: relative;
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem 1.25rem;
	border-radius: 0.875rem;
	margin-bottom: 1rem;
	transition: transform 0.15s ease, box-shadow 0.15s ease;
	z-index: 1;
}
.rank-card.owned:hover {
	transform: translateX(4px);
}

/* T1 Ensign — naval blue */
.rank-card.tier-1.owned {
	background: linear-gradient(90deg, #e8f0ff, #dce8ff);
	border: 1.5px solid #93b4f0;
	box-shadow: 0 2px 8px rgba(100,140,240,0.15);
}
.rank-card.tier-1.locked {
	background: #f8f9fa;
	border: 1.5px solid #e5e7eb;
}

/* T2 Lieutenant — silver-green */
.rank-card.tier-2.owned {
	background: linear-gradient(90deg, #e8fff0, #d0f5e0);
	border: 1.5px solid #5dbf8a;
	box-shadow: 0 2px 10px rgba(80,180,130,0.2);
}
.rank-card.tier-2.locked {
	background: #f8f9fa;
	border: 1.5px solid #e5e7eb;
}

/* T3 Commander — burnished copper */
.rank-card.tier-3.owned {
	background: linear-gradient(90deg, #fff8e8, #fef3d0);
	border: 1.5px solid #d4a040;
	box-shadow: 0 3px 12px rgba(200,150,40,0.2);
}
.rank-card.tier-3.locked {
	background: #f8f9fa;
	border: 1.5px solid #e5e7eb;
}

/* T4 Captain — deep gold */
.rank-card.tier-4.owned {
	background: linear-gradient(90deg, #1c1008, #2d1c00);
	border: 2px solid #fbbf24;
	box-shadow: 0 0 20px rgba(251,191,36,0.3);
	animation: captain-glow 3s ease-in-out infinite alternate;
}
@keyframes captain-glow {
	from { box-shadow: 0 0 12px rgba(251,191,36,0.25); }
	to   { box-shadow: 0 0 28px rgba(251,191,36,0.5); }
}
.rank-card.tier-4.locked {
	background: #f8f9fa;
	border: 1.5px solid #e5e7eb;
}

/* T5 Fleet Admiral — holographic */
.rank-card.tier-5.owned {
	background: linear-gradient(90deg, #0a0a08, #1a1200);
	border: 2px solid #ffd700;
	box-shadow: 0 0 30px rgba(255,215,0,0.4), 0 0 60px rgba(255,100,0,0.1);
	animation: admiral-halo 4s linear infinite;
}
@keyframes admiral-halo {
	0%   { box-shadow: 0 0 25px rgba(255,215,0,0.35), 0 0 50px rgba(255,100,0,0.1); }
	33%  { box-shadow: 0 0 30px rgba(255,150,0,0.4),  0 0 60px rgba(200,0,255,0.1); }
	66%  { box-shadow: 0 0 25px rgba(100,200,255,0.3), 0 0 50px rgba(0,100,255,0.1); }
	100% { box-shadow: 0 0 25px rgba(255,215,0,0.35), 0 0 50px rgba(255,100,0,0.1); }
}
.rank-card.tier-5.locked {
	background: #f8f9fa;
	border: 1.5px solid #e5e7eb;
}

/* ─── Insignia ──────────────────────────────────────────────────────── */
.rank-insignia {
	position: relative;
	width: 3rem;
	height: 3rem;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	font-size: 1.4rem;
	filter: saturate(0.3) brightness(0.7);
}
.rank-card.owned .rank-insignia {
	filter: none;
}
.rank-insignia.tier-1 { background: #dce8ff; border: 2px solid #93b4f0; }
.rank-insignia.tier-2 { background: #d0f5e0; border: 2px solid #5dbf8a; }
.rank-insignia.tier-3 { background: linear-gradient(135deg, #fde68a, #f59e0b); border: 2px solid #d4a040; }
.rank-insignia.tier-4 { background: linear-gradient(135deg, #fbbf24, #d97706); border: 2px solid #fbbf24; box-shadow: 0 0 10px rgba(251,191,36,0.4); }
.rank-insignia.tier-5 { background: conic-gradient(from 0deg, #ffd700, #ff8c00, #ff0080, #8b00ff, #00cfff, #ffd700); border: 2px solid #ffd700; box-shadow: 0 0 16px rgba(255,215,0,0.5); animation: holo-spin 6s linear infinite; }
@keyframes holo-spin {
	from { filter: hue-rotate(0deg); }
	to   { filter: hue-rotate(360deg); }
}
.rank-insignia-symbol { line-height: 1; z-index: 1; }
.rank-check {
	position: absolute;
	bottom: -3px;
	right: -3px;
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
	background: #22c55e;
	color: #fff;
	font-size: 0.55rem;
	font-weight: 900;
	display: flex;
	align-items: center;
	justify-content: center;
	border: 1.5px solid #fff;
}

/* ─── Rank info ─────────────────────────────────────────────────────── */
.rank-info { flex: 1; min-width: 0; }
.rank-tier-label {
	font-size: 0.85rem;
	font-weight: 800;
	margin-bottom: 0.2rem;
}
.rank-card.owned .rank-tier-label.tier-1 { color: #3b5bb5; }
.rank-card.owned .rank-tier-label.tier-2 { color: #1a6b40; }
.rank-card.owned .rank-tier-label.tier-3 { color: #92400e; }
.rank-card.owned .rank-tier-label.tier-4 { color: #fef3c7; }
.rank-card.owned .rank-tier-label.tier-5 { color: #ffd700; }
.rank-card.locked .rank-tier-label { color: #9ca3af; }

.rank-desc {
	font-size: 0.72rem;
	line-height: 1.4;
}
.rank-card.owned .rank-desc { color: inherit; opacity: 0.85; }
.rank-card.tier-4.owned .rank-desc { color: #d6b870; }
.rank-card.tier-5.owned .rank-desc { color: #c8a44a; }
.rank-card.locked .rank-desc { color: #9ca3af; }

/* ─── Threshold ─────────────────────────────────────────────────────── */
.rank-threshold {
	text-align: right;
	flex-shrink: 0;
}
.rank-threshold-amount {
	display: block;
	font-size: 0.8rem;
	font-weight: 700;
	white-space: nowrap;
}

/* ─── Dark mode tuning (prevents light-card inversion artifacts) ───── */
:global(:root[data-theme='dark']) .rank-status-card {
	background: linear-gradient(135deg, #100c08 0%, #1a1209 100%);
	border-color: #7a5a22;
	color: #f3dfb2;
}

:global(:root[data-theme='dark']) .rank-status-label {
	color: #b99557;
}

:global(:root[data-theme='dark']) .rank-status-title.rank-unranked {
	color: #8f7b57;
}

:global(:root[data-theme='dark']) .progress-track {
	background: #241b12;
}

:global(:root[data-theme='dark']) .rank-connector {
	background: #3a342d;
}

:global(:root[data-theme='dark']) .rank-card.tier-1.owned {
	background: linear-gradient(90deg, #1a2233, #1b2740);
	border-color: #486fb8;
	color: #d6e4ff;
}

:global(:root[data-theme='dark']) .rank-card.tier-2.owned {
	background: linear-gradient(90deg, #13281f, #173326);
	border-color: #3c9b67;
	color: #d8ffe8;
}

:global(:root[data-theme='dark']) .rank-card.tier-3.owned {
	background: linear-gradient(90deg, #2a2115, #362918);
	border-color: #ba8f3d;
	color: #ffeac0;
}

:global(:root[data-theme='dark']) .rank-card.locked {
	background: #151515;
	border-color: #2d2d2d;
}

:global(:root[data-theme='dark']) .rank-card.locked .rank-tier-label,
:global(:root[data-theme='dark']) .rank-card.locked .rank-desc,
:global(:root[data-theme='dark']) .rank-card.locked .rank-threshold-amount,
:global(:root[data-theme='dark']) .rank-card.locked .rank-threshold-label {
	color: #7b7b7b;
}

:global(:root[data-theme='dark']) .rank-card.tier-1.owned .rank-tier-label { color: #9ec3ff; }
:global(:root[data-theme='dark']) .rank-card.tier-2.owned .rank-tier-label { color: #87e0b1; }
:global(:root[data-theme='dark']) .rank-card.tier-3.owned .rank-tier-label { color: #f3cc7e; }
.rank-card.owned .rank-threshold-amount { color: inherit; }
.rank-card.tier-4.owned .rank-threshold-amount { color: #fbbf24; }
.rank-card.tier-5.owned .rank-threshold-amount { color: #ffd700; }
.rank-card.locked .rank-threshold-amount { color: #9ca3af; }
.rank-threshold-label {
	font-size: 0.6rem;
	text-transform: uppercase;
	letter-spacing: 0.1em;
	color: #9ca3af;
}
</style>
