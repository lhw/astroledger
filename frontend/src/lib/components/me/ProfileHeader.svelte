<script lang="ts">
	import UserAvatar from '$lib/components/UserAvatar.svelte';
	import type { User } from '$lib/types';

	let {
		user,
		onLogout
	}: {
		user: User;
		onLogout: () => Promise<void>;
	} = $props();
</script>

<div class="flex items-start justify-between mb-8 pb-6 border-b border-surface-200">
	<div class="flex items-start gap-4">
		<UserAvatar src={user.avatar_url} name={user.display_name} size={56} />
		<div>
			<p class="text-xs font-bold uppercase tracking-[0.15em] text-primary-600 mb-1">Profile</p>
			<h1 class="text-2xl font-bold text-surface-900">{user.display_name}</h1>
			<p class="text-surface-500 text-sm mt-1">
				Joined {new Date(user.created_at).toLocaleDateString()}
			</p>
			{#if user.is_moderator || user.is_admin}
				<span class="sc-tag mt-2 inline-flex">
					{user.is_admin ? 'Admin' : 'Moderator'}
				</span>
			{/if}
		</div>
	</div>
	<div class="text-right">
		<div class="text-3xl font-bold text-primary-600">{user.balance.toLocaleString()}</div>
		<div class="text-surface-500 text-xs uppercase tracking-widest font-semibold">bUEC</div>
		<button onclick={onLogout} class="border border-surface-300 text-surface-500 hover:border-surface-500 hover:text-surface-700 transition-colors rounded px-3 py-1 text-xs uppercase tracking-wider mt-3">
			Logout
		</button>
	</div>
</div>

{#if user.rsi_handle || user.is_rsi_verified}
	<section class="mb-8">
		<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600 mb-4">RSI Identity</h2>
		<div class="sc-card px-4 py-4 flex flex-wrap gap-6 items-center">
			{#if user.rsi_handle}
				<div>
					<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Handle</p>
					<a
						href="https://robertsspaceindustries.com/citizens/{user.rsi_handle}"
						target="_blank"
						rel="noopener noreferrer"
						class="text-primary-600 font-semibold hover:underline"
					>
						{user.rsi_handle}
					</a>
				</div>
			{/if}
			{#if user.rsi_citizen_record}
				<div>
					<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Citizen Record</p>
					<p class="text-surface-800 font-medium">#{user.rsi_citizen_record}</p>
				</div>
			{/if}
			{#if user.rsi_enlisted}
				<div>
					<p class="text-[10px] uppercase tracking-widest text-surface-400 font-semibold mb-0.5">Enlisted</p>
					<p class="text-surface-800 font-medium">{user.rsi_enlisted}</p>
				</div>
			{/if}
			{#if user.is_rsi_verified}
				<div class="ml-auto">
					<span class="sc-tag text-green-700 bg-green-50 border-green-200">✓ RSI Verified</span>
				</div>
			{/if}
		</div>
	</section>
{/if}