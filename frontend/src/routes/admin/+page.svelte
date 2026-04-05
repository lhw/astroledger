<script lang="ts">
	import {
		adminGetBadgeCatalog,
		adminListBadgeReleases,
		adminCreateBadgeRelease,
		adminUpdateBadgeRelease,
		adminArchiveBadgeRelease,
		ApiClientError
	} from '$lib/api';
	import type { BadgeCatalogEntry, AdminBadgeRelease } from '$lib/types';
	import { authReady, isAdmin } from '$lib/stores/auth';
	import AdminAnalyticsPanel from '$lib/components/admin/AdminAnalyticsPanel.svelte';
	import AdminBadgeDefinitionsPanel from '$lib/components/admin/AdminBadgeDefinitionsPanel.svelte';
	import AdminOperationsPanel from '$lib/components/admin/AdminOperationsPanel.svelte';
	import BadgeReleaseCalendar from '$lib/components/BadgeReleaseCalendar.svelte';
	import TabBar from '$lib/components/TabBar.svelte';

	let activeTab = $state<'operations' | 'analytics' | 'badges' | 'defs'>('operations');

	let catalog = $state<BadgeCatalogEntry[]>([]);
	let releases = $state<AdminBadgeRelease[]>([]);
	let badgesLoading = $state(false);
	let badgesError = $state<string | null>(null);

	async function loadBadges() {
		badgesLoading = true;
		badgesError = null;
		try {
			[catalog, releases] = await Promise.all([
				adminGetBadgeCatalog(),
				adminListBadgeReleases()
			]);
		} catch (e) {
			badgesError = e instanceof ApiClientError ? e.message : String(e);
		} finally {
			badgesLoading = false;
		}
	}

	function switchToBadges() {
		activeTab = 'badges';
		if (!catalog.length && !releases.length) loadBadges();
	}

	// API wrappers forwarded into BadgeReleaseCalendar
	async function onBadgeCreate(body: Parameters<typeof adminCreateBadgeRelease>[0]) {
		return adminCreateBadgeRelease(body);
	}
	async function onBadgeUpdate(id: number, body: Parameters<typeof adminUpdateBadgeRelease>[1]) {
		return adminUpdateBadgeRelease(id, body);
	}
	async function onBadgeArchive(id: number) {
		await adminArchiveBadgeRelease(id);
	}
</script>

<svelte:head>
	<title>Admin Panel — AstroLedger</title>
</svelte:head>

<div class="admin-shell max-w-4xl mx-auto px-6 py-10">
	{#if !$authReady}
		<p class="text-surface-400 text-sm">Checking authentication…</p>
	{:else if !$isAdmin}
		<div class="bg-surface-800 border border-surface-700 rounded-xl p-8 text-center space-y-3">
			<p class="text-red-400 text-sm font-semibold">Access denied.</p>
			<p class="text-surface-500 text-xs">This page is restricted to administrators.</p>
		</div>
	{:else}
	<h1 class="text-2xl font-bold text-primary-400 tracking-widest uppercase mb-6">Admin Panel</h1>

	<div class="mb-8">
		<TabBar
			tabs={[
				{ id: 'operations', label: 'Operations' },
				{ id: 'analytics', label: 'Analytics' },
				{ id: 'badges', label: 'Badges' },
				{ id: 'defs', label: 'Badge Defs' }
			]}
			bind:active={activeTab}
			onTabChange={(id) => {
				if (id === 'badges') switchToBadges();
			}}
		/>
	</div>

	{#if activeTab === 'operations'}
		<AdminOperationsPanel />
	{/if}

	{#if activeTab === 'analytics'}
		<AdminAnalyticsPanel />
	{/if}

	<!-- ── Badges Tab ───────────────────────────────────────────────── -->
	{#if activeTab === 'badges'}
		{#if badgesError}
			<div class="mb-4 px-4 py-3 rounded bg-red-950 border border-red-800 text-red-300 text-sm">{badgesError}</div>
		{:else if badgesLoading}
			<p class="text-surface-500 text-sm">Loading…</p>
		{:else}
			<BadgeReleaseCalendar
				{catalog}
				bind:releases
				onCreate={onBadgeCreate}
				onUpdate={onBadgeUpdate}
				onArchive={onBadgeArchive}
			/>
		{/if}
	{/if}

	{#if activeTab === 'defs'}
		<AdminBadgeDefinitionsPanel />
	{/if}
	{/if}
</div>

<style>
	:global(:root:not([data-theme='dark']) .admin-shell .bg-surface-800) {
		background: var(--card-bg) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .bg-surface-700) {
		background: color-mix(in oklch, var(--color-surface-50) 84%, var(--color-primary-100) 16%) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .border-surface-700),
	:global(:root:not([data-theme='dark']) .admin-shell .border-surface-600) {
		border-color: var(--color-surface-300) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-100) {
		color: var(--color-surface-900) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-300) {
		color: var(--color-surface-700) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-400) {
		color: var(--color-surface-500) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-surface-500) {
		color: var(--color-surface-600) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .hover\:bg-surface-700:hover) {
		background: color-mix(in oklch, var(--color-surface-100) 85%, var(--color-primary-100) 15%) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .text-primary-300),
	:global(:root:not([data-theme='dark']) .admin-shell .text-primary-400) {
		color: var(--color-primary-700) !important;
	}

	:global(:root:not([data-theme='dark']) .admin-shell .hover\:text-surface-100:hover) {
		color: var(--color-surface-900) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .bg-surface-800) {
		background: var(--card-bg) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .bg-surface-700) {
		background: color-mix(in oklch, var(--card-bg) 84%, var(--color-surface-300) 16%) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .border-surface-700),
	:global(:root[data-theme='dark'] .admin-shell .border-surface-600) {
		border-color: var(--color-surface-300) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-300) {
		color: var(--color-surface-700) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-100) {
		color: var(--color-surface-900) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-surface-400),
	:global(:root[data-theme='dark'] .admin-shell .text-surface-500) {
		color: var(--color-surface-600) !important;
	}

	:global(:root[data-theme='dark'] .admin-shell .text-primary-300),
	:global(:root[data-theme='dark'] .admin-shell .text-primary-400) {
		color: var(--color-primary-500) !important;
	}
</style>


