<script lang="ts">
	import { adminListBadgeDefinitions, adminCreateBadgeDefinition, adminUpdateBadgeDefinition, ApiClientError } from '$lib/api';
	import { formatBadgeInsurance, getBadgeTierSymbol } from '$lib/badges';
	import type { AdminBadgeDefinition } from '$lib/types';

	let badgeDefs = $state<AdminBadgeDefinition[]>([]);
	let defsLoading = $state(false);
	let defsError = $state<string | null>(null);
	let bootstrapped = $state(false);

	let newDefKey = $state('');
	let newDefTitle = $state('');
	let newDefDesc = $state('');
	let newDefTier = $state(1);
	let newDefIcon = $state('');
	let newDefInsurance = $state('');
	let newDefSaving = $state(false);
	let newDefError = $state<string | null>(null);

	let editingDefKey = $state<string | null>(null);
	let editDefTitle = $state('');
	let editDefDesc = $state('');
	let editDefTier = $state(1);
	let editDefIcon = $state('');
	let editDefInsurance = $state('');
	let editDefSaving = $state(false);
	let editDefError = $state<string | null>(null);

	$effect(() => {
		if (bootstrapped) return;
		bootstrapped = true;
		void loadDefs();
	});

	async function loadDefs() {
		defsLoading = true;
		defsError = null;
		try {
			badgeDefs = await adminListBadgeDefinitions();
		} catch (err) {
			defsError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			defsLoading = false;
		}
	}

	function badgeDefIcon(def: AdminBadgeDefinition) {
		return def.icon.trim() || getBadgeTierSymbol(def.tier);
	}

	async function submitNewDef() {
		newDefError = null;
		const key = newDefKey.trim();
		const title = newDefTitle.trim();
		const description = newDefDesc.trim();
		const icon = newDefIcon.trim();
		if (!key || !title) {
			newDefError = 'Key and title are required.';
			return;
		}
		if (!/^[a-z0-9_-]+$/.test(key)) {
			newDefError = 'Key must be lowercase letters, digits, underscores, or hyphens.';
			return;
		}
		newDefSaving = true;
		try {
			const definition = await adminCreateBadgeDefinition({
				key,
				title,
				description,
				tier: newDefTier,
				icon,
				insurance: newDefInsurance
			});
			badgeDefs = [...badgeDefs, definition];
			newDefKey = '';
			newDefTitle = '';
			newDefDesc = '';
			newDefTier = 1;
			newDefIcon = '';
			newDefInsurance = '';
		} catch (err) {
			newDefError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			newDefSaving = false;
		}
	}

	function startEditDef(def: AdminBadgeDefinition) {
		editingDefKey = def.key;
		editDefTitle = def.title;
		editDefDesc = def.description;
		editDefTier = def.tier;
		editDefIcon = def.icon;
		editDefInsurance = def.insurance;
		editDefError = null;
	}

	async function saveEditDef(key: string) {
		editDefError = null;
		const title = editDefTitle.trim();
		if (!title) {
			editDefError = 'Title is required.';
			return;
		}
		editDefSaving = true;
		try {
			const updated = await adminUpdateBadgeDefinition(key, {
				title,
				description: editDefDesc.trim(),
				tier: editDefTier,
				icon: editDefIcon.trim(),
				insurance: editDefInsurance
			});
			badgeDefs = badgeDefs.map((definition) => (definition.key === key ? updated : definition));
			editingDefKey = null;
		} catch (err) {
			editDefError = err instanceof ApiClientError ? err.message : String(err);
		} finally {
			editDefSaving = false;
		}
	}
</script>

{#if defsError}
	<div class="mb-4 px-4 py-3 rounded bg-red-950 border border-red-800 text-red-300 text-sm">{defsError}</div>
{:else if defsLoading}
	<p class="text-surface-500 text-sm">Loading…</p>
{:else}
	<section class="bg-surface-800 border border-surface-700 rounded-xl p-6 mb-6 space-y-4">
		<h2 class="text-sm font-semibold text-surface-100 uppercase tracking-widest">New Badge Definition</h2>
		<div class="grid grid-cols-2 gap-3">
			<div>
				<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-key">Key <span class="text-red-400">*</span></label>
				<input id="nd-key" type="text" bind:value={newDefKey} placeholder="e.g. explorer_badge" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none font-mono" />
			</div>
			<div>
				<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-title">Title <span class="text-red-400">*</span></label>
				<input id="nd-title" type="text" bind:value={newDefTitle} placeholder="Explorer" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
			</div>
			<div class="col-span-2">
				<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-desc">Description</label>
				<input id="nd-desc" type="text" bind:value={newDefDesc} placeholder="Awarded for exploring the unknown" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
			</div>
			<div>
				<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-tier">Tier (1–5)</label>
				<input id="nd-tier" type="number" min="1" max="5" bind:value={newDefTier} class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
			</div>
			<div>
				<label class="block text-surface-400 text-xs uppercase tracking-wide mb-1" for="nd-icon">Icon (emoji or short text)</label>
				<input id="nd-icon" type="text" bind:value={newDefIcon} placeholder="🔭" class="input w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100 text-sm focus:border-primary-500 outline-none" />
			</div>
		</div>
		{#if newDefError}<p class="text-red-400 text-xs">{newDefError}</p>{/if}
		<button onclick={submitNewDef} disabled={newDefSaving} class="btn preset-filled-primary-500 tracking-wider uppercase text-xs disabled:opacity-50">
			{newDefSaving ? 'Creating…' : 'Create Badge Definition'}
		</button>
	</section>

	<section class="bg-surface-800 border border-surface-700 rounded-xl overflow-hidden">
		<table class="w-full table-fixed text-left admin-defs-table">
			<thead class="bg-surface-700">
				<tr class="text-surface-400 text-xs uppercase tracking-widest">
					<th class="px-4 py-3 w-[18%]">Key</th>
					<th class="px-4 py-3 w-[18%]">Title</th>
					<th class="px-4 py-3 w-[8%]">Tier</th>
					<th class="px-4 py-3 w-[28%]">Description</th>
					<th class="px-4 py-3 w-[10%] text-center">Icon</th>
					<th class="px-4 py-3 w-[12%]">Insurance</th>
					<th class="px-4 py-3 w-[10%]"></th>
				</tr>
			</thead>
			<tbody>
				{#each badgeDefs as def}
					<tr class="border-t border-surface-700 hover:bg-surface-700/40 transition-colors">
						{#if editingDefKey === def.key}
							<td class="px-4 py-3 font-mono text-xs text-surface-400 break-all align-top">{def.key}</td>
							<td class="px-4 py-3">
								<input type="text" bind:value={editDefTitle} class="input w-full bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
							</td>
							<td class="px-4 py-3 align-top">
								<input type="number" min="1" max="5" bind:value={editDefTier} class="input w-16 bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
							</td>
							<td class="px-4 py-3">
								<input type="text" bind:value={editDefDesc} class="input w-full bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
							</td>
							<td class="px-4 py-3 align-top">
								<input type="text" bind:value={editDefIcon} class="input w-16 bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none" />
								<p class="text-surface-500 text-[10px] mt-1 text-center">{getBadgeTierSymbol(editDefTier)}</p>
							</td>
							<td class="px-4 py-3 align-top">
								<select bind:value={editDefInsurance} class="bg-surface-700 border border-primary-600 rounded px-2 py-1 text-surface-100 text-xs outline-none">
									<option value="">None</option>
									<option value="6w">6 Weeks</option>
									<option value="120w">120 Weeks</option>
									<option value="lti">LTI</option>
								</select>
							</td>
							<td class="px-4 py-3 align-top">
								<div class="flex gap-2 items-center">
									{#if editDefError}<span class="text-red-400 text-xs">{editDefError}</span>{/if}
									<button onclick={() => saveEditDef(def.key)} disabled={editDefSaving} class="btn preset-filled-primary-500 text-xs py-1 px-3 disabled:opacity-50">{editDefSaving ? '…' : 'Save'}</button>
									<button onclick={() => { editingDefKey = null; }} class="text-surface-500 hover:text-surface-200 text-xs">Cancel</button>
								</div>
							</td>
						{:else}
							<td class="px-4 py-3 font-mono text-xs text-surface-400 break-all align-top">{def.key}</td>
							<td class="px-4 py-3 text-surface-100 text-sm font-medium align-top">{def.title}</td>
							<td class="px-4 py-3 align-top">
								<span class="def-tier-pip tier-{def.tier}">{def.tier}</span>
							</td>
							<td class="px-4 py-3 text-surface-400 text-xs align-top">
								<div class="line-clamp-2 break-words" title={def.description}>{def.description || '—'}</div>
							</td>
							<td class="px-4 py-3 text-center align-top">
								<span class="def-icon-chip" title={def.icon.trim() ? 'Custom icon' : 'Tier default icon'}>{badgeDefIcon(def)}</span>
							</td>
							<td class="px-4 py-3 align-top">
								{#if def.insurance}
									<span class="text-xs px-2 py-0.5 rounded-full bg-amber-900/40 border border-amber-800/50 text-amber-300">
										{formatBadgeInsurance(def.insurance, 'long')}
									</span>
								{:else}
									<span class="text-surface-600 text-xs">—</span>
								{/if}
							</td>
							<td class="px-4 py-3 align-top text-right">
								{#if def.purchasable}
									<button onclick={() => startEditDef(def)} class="text-xs text-primary-400 hover:text-primary-200 font-semibold uppercase tracking-wide">Edit</button>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</section>
{/if}

<style>
	.def-tier-pip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		border-radius: 50%;
		font-size: 0.65rem;
		font-weight: 700;
	}
	.def-tier-pip.tier-1 { background: #f5ede0; color: #b08050; border: 1.5px solid #d4b896; }
	.def-tier-pip.tier-2 { background: #fde68a; color: #78350f; border: 1.5px solid #f59e0b; }
	.def-tier-pip.tier-3 { background: linear-gradient(135deg, #fde047, #ea580c); color: #fff; border: 1.5px solid #f59e0b; }
	.def-tier-pip.tier-4 { background: #fbbf24; color: #1c1008; border: 1.5px solid #fde68a; }
	.def-tier-pip.tier-5 { background: conic-gradient(from 0deg, #ff0080, #ff8c00, #ffd700, #00ff88, #ff0080); color: #fff; border: 1.5px solid rgba(255,255,255,0.5); }
	.def-icon-chip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 9999px;
		background: color-mix(in oklch, var(--color-primary-100) 70%, white 30%);
		border: 1px solid color-mix(in oklch, var(--color-primary-300) 60%, var(--color-surface-300) 40%);
		color: var(--color-surface-900);
		font-size: 1rem;
		line-height: 1;
	}
	.admin-defs-table td,
	.admin-defs-table th {
		vertical-align: top;
	}
</style>