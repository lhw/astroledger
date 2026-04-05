<script lang="ts">
	import { formatBadgeInsurance, getBadgeTierSymbol } from '$lib/badges';
	import type { Badge } from '$lib/types';

	let {
		badges,
		badgesLoading,
		activeBadgeKey,
		badgeSaving,
		onPickBadge
	}: {
		badges: Badge[];
		badgesLoading: boolean;
		activeBadgeKey: string;
		badgeSaving: boolean;
		onPickBadge: (key: string) => Promise<void>;
	} = $props();
</script>

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
					<div class="hangar-icon tier-{badge.tier}">
						{getBadgeTierSymbol(badge.tier)}
					</div>
					<div class="hangar-info">
						<div class="hangar-title">{badge.title}</div>
						<div class="hangar-desc">{badge.description}</div>
						<div class="hangar-date">{new Date(badge.awarded_at).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}</div>
					</div>
					<div class="hangar-actions">
						{#if badge.insurance}
							<span class="ins-pip ins-{badge.insurance}" title="Insurance tier">
								{formatBadgeInsurance(badge.insurance, 'long')}
							</span>
						{:else if badge.purchasable}
							<span class="ins-pip ins-none" title="No insurance selected at purchase">No Ins.</span>
						{/if}
						<button
							class="hangar-btn"
							class:active={isActive}
							disabled={badgeSaving}
							onclick={() => onPickBadge(badge.badge_key)}
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

<style>
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

	:global(:root[data-theme='dark']) .hangar-list { border-color: #2e261c; }
	:global(:root[data-theme='dark']) .hangar-row {
		background: #161210;
		border-bottom-color: #2e261c;
	}
	:global(:root[data-theme='dark']) .hangar-row:hover { background: #201a12; }
	:global(:root[data-theme='dark']) .hangar-row.active {
		background: #1e1800;
		border-left-color: #d4a017;
	}
	:global(:root[data-theme='dark']) .hangar-icon.tier-1 { background: #2a1e10; color: #c8904a; border-color: #5a3e20; }
	:global(:root[data-theme='dark']) .hangar-title { color: #e8d5b8; }
	:global(:root[data-theme='dark']) .hangar-desc { color: #a08870; }
	:global(:root[data-theme='dark']) .hangar-date { color: #6e5e50; }
	:global(:root[data-theme='dark']) .ins-pip.ins-none { background: #1c1c1c; color: #9ca3af; border-color: #374151; }
	:global(:root[data-theme='dark']) .ins-pip.ins-earned { background: #061808; color: #4ade80; border-color: #166534; }
	:global(:root[data-theme='dark']) .ins-pip.ins-6w { background: #1e0a38; color: #c084fc; border-color: #7e22ce; }
	:global(:root[data-theme='dark']) .ins-pip.ins-120w { background: #200c00; color: #fb923c; border-color: #c2410c; }
	:global(:root[data-theme='dark']) .ins-pip.ins-lti { background: #200808; color: #f87171; border-color: #b91c1c; }
	:global(:root[data-theme='dark']) .hangar-btn {
		border-color: #4a3820;
		color: #a08870;
	}
	:global(:root[data-theme='dark']) .hangar-btn:hover:not(:disabled) { background: #2a1e10; border-color: #c8904a; color: #c8904a; }
	:global(:root[data-theme='dark']) .hangar-btn.active { background: #d4a017; border-color: #d4a017; color: #fff; }
</style>