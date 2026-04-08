<script lang="ts">
	import { RangeCalendar } from 'bits-ui';
	import { today, getLocalTimeZone, CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';
	import { getBadgeTierLabel, getBadgeTierSymbol } from '$lib/badges';
	import type { AdminBadgeRelease, BadgeCatalogEntry } from '$lib/types';
	import { ApiClientError } from '$lib/api';
	import AdminBadgeReleaseList from '$lib/components/admin/AdminBadgeReleaseList.svelte';
	import BadgePill from '$lib/components/BadgePill.svelte';
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';

	let {
		catalog,
		releases = $bindable(),
		onCreate,
		onUpdate,
		onArchive
	}: {
		catalog: BadgeCatalogEntry[];
		releases: AdminBadgeRelease[];
		onCreate: (body: {
			badge_key: string;
			price: number;
			stock: number | null;
			released_at: string;
			expires_at: string | null;
			notes: string | null;
			insurance: string;
		}) => Promise<AdminBadgeRelease>;
		onUpdate: (
			id: number,
			body: {
				price: number;
				stock: number | null;
				expires_at: string | null;
				active: boolean;
				notes: string | null;
				insurance: string;
			}
		) => Promise<AdminBadgeRelease>;
		onArchive: (id: number) => Promise<void>;
	} = $props();

	// ── Calendar selection ────────────────────────────────────────────────
	const tz = getLocalTimeZone();
	let calValue = $state<DateRange>({ start: undefined, end: undefined });

	// When range selected, fill form dates
	$effect(() => {
		if (calValue?.start) {
			const s = calValue.start;
			cfReleasedAt = `${s.year}-${String(s.month).padStart(2, '0')}-${String(s.day).padStart(2, '0')}T00:00`;
		}
		if (calValue?.end) {
			const e = calValue.end;
			cfExpiresAt = `${e.year}-${String(e.month).padStart(2, '0')}-${String(e.day).padStart(2, '0')}T23:59`;
			cfIndefinite = false;
		}
	});

	// Returns releases active on a given calendar day (for dot indicators)
	function releasesOnDay(date: DateValue): AdminBadgeRelease[] {
		const d = new Date(date.year, date.month - 1, date.day);
		return releases.filter((rel) => {
			if (!rel.active) return false;
			const start = new Date(rel.released_at.slice(0, 10));
			const end = rel.expires_at ? new Date(rel.expires_at.slice(0, 10)) : null;
			return d >= start && (!end || d <= end);
		});
	}

	// ── Create form ───────────────────────────────────────────────────────
	let cfBadgeKey = $state('');
	let cfPrice = $state('');
	let cfStock = $state('');
	let cfReleasedAt = $state('');
	let cfExpiresAt = $state('');
	let cfIndefinite = $state(false);
	let cfNotes = $state('');
	let cfInsurance = $state('6w');
	let cfLoading = $state(false);
	let cfError = $state<string | null>(null);

	// Derived selected badge catalog entry for preview
	const selectedBadge = $derived(catalog.find((e) => e.key === cfBadgeKey) ?? null);
	const sortedCatalog = $derived(
		[...catalog].filter((e) => e.purchasable).sort((a, b) => (a.tier === b.tier ? a.title.localeCompare(b.title) : a.tier - b.tier))
	);

	// Combobox state for badge search
	let cfBadgeSearch = $state('');
	let cfBadgeOpen = $state(false);
	const cfFilteredCatalog = $derived(
		cfBadgeSearch.trim()
			? sortedCatalog.filter(
					(e) =>
						e.title.toLowerCase().includes(cfBadgeSearch.toLowerCase()) ||
						e.key.toLowerCase().includes(cfBadgeSearch.toLowerCase())
			  )
			: sortedCatalog
	);

	function selectBadge(entry: BadgeCatalogEntry) {
		cfBadgeKey = entry.key;
		cfBadgeSearch = `T${entry.tier} · ${entry.title}`;
		cfBadgeOpen = false;
	}

	function onBadgeSearchInput() {
		const currentLabel = selectedBadge ? `T${selectedBadge.tier} · ${selectedBadge.title}` : '';
		if (cfBadgeKey && cfBadgeSearch !== currentLabel) {
			cfBadgeKey = '';
		}
		cfBadgeOpen = true;
	}

	function stringValue(value: string | number | null | undefined): string {
		if (value == null) return '';
		return String(value);
	}

	async function createRelease() {
		cfError = null;
		if (!cfBadgeKey) { cfError = 'Select a badge.'; return; }
		const price = parseInt(cfPrice, 10);
		if (!Number.isInteger(price) || price < 0) { cfError = 'Price must be a non-negative integer.'; return; }
		const stockRaw = stringValue(cfStock).trim();
		const stock = stockRaw ? parseInt(stockRaw, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { cfError = 'Stock must be positive.'; return; }
		const releasedAt = cfReleasedAt ? new Date(cfReleasedAt).toISOString() : new Date().toISOString();
		const expiresAt = cfIndefinite || !cfExpiresAt ? null : new Date(cfExpiresAt).toISOString();
		cfLoading = true;
		try {
			const created = await onCreate({
				badge_key: cfBadgeKey,
				price,
				stock,
				released_at: releasedAt,
				expires_at: expiresAt,
				notes: cfNotes.trim() || null,
				insurance: cfInsurance
			});
			releases = [created, ...releases];
			cfBadgeKey = '';		cfBadgeSearch = '';
		cfBadgeOpen = false;			cfPrice = '';
			cfStock = '';
			cfReleasedAt = '';
			cfExpiresAt = '';
			cfIndefinite = false;
			cfNotes = '';
			cfInsurance = '6w';
			calValue = { start: undefined, end: undefined };
		} catch (e) {
			cfError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			cfLoading = false;
		}
	}

	// ── Display helpers ───────────────────────────────────────────────────
	// Tier color dots for calendar indicators (limited to 3 distinct colors per day)
	const TIER_DOT: Record<number, string> = {
		1: 'bg-surface-400',
		2: 'bg-green-500',
		3: 'bg-blue-500',
		4: 'bg-purple-500',
		5: 'bg-yellow-400'
	};

</script>

<div class="space-y-8">
	<!-- ── Two-column: Calendar + Form ───────────────────────────────────── -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<!-- Left: Create form -->
		<section class="bg-surface-800 border border-surface-700 rounded-xl p-5 space-y-4">
			<h2 class="text-xs font-bold uppercase tracking-widest text-surface-400">New Release</h2>
			<p class="text-[11px] text-surface-500">Select a date range on the calendar →, then fill in the details below.</p>

			<!-- Badge selector (searchable combobox) -->
			<div>
				<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-badge-search">Badge</label>
				<div class="relative">
					<input
						id="brc-badge-search"
						type="text"
						bind:value={cfBadgeSearch}
						onfocus={() => (cfBadgeOpen = true)}
						onblur={() => setTimeout(() => { cfBadgeOpen = false; }, 160)}
						oninput={onBadgeSearchInput}
						placeholder={selectedBadge ? `T${selectedBadge.tier} · ${selectedBadge.title}` : 'Search by name or key…'}
						autocomplete="off"
						class="w-full bg-surface-700 border {cfBadgeKey ? 'border-primary-500' : 'border-surface-600'} rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
					/>
					{#if cfBadgeKey && !cfBadgeOpen}
						<button
							type="button"
							class="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-100 text-xs px-1"
							onclick={() => { cfBadgeKey = ''; cfBadgeSearch = ''; }}
							aria-label="Clear selection"
						>✕</button>
					{/if}
					{#if cfBadgeOpen}
						<div class="absolute z-30 left-0 right-0 top-full mt-1 bg-surface-800 border border-surface-600 rounded-lg shadow-2xl max-h-56 overflow-y-auto">
							{#each cfFilteredCatalog as entry (entry.key)}
								<button
									type="button"
									class="w-full text-left px-3 py-2.5 flex items-center gap-2.5 hover:bg-surface-700 transition-colors border-b border-surface-700/50 last:border-0 {cfBadgeKey === entry.key ? 'bg-primary-900/30' : ''}"
									onmousedown={() => selectBadge(entry)}
								>
									<span class="text-base shrink-0 w-6 text-center opacity-75">{getBadgeTierSymbol(entry.tier)}</span>
									<div class="min-w-0 flex-1">
										<div class="text-xs font-semibold text-surface-100 leading-tight">{entry.title}</div>
										{#if entry.description}
											<div class="text-[10px] text-surface-400 truncate leading-tight mt-0.5">{entry.description}</div>
										{/if}
									</div>
									<span class="ml-auto shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded bg-surface-600 text-surface-300">T{entry.tier}</span>
								</button>
							{/each}
							{#if cfFilteredCatalog.length === 0}
								<p class="px-3 py-3 text-xs text-surface-500 text-center">No badges match "{cfBadgeSearch}"</p>
							{/if}
						</div>
					{/if}
				</div>
				{#if selectedBadge}
					<div class="mt-2 space-y-1">
						<BadgePill tier={selectedBadge.tier} title={selectedBadge.title} />
						<p class="text-[11px] text-surface-400 leading-relaxed">{selectedBadge.description}</p>
					</div>
				{/if}
			</div>

			<!-- Price + Stock row -->
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-price">Price (bUEC)</label>
					<input
						id="brc-price"
						type="number"
						min="0"
						bind:value={cfPrice}
						placeholder="e.g. 500"
						class="w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
					/>
				</div>
				<div>
					<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-stock">Stock (blank=∞)</label>
					<input
						id="brc-stock"
						type="number"
						min="1"
						bind:value={cfStock}
						placeholder="e.g. 25"
						class="w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
					/>
				</div>
			</div>

			<!-- Date fields (pre-filled from calendar selection) -->
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-start">
						Release Date
					</label>
					<input
						id="brc-start"
						type="datetime-local"
						bind:value={cfReleasedAt}
						class="w-full bg-surface-700 border border-surface-600 rounded-lg px-2 py-2 text-xs text-surface-100 focus:border-primary-500 outline-none"
					/>
				</div>
				<div>
					<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-end">
						Expires (optional)
					</label>
					<input
						id="brc-end"
						type="datetime-local"
						bind:value={cfExpiresAt}
						disabled={cfIndefinite}
						class="w-full bg-surface-700 border border-surface-600 rounded-lg px-2 py-2 text-xs text-surface-100 focus:border-primary-500 outline-none"
					/>
					<label class="mt-2 inline-flex items-center gap-2 text-[11px] text-surface-400">
						<input
							type="checkbox"
							bind:checked={cfIndefinite}
							onchange={() => {
								if (cfIndefinite) cfExpiresAt = '';
							}}
						/>
						Available indefinitely
					</label>
				</div>
			</div>

			<!-- Notes -->
			<div>
				<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-notes">Notes (optional)</label>
				<input
					id="brc-notes"
					type="text"
					bind:value={cfNotes}
					placeholder="e.g. CitizenCon 2026 drop"
					class="w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
				/>
			</div>

			<!-- Insurance tier -->
			<div>
				<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-insurance">Insurance Tier</label>
				<select
					id="brc-insurance"
					bind:value={cfInsurance}
					class="w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
				>
					<option value="6w">6 Weeks</option>
					<option value="120w">120 Weeks</option>
					<option value="lti">LTI</option>
				</select>
			</div>

			{#if cfError}
				<p class="text-red-400 text-xs">{cfError}</p>
			{/if}

			<button
				onclick={createRelease}
				disabled={cfLoading}
				class="w-full btn preset-filled-primary-500 text-xs uppercase tracking-widest disabled:opacity-50"
			>
				{cfLoading ? 'Creating…' : 'Create Release'}
			</button>
		</section>

		<!-- Right: Range Calendar -->
		<section class="bg-surface-800 border border-surface-700 rounded-xl p-5">
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-xs font-bold uppercase tracking-widest text-surface-400">Release Window</h2>
				<span class="text-[11px] text-surface-500">Drag to select start → end</span>
			</div>

			<RangeCalendar.Root
				bind:value={calValue}
				weekdayFormat="short"
				fixedWeeks={true}
				class="w-full select-none"
			>
				{#snippet children({ months, weekdays })}
					<RangeCalendar.Header class="flex items-center justify-between mb-3">
						<RangeCalendar.PrevButton
							class="inline-flex size-8 items-center justify-center rounded-lg
							       text-surface-400 hover:text-surface-100 hover:bg-surface-700
							       transition-colors"
						>
							<ChevronLeft size={16} />
						</RangeCalendar.PrevButton>
						<RangeCalendar.Heading class="text-sm font-semibold text-surface-200" />
						<RangeCalendar.NextButton
							class="inline-flex size-8 items-center justify-center rounded-lg
							       text-surface-400 hover:text-surface-100 hover:bg-surface-700
							       transition-colors"
						>
							<ChevronRight size={16} />
						</RangeCalendar.NextButton>
					</RangeCalendar.Header>

					{#each months as month (month.value)}
						<RangeCalendar.Grid class="w-full border-collapse">
							<RangeCalendar.GridHead>
								<RangeCalendar.GridRow class="flex justify-between mb-1">
									{#each weekdays as day (day)}
										<RangeCalendar.HeadCell class="w-9 text-center text-[10px] font-medium text-surface-500 uppercase">
											{day.slice(0, 2)}
										</RangeCalendar.HeadCell>
									{/each}
								</RangeCalendar.GridRow>
							</RangeCalendar.GridHead>

							<RangeCalendar.GridBody class="space-y-0.5">
								{#each month.weeks as weekDates (weekDates)}
									<RangeCalendar.GridRow class="flex justify-between">
										{#each weekDates as date (date)}
											{@const dots = releasesOnDay(date)}
											<RangeCalendar.Cell
												{date}
												month={month.value}
												class="relative p-0 m-0 size-9 text-center text-sm focus-within:z-10"
											>
												<RangeCalendar.Day
													class="relative inline-flex size-9 items-center justify-center rounded-md
													       text-surface-300 text-xs
													       hover:bg-surface-700
													       data-outside-month:opacity-30 data-outside-month:pointer-events-none
													       data-disabled:opacity-20 data-disabled:pointer-events-none
													       data-today:font-bold data-today:text-primary-400
													       data-selection-start:bg-primary-500 data-selection-start:text-surface-900 data-selection-start:rounded-md
													       data-selection-end:bg-primary-500 data-selection-end:text-surface-900 data-selection-end:rounded-md
													       data-highlighted:bg-primary-500/15 data-highlighted:text-surface-100 data-highlighted:rounded-none
													       data-range-middle:rounded-none
													       transition-colors"
												>
													{date.day}
													<!-- Release dot indicators -->
													{#if dots.length > 0}
														<span class="absolute bottom-0.5 left-0 right-0 flex justify-center gap-px pointer-events-none">
															{#each dots.slice(0, 3) as rel (rel.id)}
																<span class="size-1 rounded-full {TIER_DOT[rel.tier] ?? 'bg-surface-400'}"></span>
															{/each}
														</span>
													{/if}
												</RangeCalendar.Day>
											</RangeCalendar.Cell>
										{/each}
									</RangeCalendar.GridRow>
								{/each}
							</RangeCalendar.GridBody>
						</RangeCalendar.Grid>
					{/each}

					<!-- Legend -->
					{#if releases.some((r) => r.active)}
						<div class="mt-3 pt-3 border-t border-surface-700 flex flex-wrap gap-3">
							{#each [...new Set(releases.filter((r) => r.active).map((r) => r.tier))] as tier}
								<span class="flex items-center gap-1 text-[10px] text-surface-400">
									<span class="size-2 rounded-full {TIER_DOT[tier] ?? 'bg-surface-400'}"></span>
									{getBadgeTierLabel(tier)}
								</span>
							{/each}
						</div>
					{/if}
				{/snippet}
			</RangeCalendar.Root>
		</section>
	</div>

	<AdminBadgeReleaseList bind:releases {onUpdate} {onArchive} />
</div>
