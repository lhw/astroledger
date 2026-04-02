<script lang="ts">
	import { RangeCalendar } from 'bits-ui';
	import { today, getLocalTimeZone, CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';
	import type { AdminBadgeRelease, BadgeCatalogEntry } from '$lib/types';
	import { ApiClientError } from '$lib/api';
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
	let badgePickerOpen = $state(false);

	// Derived selected badge catalog entry for preview
	const selectedBadge = $derived(catalog.find((e) => e.key === cfBadgeKey) ?? null);
	const selectedBadgeLabel = $derived(selectedBadge ? `[T${selectedBadge.tier}] ${selectedBadge.title}` : 'Select badge');
	const sortedCatalog = $derived(
		[...catalog].filter((e) => e.purchasable).sort((a, b) => (a.tier === b.tier ? a.title.localeCompare(b.title) : a.tier - b.tier))
	);

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
			cfBadgeKey = '';
			cfPrice = '';
			cfStock = '';
			cfReleasedAt = '';
			cfExpiresAt = '';
			cfIndefinite = false;
			cfNotes = '';
			cfInsurance = '6w';
			badgePickerOpen = false;
			calValue = { start: undefined, end: undefined };
		} catch (e) {
			cfError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			cfLoading = false;
		}
	}

	function selectBadge(key: string) {
		cfBadgeKey = key;
		badgePickerOpen = false;
	}

	// ── Edit state ────────────────────────────────────────────────────────
	type EditDraft = { price: string; stock: string; expiresAt: string; indefinite: boolean; active: boolean; notes: string; insurance: string };
	let editDrafts = $state<Record<number, EditDraft>>({});
	let editSaving = $state<Record<number, boolean>>({});
	let editError = $state<Record<number, string | null>>({});
	let archiveConfirm = $state<number | null>(null);

	function startEdit(rel: AdminBadgeRelease) {
		editDrafts[rel.id] = {
			price: String(rel.price),
			stock: rel.stock != null ? String(rel.stock) : '',
			expiresAt: rel.expires_at ? rel.expires_at.slice(0, 16) : '',
			indefinite: rel.expires_at == null,
			active: rel.active,
			notes: rel.notes ?? '',
			insurance: rel.insurance
		};
	}

	function cancelEdit(id: number) {
		const { [id]: _, ...rest } = editDrafts;
		editDrafts = rest;
		const { [id]: _e, ...eRest } = editError;
		editError = eRest;
	}

	async function saveEdit(rel: AdminBadgeRelease) {
		const d = editDrafts[rel.id];
		if (!d) return;
		editError[rel.id] = null;
		const price = parseInt(d.price, 10);
		if (!Number.isInteger(price) || price < 0) { editError[rel.id] = 'Price must be a non-negative integer.'; return; }
		const stockRaw = stringValue(d.stock).trim();
		const stock = stockRaw ? parseInt(stockRaw, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { editError[rel.id] = 'Stock must be positive.'; return; }
		editSaving[rel.id] = true;
		try {
			const updated = await onUpdate(rel.id, {
				price,
				stock,
				expires_at: d.indefinite || !d.expiresAt ? null : new Date(d.expiresAt).toISOString(),
				active: d.active,
				notes: d.notes.trim() || null,
				insurance: d.insurance
			});
			releases = releases.map((r) => (r.id === rel.id ? updated : r));
			cancelEdit(rel.id);
		} catch (e) {
			editError[rel.id] = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			editSaving[rel.id] = false;
		}
	}

	async function confirmArchive(id: number) {
		editSaving[id] = true;
		try {
			await onArchive(id);
			releases = releases.map((r) => (r.id === id ? { ...r, active: false } : r));
			cancelEdit(id);
		} catch (e) {
			editError[id] = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			editSaving[id] = false;
			archiveConfirm = null;
		}
	}

	// ── Display helpers ───────────────────────────────────────────────────
	const TIER_LABELS: Record<number, string> = { 1: 'Common', 2: 'Uncommon', 3: 'Rare', 4: 'Epic', 5: 'Legendary' };
	const TIER_COLORS: Record<number, string> = {
		1: 'text-surface-400',
		2: 'text-green-400',
		3: 'text-blue-400',
		4: 'text-purple-400',
		5: 'text-yellow-400'
	};
	const TIER_BG: Record<number, string> = {
		1: 'bg-surface-600',
		2: 'bg-green-900/60',
		3: 'bg-blue-900/60',
		4: 'bg-purple-900/60',
		5: 'bg-yellow-900/60'
	};

	function releaseStatus(rel: AdminBadgeRelease): 'active' | 'scheduled' | 'expired' | 'archived' {
		if (!rel.active) return 'archived';
		const now = new Date();
		if (new Date(rel.released_at) > now) return 'scheduled';
		if (rel.expires_at && new Date(rel.expires_at) < now) return 'expired';
		return 'active';
	}

	const STATUS_CHIP: Record<string, string> = {
		active: 'release-status-chip release-status-chip-active',
		scheduled: 'release-status-chip release-status-chip-scheduled',
		expired: 'release-status-chip release-status-chip-expired',
		archived: 'release-status-chip release-status-chip-archived'
	};

	function fmtDate(iso: string) {
		return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	// Tier color dots for calendar indicators (limited to 3 distinct colors per day)
	const TIER_DOT: Record<number, string> = {
		1: 'bg-surface-400',
		2: 'bg-green-500',
		3: 'bg-blue-500',
		4: 'bg-purple-500',
		5: 'bg-yellow-400'
	};

	// Sorted releases: active/scheduled first, then archived
	const sortedReleases = $derived(
		[...releases].sort((a, b) => {
			const sa = releaseStatus(a);
			const sb = releaseStatus(b);
			const order = { active: 0, scheduled: 1, expired: 2, archived: 3 };
			if (order[sa] !== order[sb]) return order[sa] - order[sb];
			return new Date(b.released_at).getTime() - new Date(a.released_at).getTime();
		})
	);
</script>

<div class="space-y-8">
	<!-- ── Two-column: Calendar + Form ───────────────────────────────────── -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<!-- Left: Create form -->
		<section class="bg-surface-800 border border-surface-700 rounded-xl p-5 space-y-4">
			<h2 class="text-xs font-bold uppercase tracking-widest text-surface-400">New Release</h2>
			<p class="text-[11px] text-surface-500">Select a date range on the calendar →, then fill in the details below.</p>

			<!-- Badge selector -->
			<div>
				<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-badge-trigger">Badge</label>
				<div class="relative">
					<button
						id="brc-badge-trigger"
						type="button"
						onclick={() => (badgePickerOpen = !badgePickerOpen)}
						class="w-full flex items-center justify-between gap-3 bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
						aria-expanded={badgePickerOpen}
						aria-label="Select badge to release"
					>
						{#if selectedBadge}
							<div class="flex items-center gap-2 min-w-0">
								<BadgePill tier={selectedBadge.tier} title={selectedBadge.title} />
								<span class="text-[11px] text-surface-400 truncate">{selectedBadgeLabel}</span>
							</div>
						{:else}
							<span class="text-surface-400">— select a badge —</span>
						{/if}
						<span class="text-surface-400">▾</span>
					</button>

					{#if badgePickerOpen}
						<div class="badge-picker absolute z-20 mt-1 w-full max-h-72 overflow-auto rounded-lg border border-surface-600 shadow-xl">
							{#each sortedCatalog as entry}
								<button
									type="button"
									onclick={() => selectBadge(entry.key)}
									class="badge-option w-full px-3 py-2 text-left transition-colors border-b border-surface-700 last:border-b-0"
								>
									<div class="flex items-center gap-2 mb-1">
										<BadgePill tier={entry.tier} title={entry.title} />
										<span class="text-[10px] text-surface-400">T{entry.tier}</span>
									</div>
									<p class="text-[11px] text-surface-400 leading-relaxed line-clamp-2">{entry.description}</p>
								</button>
							{/each}
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
									{TIER_LABELS[tier] ?? `T${tier}`}
								</span>
							{/each}
						</div>
					{/if}
				{/snippet}
			</RangeCalendar.Root>
		</section>
	</div>

	<!-- ── Releases timeline list ─────────────────────────────────────────── -->
	<section class="bg-surface-800 border border-surface-700 rounded-xl overflow-hidden">
		<div class="px-5 py-4 border-b border-surface-700">
			<h2 class="text-xs font-bold uppercase tracking-widest text-surface-400">All Releases</h2>
		</div>

		{#if sortedReleases.length === 0}
			<p class="text-surface-500 text-sm px-5 py-8 text-center">No releases yet. Create one above.</p>
		{:else}
			<div class="divide-y divide-surface-700/50">
				{#each sortedReleases as rel (rel.id)}
					{@const draft = editDrafts[rel.id]}
					{@const saving = editSaving[rel.id] ?? false}
					{@const status = releaseStatus(rel)}

					<div class="px-5 py-4">
						{#if draft}
							<!-- Edit mode -->
							<div class="space-y-3">
								<div class="flex items-center gap-2 flex-wrap">
									<span class="text-[10px] font-bold uppercase tracking-widest {TIER_COLORS[rel.tier]}">
										{TIER_LABELS[rel.tier] ?? `T${rel.tier}`}
									</span>
									<span class="text-surface-100 text-sm font-semibold">{rel.title}</span>
									<code class="text-[10px] text-surface-500 font-mono">{rel.badge_key}</code>
								</div>
								<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
									<label class="block text-[11px] text-surface-400">Price (bUEC)
										<input type="number" min="0" bind:value={draft.price}
											class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
									</label>
									<label class="block text-[11px] text-surface-400">Stock (blank=∞)
										<input type="number" min="1" bind:value={draft.stock}
											class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
									</label>
									<label class="block text-[11px] text-surface-400 sm:col-span-2">Expires (blank=never)
										<input type="datetime-local" bind:value={draft.expiresAt}
											disabled={draft.indefinite}
											class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-xs text-surface-100 outline-none focus:border-primary-500" />
										<label class="mt-2 inline-flex items-center gap-2 text-[11px] text-surface-400">
											<input
												type="checkbox"
												bind:checked={draft.indefinite}
												onchange={() => {
													if (draft.indefinite) draft.expiresAt = '';
												}}
											/>
											Available indefinitely
										</label>
									</label>
									<label class="block text-[11px] text-surface-400 sm:col-span-3">Notes
										<input type="text" bind:value={draft.notes}
											class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-sm text-surface-100 outline-none focus:border-primary-500" />
									</label>
									<label class="flex items-end gap-2 pb-1 text-sm text-surface-300 cursor-pointer">
										<input type="checkbox" bind:checked={draft.active} class="w-4 h-4 accent-primary-400" />
										Active
									</label>
								</div>
								{#if editError[rel.id]}
									<p class="text-red-400 text-xs">{editError[rel.id]}</p>
								{/if}
								<div class="flex flex-wrap gap-2">
									<button onclick={() => saveEdit(rel)} disabled={saving}
										class="btn preset-filled-primary-500 text-xs uppercase tracking-wider disabled:opacity-50 px-3 py-1">
										{saving ? 'Saving…' : 'Save'}
									</button>
									<button onclick={() => cancelEdit(rel.id)} disabled={saving}
										class="btn bg-surface-700 text-surface-300 hover:bg-surface-600 text-xs uppercase tracking-wider px-3 py-1">
										Cancel
									</button>
									{#if status !== 'archived'}
										{#if archiveConfirm === rel.id}
											<button onclick={() => confirmArchive(rel.id)} disabled={saving}
												class="btn bg-red-700 text-red-100 text-xs uppercase tracking-wider px-3 py-1 ml-auto">
												Confirm archive
											</button>
											<button onclick={() => (archiveConfirm = null)}
												class="btn bg-surface-700 text-surface-400 text-xs px-3 py-1">Cancel</button>
										{:else}
											<button onclick={() => (archiveConfirm = rel.id)} disabled={saving}
												class="btn bg-surface-700 text-red-400 border border-red-900 hover:bg-red-900/30 text-xs uppercase tracking-wider px-3 py-1 ml-auto">
												Archive
											</button>
										{/if}
									{/if}
								</div>
							</div>
						{:else}
							<!-- View mode -->
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1 min-w-0 space-y-2">
									<!-- Badge pill + status -->
									<div class="flex items-center gap-2 flex-wrap">
										<BadgePill tier={rel.tier} title={rel.title} />
										<code class="text-[10px] text-surface-500 font-mono">{rel.badge_key}</code>
										<span class="inline-block px-1.5 py-0.5 rounded text-[10px] font-bold uppercase border {STATUS_CHIP[status]}">
											{status}
										</span>
									</div>

									<!-- Timeline bar -->
									<div class="flex items-center gap-2 text-[11px] text-surface-500 flex-wrap">
										<span class="text-primary-400 font-semibold">{rel.price.toLocaleString()} bUEC</span>
										<span class="text-surface-600">·</span>
										{#if rel.stock != null}
											<span>{rel.sold}/{rel.stock} sold</span>
										{:else}
											<span>{rel.sold} sold · ∞ stock</span>
										{/if}
										<span class="text-surface-600">·</span>
										<span>From {fmtDate(rel.released_at)}</span>
										{#if rel.expires_at}
											<span>→ {fmtDate(rel.expires_at)}</span>
										{:else}
											<span>→ no expiry</span>
										{/if}
										{#if rel.notes}
											<span class="text-surface-600">·</span>
											<span class="italic text-surface-500">{rel.notes}</span>
										{/if}
										{#if rel.insurance}
											<span class="text-surface-600">·</span>
											<span class="release-insurance-chip inline-flex items-center px-1 py-0.5 rounded text-[10px] font-semibold border">
												{rel.insurance === '6w' ? '6W Ins.' : rel.insurance === '120w' ? '120W Ins.' : 'LTI'}
											</span>
										{/if}
									</div>
								</div>

								{#if status !== 'archived'}
									<button
										onclick={() => startEdit(rel)}
										class="shrink-0 text-xs text-surface-400 hover:text-surface-200 uppercase tracking-wider border border-surface-600 hover:border-surface-400 rounded px-2 py-1 transition-colors"
									>
										Edit
									</button>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>

<style>
	.badge-picker {
		background: var(--card-bg);
		border-color: var(--color-surface-300);
	}

	.badge-option {
		color: var(--app-text);
	}

	.badge-option:hover {
		background: color-mix(in oklch, var(--card-bg) 82%, var(--color-primary-300) 18%);
	}

	.release-status-chip {
		display: inline-block;
		border-width: 1px;
	}

	.release-status-chip-active {
		background: #dcfce7;
		color: #166534;
		border-color: #86efac;
	}

	.release-status-chip-scheduled {
		background: #dbeafe;
		color: #1d4ed8;
		border-color: #93c5fd;
	}

	.release-status-chip-expired {
		background: #fee2e2;
		color: #b91c1c;
		border-color: #fca5a5;
	}

	.release-status-chip-archived {
		background: color-mix(in oklch, var(--color-surface-100) 86%, white 14%);
		color: var(--color-surface-700);
		border-color: var(--color-surface-300);
	}

	.release-insurance-chip {
		background: #fef3c7;
		color: #92400e;
		border-color: #f59e0b;
	}

	:global(:root[data-theme='dark']) .badge-picker {
		background: color-mix(in oklch, var(--card-bg) 92%, black 8%);
		box-shadow: 0 8px 28px rgba(0, 0, 0, 0.55);
	}

	:global(:root[data-theme='dark']) .badge-option {
		border-color: var(--color-surface-300) !important;
	}

	:global(:root[data-theme='dark']) .release-status-chip-active {
		background: rgba(20, 83, 45, 0.7);
		color: #86efac;
		border-color: rgba(22, 101, 52, 0.8);
	}

	:global(:root[data-theme='dark']) .release-status-chip-scheduled {
		background: rgba(30, 64, 175, 0.5);
		color: #93c5fd;
		border-color: rgba(30, 64, 175, 0.8);
	}

	:global(:root[data-theme='dark']) .release-status-chip-expired {
		background: rgba(127, 29, 29, 0.55);
		color: #fca5a5;
		border-color: rgba(153, 27, 27, 0.85);
	}

	:global(:root[data-theme='dark']) .release-status-chip-archived {
		background: color-mix(in oklch, var(--card-bg) 84%, var(--color-surface-300) 16%);
		color: var(--color-surface-500);
		border-color: var(--color-surface-300);
	}

	:global(:root[data-theme='dark']) .release-insurance-chip {
		background: rgba(120, 53, 15, 0.45);
		color: #fcd34d;
		border-color: rgba(180, 83, 9, 0.6);
	}
</style>
