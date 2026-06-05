<script lang="ts">
	import {
		getMyPositions,
		getMyTrades,
		getMyBadges,
		logout,
		setActiveBadge,
		listBotTokens,
		createBotToken,
		revokeBotToken,
		ApiClientError
	} from '$lib/api';
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import BadgeHangarSection from '$lib/components/me/BadgeHangarSection.svelte';
	import BotTokensSection from '$lib/components/me/BotTokensSection.svelte';
	import PositionsSection from '$lib/components/me/PositionsSection.svelte';
	import ProfileHeader from '$lib/components/me/ProfileHeader.svelte';
	import TradesSection from '$lib/components/me/TradesSection.svelte';
	import type { Position, TradeWithMarket, Badge, BotApiToken } from '$lib/types';

	let positions = $state<Position[]>([]);
	let trades = $state<TradeWithMarket[]>([]);
	let badges = $state<Badge[]>([]);
	let botTokens = $state<BotApiToken[]>([]);
	let activeBadgeKey = $state<string>('');
	let badgeSaving = $state(false);
	let loading = $state(true);
	let positionsLoading = $state(false);
	let tradesLoading = $state(false);
	let badgesLoading = $state(false);
	let botLoading = $state(false);
	let botCreateLoading = $state(false);
	let botError = $state('');
	let botName = $state('');
	let botCanTrade = $state(true);
	let botCanCreateMarkets = $state(false);
	let createdTokenValue = $state('');
	let error = $state('');

	let dataLoaded = $state(false);

	$effect(() => {
		// Wait until auth state is resolved (not undefined) before fetching.
		if ($currentUser === undefined) return;

		if ($currentUser === null) {
			loading = false;
			return;
		}

		// Only fetch once per page load.
		if (dataLoaded) return;
		dataLoaded = true;

		activeBadgeKey = $currentUser.active_badge_key ?? '';
		loading = false;
		error = '';

		positionsLoading = true;
		tradesLoading = true;
		badgesLoading = true;
		botLoading = true;

		const jobs = [
			(async () => {
				try {
					positions = await getMyPositions();
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					positionsLoading = false;
				}
			})(),
			(async () => {
				try {
					trades = await getMyTrades(0);
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					tradesLoading = false;
				}
			})(),
			(async () => {
				try {
					badges = await getMyBadges();
				} catch (e) {
					error ||= e instanceof Error ? e.message : String(e);
				} finally {
					badgesLoading = false;
				}
			})(),
			(async () => {
				try {
					botTokens = await listBotTokens();
				} catch (e) {
					botError = e instanceof Error ? e.message : String(e);
				} finally {
					botLoading = false;
				}
			})()
		];

		Promise.allSettled(jobs);
	});

	const openPositions = $derived(positions.filter(p => p.market_status !== 'resolved'));
	const resolvedPositions = $derived(positions.filter(p => p.market_status === 'resolved'));

	async function pickBadge(key: string) {
		// Toggle off if already active.
		const newKey = activeBadgeKey === key ? '' : key;
		badgeSaving = true;
		try {
			const res = await setActiveBadge(newKey);
			activeBadgeKey = res.active_badge_key;
		} catch {
			// ignore — UI stays consistent on failure
		} finally {
			badgeSaving = false;
		}
	}

	async function createToken() {
		botError = '';
		createdTokenValue = '';
		const name = botName.trim();
		if (name.length < 3 || name.length > 64) {
			botError = 'Token name must be between 3 and 64 characters.';
			return;
		}

		botCreateLoading = true;
		try {
		const created = await createBotToken({
			name,
			can_trade: botCanTrade,
			can_create_markets: botCanCreateMarkets
		});
			createdTokenValue = created.token;
			botName = '';
			botTokens = await listBotTokens();
		} catch (e) {
			botError = e instanceof ApiClientError ? e.message : e instanceof Error ? e.message : String(e);
		} finally {
			botCreateLoading = false;
		}
	}

	async function removeToken(id: number) {
		botError = '';
		try {
			await revokeBotToken(id);
			botTokens = botTokens.filter((t) => t.id !== id);
		} catch (e) {
			botError = e instanceof ApiClientError ? e.message : e instanceof Error ? e.message : String(e);
		}
	}

	async function copyCreatedToken() {
		if (!createdTokenValue) return;
		try {
			await navigator.clipboard.writeText(createdTokenValue);
		} catch {
			// Clipboard errors are non-fatal; token remains visible.
		}
	}
</script>

<svelte:head>
	<title>My Profile — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-10">
	{#if !$isLoggedIn}
		<div class="text-center py-16">
			<h1 class="text-2xl font-bold text-surface-900 mb-4">Not logged in</h1>
			<a href="/auth/login" class="btn preset-filled-primary-500 uppercase tracking-wider text-xs">Login with SCID</a>
		</div>
	{:else if loading}
		<div class="text-surface-400 text-center py-16 text-sm">Loading profile…</div>
	{:else}
		{@const user = $currentUser}

		{#if user}
			<ProfileHeader {user} onLogout={logout} />
		{/if}

		{#if error}
			<div class="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm mb-6">
				{error}
			</div>
		{/if}

		<BadgeHangarSection
			{badges}
			{badgesLoading}
			{activeBadgeKey}
			{badgeSaving}
			onPickBadge={pickBadge}
		/>

		<BotTokensSection
			{botLoading}
			{botCreateLoading}
			{botError}
			bind:botName
			bind:botCanTrade
			bind:botCanCreateMarkets
			{createdTokenValue}
			{botTokens}
			onCreateToken={createToken}
			onRemoveToken={removeToken}
			onCopyCreatedToken={copyCreatedToken}
		/>

		<PositionsSection {positionsLoading} {openPositions} {resolvedPositions} />

		<TradesSection {tradesLoading} {trades} />
	{/if}
</div>
