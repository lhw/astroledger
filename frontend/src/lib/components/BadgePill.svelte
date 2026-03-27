<!--
  Reusable badge pill visual component.
  Renders the tier-styled badge visual (no interaction — wrap in <button> if needed).

  Usage:
    <BadgePill tier={3} title="Crash Lord" />
    <BadgePill tier={5} title="Legend" showCheck />
-->
<script lang="ts">
	let {
		tier = 1,
		title,
		showPip = true,
		showCheck = false,
		active = false
	}: {
		tier: number;
		title: string;
		showPip?: boolean;
		showCheck?: boolean;
		active?: boolean;
	} = $props();

	const pips: Record<number, string> = { 1: '▲', 2: '●', 3: '◆', 4: '◈', 5: '★' };
</script>

<span class="badge-pill tier-{tier}" class:badge-pill-active={active}>
	{#if showPip}
		<span class="badge-pip">{pips[tier] ?? '●'}</span>
	{/if}
	<span class="badge-pill-title">{title}</span>
	{#if showCheck}
		<span class="badge-check">✓</span>
	{/if}
</span>

<style>
.badge-pill {
	display: inline-flex;
	align-items: center;
	gap: 0.4rem;
	padding: 0.35rem 0.85rem 0.35rem 0.65rem;
	border-radius: 9999px;
	font-size: 0.75rem;
	font-weight: 700;
	pointer-events: none;
	white-space: nowrap;
	transition: box-shadow 0.15s ease;
}
.badge-pill-active {
	outline: 2px solid currentColor;
	outline-offset: 3px;
}
.badge-pip {
	font-size: 0.75rem;
	line-height: 1;
}
.badge-pill-title {
	letter-spacing: 0.02em;
}
.badge-check {
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
	box-shadow: 0 0 6px rgba(240, 192, 64, 0.3);
}
/* T4 Epic */
.badge-pill.tier-4 {
	background: linear-gradient(90deg, #1c1008, #2d1c00);
	border: 1.5px solid #fbbf24;
	color: #fef3c7;
	box-shadow: 0 0 8px rgba(251, 191, 36, 0.4);
	animation: pill-glow 3s ease-in-out infinite alternate;
}
@keyframes pill-glow {
	from { box-shadow: 0 0 6px rgba(251, 191, 36, 0.35); }
	to   { box-shadow: 0 0 14px rgba(251, 191, 36, 0.65); }
}
/* T5 Legendary */
.badge-pill.tier-5 {
	background: linear-gradient(90deg, #0d0d0d, #1a1200);
	border: 1.5px solid #ffd700;
	color: #ffd700;
	text-shadow: 0 0 6px rgba(255, 215, 0, 0.5);
	box-shadow: 0 0 12px rgba(255, 215, 0, 0.35);
	animation: pill-legendary 4s linear infinite;
}
@keyframes pill-legendary {
	from { box-shadow: 0 0 10px rgba(255, 215, 0, 0.3), 0 0 0 1.5px #ffd700; }
	50%  { box-shadow: 0 0 18px rgba(200, 100, 255, 0.4), 0 0 0 1.5px #c880ff; }
	to   { box-shadow: 0 0 10px rgba(255, 215, 0, 0.3), 0 0 0 1.5px #ffd700; }
}
</style>
