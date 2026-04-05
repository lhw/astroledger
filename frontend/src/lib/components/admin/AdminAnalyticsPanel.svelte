<script lang="ts">
	import { adminGetAnalytics, ApiClientError } from '$lib/api';
	import type { AnalyticsStats, AnalyticsStat } from '$lib/types';

	let analyticsPeriod = $state<'7d' | '30d'>('7d');
	let analyticsLoading = $state(false);
	let analyticsData = $state<AnalyticsStats | null>(null);
	let analyticsError = $state<string | null>(null);

	const CHART_W = 840;
	const CHART_H = 96;

	$effect(() => {
		if (analyticsData || analyticsLoading) return;
		void loadAnalytics();
	});

	async function loadAnalytics() {
		analyticsLoading = true;
		analyticsError = null;
		try {
			analyticsData = await adminGetAnalytics(analyticsPeriod);
		} catch (err) {
			analyticsError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			analyticsLoading = false;
		}
	}

	function chartBars(daily: AnalyticsStats['daily']) {
		if (!daily?.length) return [];
		const maxViews = Math.max(...daily.map((day) => day.views), 1);
		const slotWidth = CHART_W / daily.length;
		const gap = Math.max(1, slotWidth * 0.15);
		return daily.map((day, index) => ({
			x: index * slotWidth + gap / 2,
			y: CHART_H - (day.views / maxViews) * CHART_H,
			w: slotWidth - gap,
			h: (day.views / maxViews) * CHART_H,
			views: day.views,
			date: day.date
		}));
	}

	function xLabels(daily: AnalyticsStats['daily']) {
		if (!daily?.length) return [];
		const slotWidth = CHART_W / daily.length;
		const step = daily.length <= 7 ? 1 : 5;
		return daily
			.map((day, index) => ({ date: day.date, x: index * slotWidth + slotWidth / 2 }))
			.filter((_, index) => index % step === 0 || index === daily.length - 1);
	}

	function fmtDate(iso: string) {
		const [, month, day] = iso.split('-');
		const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
		return `${months[parseInt(month) - 1]} ${parseInt(day)}`;
	}

	function avgDailyViews(data: AnalyticsStats) {
		const days = data.daily?.length || 1;
		return Math.round(data.total_views / days);
	}

	function statBarWidth(item: AnalyticsStat, list: AnalyticsStat[]) {
		const max = list[0]?.count || 1;
		return Math.round((item.count / max) * 100);
	}

	function statPanels(data: AnalyticsStats): { label: string; items: AnalyticsStat[] }[] {
		return [
			{ label: 'Browsers', items: data.browsers ?? [] },
			{ label: 'Operating Systems', items: data.systems ?? [] },
			{ label: 'Locations', items: data.locations ?? [] },
			{ label: 'Languages', items: data.languages ?? [] }
		];
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<p class="text-surface-400 text-xs uppercase tracking-widest">Traffic overview</p>
		<div class="flex gap-1 bg-surface-800 border border-surface-700 rounded-lg p-1">
			{#each (['7d', '30d'] as const) as period}
				<button
					onclick={async () => {
						analyticsPeriod = period;
						await loadAnalytics();
					}}
					disabled={analyticsLoading}
					class="px-3 py-1 text-xs font-semibold uppercase tracking-wider rounded-md transition-colors disabled:opacity-40 {analyticsPeriod === period ? 'bg-primary-500 text-white' : 'text-surface-400 hover:text-surface-200'}"
				>{period === '7d' ? '7 Days' : '30 Days'}</button>
			{/each}
		</div>
	</div>

	{#if analyticsLoading}
		<div class="text-surface-400 text-sm py-12 text-center tracking-wide">Loading analytics…</div>
	{:else if analyticsError}
		<div class="bg-red-950/40 border border-red-800 rounded-xl p-5 text-red-400 text-sm">
			Failed to load analytics: {analyticsError}
		</div>
	{:else if analyticsData && !analyticsData.configured}
		<div class="bg-surface-800 border border-surface-700 rounded-xl p-8 text-center space-y-3">
			<p class="text-surface-200 text-sm font-semibold">GoatCounter not configured</p>
			<p class="text-surface-400 text-xs leading-relaxed max-w-md mx-auto">
				Set <code class="text-primary-400 bg-surface-700 px-1 rounded">GOATCOUNTER_API_KEY</code>
				in your environment. Create an API token in the GoatCounter settings UI at your stats
				subdomain, then redeploy.
			</p>
		</div>
	{:else if analyticsData}
		<div class="grid grid-cols-3 gap-4">
			{#each [
				{ label: 'Total Views', value: analyticsData.total_views.toLocaleString(), sub: analyticsPeriod === '7d' ? 'last 7 days' : 'last 30 days' },
				{ label: 'Unique Visitors', value: analyticsData.total_unique.toLocaleString(), sub: 'estimated' },
				{ label: 'Avg Daily Views', value: avgDailyViews(analyticsData).toLocaleString(), sub: 'per day' }
			] as card}
				<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
					<p class="text-surface-400 text-xs uppercase tracking-widest mb-2">{card.label}</p>
					<p class="text-2xl font-bold text-primary-300">{card.value}</p>
					<p class="text-surface-500 text-xs mt-1">{card.sub}</p>
				</div>
			{/each}
		</div>

		{#if analyticsData.daily?.length}
			<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
				<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Daily Traffic</p>
				<div class="text-primary-400 w-full overflow-hidden">
					<svg
						viewBox="0 0 {CHART_W} {CHART_H + 28}"
						class="w-full"
						aria-label="Daily traffic bar chart"
					>
						<line x1="0" y1={CHART_H} x2={CHART_W} y2={CHART_H} stroke="currentColor" stroke-opacity="0.15" stroke-width="1" />

						{#each chartBars(analyticsData.daily) as bar}
							<g>
								<rect
									x={bar.x}
									y={bar.y}
									width={bar.w}
									height={bar.h}
									fill="currentColor"
									opacity="0.85"
									rx="2"
								/>
								<title>{fmtDate(bar.date)}: {bar.views.toLocaleString()} views</title>
							</g>
						{/each}

						{#each xLabels(analyticsData.daily) as label}
							<text
								x={label.x}
								y={CHART_H + 18}
								text-anchor="middle"
								font-size="11"
								fill="currentColor"
								opacity="0.75"
							>{fmtDate(label.date)}</text>
						{/each}
					</svg>
				</div>
			</div>
		{/if}

		<div class="grid grid-cols-2 gap-4">
			<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
				<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Top Pages</p>
				{#if analyticsData.top_pages?.length}
					<div class="space-y-2">
						{#each analyticsData.top_pages as page}
							<div class="flex items-center justify-between gap-3 group">
								<span class="text-surface-300 text-xs truncate font-mono flex-1" title={page.path}>{page.path || '/'}</span>
								<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{page.views.toLocaleString()}</span>
							</div>
							<div class="w-full bg-surface-700 rounded-full h-0.5">
								<div
									class="bg-primary-500 h-0.5 rounded-full transition-all"
									style="width: {Math.round((page.views / (analyticsData.top_pages[0]?.views || 1)) * 100)}%"
								></div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-surface-500 text-xs">No page data yet.</p>
				{/if}
			</div>

			<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
				<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">Top Referrers</p>
				{#if analyticsData.top_refs?.length}
					<div class="space-y-2">
						{#each analyticsData.top_refs as ref}
							<div class="flex items-center justify-between gap-3">
								<span class="text-surface-300 text-xs truncate flex-1" title={ref.name}>{ref.name || '(direct)'}</span>
								<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{ref.views.toLocaleString()}</span>
							</div>
							<div class="w-full bg-surface-700 rounded-full h-0.5">
								<div
									class="bg-primary-500 h-0.5 rounded-full"
									style="width: {Math.round((ref.views / (analyticsData.top_refs[0]?.views || 1)) * 100)}%"
								></div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-surface-500 text-xs">No referrer data yet.</p>
				{/if}
			</div>
		</div>

		<div class="grid grid-cols-2 gap-4">
			{#each statPanels(analyticsData) as panel}
				<div class="bg-surface-800 border border-surface-700 rounded-xl p-5">
					<p class="text-surface-400 text-xs uppercase tracking-widest mb-4">{panel.label}</p>
					{#if panel.items?.length}
						<div class="space-y-2">
							{#each panel.items as item}
								<div class="flex items-center justify-between gap-3">
									<span class="text-surface-300 text-xs truncate flex-1" title={item.name}>{item.name}</span>
									<span class="text-primary-300 text-xs font-semibold tabular-nums shrink-0">{item.count.toLocaleString()}</span>
								</div>
								<div class="w-full bg-surface-700 rounded-full h-0.5">
									<div class="bg-primary-500 h-0.5 rounded-full" style="width: {statBarWidth(item, panel.items)}%"></div>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-surface-500 text-xs">No data yet.</p>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>