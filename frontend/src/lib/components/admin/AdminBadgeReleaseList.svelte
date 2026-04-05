<script lang="ts">
	import { formatBadgeInsurance, getBadgeTierLabel } from '$lib/badges';
	import type { BadgeInsurance } from '$lib/badges';
	import type { AdminBadgeRelease } from '$lib/types';
	import { ApiClientError } from '$lib/api';
	import BadgePill from '$lib/components/BadgePill.svelte';

	let {
		releases = $bindable(),
		onUpdate,
		onArchive
	}: {
		releases: AdminBadgeRelease[];
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

	type EditDraft = {
		price: string;
		stock: string;
		expiresAt: string;
		indefinite: boolean;
		active: boolean;
		notes: string;
		insurance: BadgeInsurance;
	};

	let editDrafts = $state<Record<number, EditDraft>>({});
	let editSaving = $state<Record<number, boolean>>({});
	let editError = $state<Record<number, string | null>>({});
	let archiveConfirm = $state<number | null>(null);

	const TIER_COLORS: Record<number, string> = {
		1: 'text-surface-400',
		2: 'text-green-400',
		3: 'text-blue-400',
		4: 'text-purple-400',
		5: 'text-yellow-400'
	};

	const STATUS_CHIP: Record<string, string> = {
		active: 'release-status-chip release-status-chip-active',
		scheduled: 'release-status-chip release-status-chip-scheduled',
		expired: 'release-status-chip release-status-chip-expired',
		archived: 'release-status-chip release-status-chip-archived'
	};

	function stringValue(value: string | number | null | undefined): string {
		if (value == null) return '';
		return String(value);
	}

	function releaseStatus(rel: AdminBadgeRelease): 'active' | 'scheduled' | 'expired' | 'archived' {
		if (!rel.active) return 'archived';
		const now = new Date();
		if (new Date(rel.released_at) > now) return 'scheduled';
		if (rel.expires_at && new Date(rel.expires_at) < now) return 'expired';
		return 'active';
	}

	function fmtDate(iso: string) {
		return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	const sortedReleases = $derived(
		[...releases].sort((a, b) => {
			const statusA = releaseStatus(a);
			const statusB = releaseStatus(b);
			const order = { active: 0, scheduled: 1, expired: 2, archived: 3 };
			if (order[statusA] !== order[statusB]) return order[statusA] - order[statusB];
			return new Date(b.released_at).getTime() - new Date(a.released_at).getTime();
		})
	);

	function startEdit(rel: AdminBadgeRelease) {
		editDrafts[rel.id] = {
			price: String(rel.price),
			stock: rel.stock != null ? String(rel.stock) : '',
			expiresAt: rel.expires_at ? rel.expires_at.slice(0, 16) : '',
			indefinite: rel.expires_at == null,
			active: rel.active,
			notes: rel.notes ?? '',
			insurance: (rel.insurance || '') as BadgeInsurance
		};
	}

	function cancelEdit(id: number) {
		const { [id]: _removed, ...rest } = editDrafts;
		editDrafts = rest;
		const { [id]: _error, ...errorRest } = editError;
		editError = errorRest;
	}

	async function saveEdit(rel: AdminBadgeRelease) {
		const draft = editDrafts[rel.id];
		if (!draft) return;
		editError[rel.id] = null;
		const price = parseInt(draft.price, 10);
		if (!Number.isInteger(price) || price < 0) {
			editError[rel.id] = 'Price must be a non-negative integer.';
			return;
		}
		const stockRaw = stringValue(draft.stock).trim();
		const stock = stockRaw ? parseInt(stockRaw, 10) : null;
		if (stock !== null && (!Number.isInteger(stock) || stock <= 0)) {
			editError[rel.id] = 'Stock must be positive.';
			return;
		}
		editSaving[rel.id] = true;
		try {
			const updated = await onUpdate(rel.id, {
				price,
				stock,
				expires_at: draft.indefinite || !draft.expiresAt ? null : new Date(draft.expiresAt).toISOString(),
				active: draft.active,
				notes: draft.notes.trim() || null,
				insurance: draft.insurance
			});
			releases = releases.map((release) => (release.id === rel.id ? updated : release));
			cancelEdit(rel.id);
		} catch (err) {
			editError[rel.id] = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			editSaving[rel.id] = false;
		}
	}

	async function confirmArchive(id: number) {
		editSaving[id] = true;
		try {
			await onArchive(id);
			releases = releases.map((release) => (release.id === id ? { ...release, active: false } : release));
			cancelEdit(id);
		} catch (err) {
			editError[id] = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			editSaving[id] = false;
			archiveConfirm = null;
		}
	}
</script>

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
						<div class="space-y-3">
							<div class="flex items-center gap-2 flex-wrap">
								<span class="text-[10px] font-bold uppercase tracking-widest {TIER_COLORS[rel.tier]}">
									{getBadgeTierLabel(rel.tier)}
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
								<label class="block text-[11px] text-surface-400 sm:col-span-2">Insurance
									<select bind:value={draft.insurance}
										class="mt-1 w-full bg-surface-700 border border-surface-600 rounded px-2 py-1 text-xs text-surface-100 outline-none focus:border-primary-500">
										<option value="">None</option>
										<option value="6w">6 Weeks</option>
										<option value="120w">120 Weeks</option>
										<option value="lti">LTI</option>
									</select>
								</label>
								<label class="block text-[11px] text-surface-400 sm:col-span-2">Notes
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
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 min-w-0 space-y-2">
								<div class="flex items-center gap-2 flex-wrap">
									<BadgePill tier={rel.tier} title={rel.title} />
									<code class="text-[10px] text-surface-500 font-mono">{rel.badge_key}</code>
									<span class="inline-block px-1.5 py-0.5 rounded text-[10px] font-bold uppercase border {STATUS_CHIP[status]}">
										{status}
									</span>
								</div>

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
											{formatBadgeInsurance(rel.insurance)}
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

<style>
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