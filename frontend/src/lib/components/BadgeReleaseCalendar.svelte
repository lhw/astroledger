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
		}) => Promise<AdminBadgeRelease>;
		onUpdate: (
			id: number,
			body: {
				price: number;
				stock: number | null;
				expires_at: string | null;
				active: boolean;
				notes: string | null;
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
	let cfNotes = $state('');
	let cfLoading = $state(false);
	let cfError = $state<string | null>(null);

	// Derived selected badge catalog entry for preview
	const selectedBadge = $derived(catalog.find((e) => e.key === cfBadgeKey) ?? null);

	async function createRelease() {
		cfError = null;
		if (!cfBadgeKey) { cfError = 'Select a badge.'; return; }
		const price = parseInt(cfPrice, 10);
		if (!Number.isInteger(price) || price < 0) { cfError = 'Price must be a non-negative integer.'; return; }
		const stock = cfStock.trim() ? parseInt(cfStock, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { cfError = 'Stock must be positive.'; return; }
		const releasedAt = cfReleasedAt ? new Date(cfReleasedAt).toISOString() : new Date().toISOString();
		const expiresAt = cfExpiresAt ? new Date(cfExpiresAt).toISOString() : null;
		cfLoading = true;
		try {
			const created = await onCreate({
				badge_key: cfBadgeKey,
				price,
				stock,
				released_at: releasedAt,
				expires_at: expiresAt,
				notes: cfNotes.trim() || null
			});
			releases = [created, ...releases];
			cfBadgeKey = '';
			cfPrice = '';
			cfStock = '';
			cfReleasedAt = '';
			cfExpiresAt = '';
			cfNotes = '';
			calValue = { start: undefined, end: undefined };
		} catch (e) {
			cfError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			cfLoading = false;
		}
	}

	// ── Edit state ────────────────────────────────────────────────────────
	type EditDraft = { price: string; stock: string; expiresAt: string; active: boolean; notes: string };
	let editDrafts = $state<Record<number, EditDraft>>({});
	let editSaving = $state<Record<number, boolean>>({});
	let editError = $state<Record<number, string | null>>({});
	let archiveConfirm = $state<number | null>(null);

	function startEdit(rel: AdminBadgeRelease) {
		editDrafts[rel.id] = {
			price: String(rel.price),
			stock: rel.stock != null ? String(rel.stock) : '',
			expiresAt: rel.expires_at ? rel.expires_at.slice(0, 16) : '',
			active: rel.active,
			notes: rel.notes ?? ''
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
		const stock = d.stock.trim() ? parseInt(d.stock, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) { editError[rel.id] = 'Stock must be positive.'; return; }
		editSaving[rel.id] = true;
		try {
			const updated = await onUpdate(rel.id, {
				price,
				stock,
				expires_at: d.expiresAt ? new Date(d.expiresAt).toISOString() : null,
				active: d.active,
				notes: d.notes.trim() || null
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
		active: 'bg-green-900/50 text-green-400 border-green-800',
		scheduled: 'bg-blue-900/50 text-blue-400 border-blue-800',
		expired: 'bg-red-900/50 text-red-400 border-red-800',
		archived: 'bg-surface-700 text-surface-500 border-surface-600'
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
				<label class="block text-[11px] uppercase tracking-wide text-surface-400 mb-1" for="brc-badge">Badge</label>
				<select
					id="brc-badge"
					bind:value={cfBadgeKey}
					class="w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-sm text-surface-100 focus:border-primary-500 outline-none"
				>
					<option value="">— select a badge —</option>
					{#each catalog as entry}
						<option value={entry.key}>[T{entry.tier}] {entry.title}</option>
					{/each}
				</select>
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
						class="w-full bg-surface-700 border border-surface-600 rounded-lg px-2 py-2 text-xs text-surface-100 focus:border-primary-500 outline-none"
					/>
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
											class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-xs text-surface-100 outline-none focus:border-primary-500" />
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
