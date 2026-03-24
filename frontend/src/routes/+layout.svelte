<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { AppShell, AppBar, AppRailAnchor } from '@skeletonlabs/skeleton';
	import { initAuth, currentUser, isLoggedIn, isModerator } from '$lib/stores/auth';
	import { loginWithSCID, logout } from '$lib/api';

	onMount(async () => {
		await initAuth();
	});
</script>

<AppShell>
	<svelte:fragment slot="header">
		<AppBar background="bg-surface-800" padding="px-4 py-2">
			<svelte:fragment slot="lead">
				<a href="/" class="flex items-center gap-2 no-underline">
					<span class="text-primary-400 font-bold text-xl tracking-tight">⚖ ScolyMarket</span>
					<span class="text-surface-400 text-xs hidden sm:inline">
						The galaxy's finest prediction market
					</span>
				</a>
			</svelte:fragment>

			<svelte:fragment slot="trail">
				<nav class="flex items-center gap-4">
					<a href="/markets" class="text-surface-200 hover:text-primary-400 text-sm transition-colors">
						Markets
					</a>
					<a href="/leaderboard" class="text-surface-200 hover:text-primary-400 text-sm transition-colors">
						Leaderboard
					</a>

					{#if $isModerator}
						<a href="/mod" class="text-warning-400 hover:text-warning-300 text-sm transition-colors">
							Mod Queue
						</a>
					{/if}

					{#if $isLoggedIn && $currentUser}
						<span class="text-primary-400 text-sm font-semibold">
							{$currentUser.balance.toLocaleString()} ScollyBucks™
						</span>
						<a href="/me" class="btn btn-sm variant-filled-surface">
							{$currentUser.display_name}
						</a>
						<button on:click={logout} class="btn btn-sm variant-ghost">
							Logout
						</button>
					{:else if $currentUser === null}
						<button on:click={loginWithSCID} class="btn btn-sm variant-filled-primary">
							Login with SCID
						</button>
					{:else}
						<span class="text-surface-400 text-sm">Loading…</span>
					{/if}
				</nav>
			</svelte:fragment>
		</AppBar>
	</svelte:fragment>

	<!-- Page content -->
	<slot />

	<svelte:fragment slot="footer">
		<div class="text-center text-surface-500 text-xs py-4 border-t border-surface-700">
			ScolyMarket™ — No real money. No real promises.
			All ScollyBucks™ are imaginary and so is Star Citizen's release date.
		</div>
	</svelte:fragment>
</AppShell>
