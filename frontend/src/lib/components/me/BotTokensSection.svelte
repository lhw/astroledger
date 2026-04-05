<script lang="ts">
	import type { BotApiToken } from '$lib/types';

	let {
		botLoading,
		botCreateLoading,
		botError,
		botName = $bindable(),
		botCanTrade = $bindable(),
		createdTokenValue,
		botTokens,
		onCreateToken,
		onRemoveToken,
		onCopyCreatedToken
	}: {
		botLoading: boolean;
		botCreateLoading: boolean;
		botError: string;
		botName: string;
		botCanTrade: boolean;
		createdTokenValue: string;
		botTokens: BotApiToken[];
		onCreateToken: () => Promise<void>;
		onRemoveToken: (id: number) => Promise<void>;
		onCopyCreatedToken: () => Promise<void>;
	} = $props();
</script>

<section class="mb-8">
	<div class="flex items-center justify-between mb-2">
		<h2 class="text-xs font-bold uppercase tracking-[0.15em] text-surface-600">Bot API Tokens</h2>
	</div>
	<p class="text-[11px] text-surface-500 mb-3">
		Create scoped tokens for read/trade bot access. Tokens are shown once at creation.
	</p>

	<div class="sc-card p-4 mb-3 space-y-3">
		<div>
			<label class="block text-[10px] uppercase tracking-wider text-surface-500 mb-1" for="bot-token-name">Token Name</label>
			<input id="bot-token-name" bind:value={botName} class="sc-input text-sm" placeholder="e.g. Backtesting Bot" maxlength="64" />
		</div>

		<label class="flex items-center gap-2 text-sm text-surface-600">
			<input type="checkbox" bind:checked={botCanTrade} />
			Allow trading scope
		</label>

		<div class="flex items-center gap-2">
			<button onclick={onCreateToken} disabled={botCreateLoading} class="btn btn-sm preset-filled-primary-500 uppercase tracking-wider text-xs">
				{botCreateLoading ? 'Creating…' : 'Create Token'}
			</button>
		</div>

		{#if createdTokenValue}
			<div class="rounded border border-primary-300 bg-primary-100/40 p-3 space-y-2">
				<p class="text-[11px] font-semibold text-primary-700 uppercase tracking-wider">Copy now - this value will not be shown again</p>
				<div class="text-xs break-all font-mono text-surface-800">{createdTokenValue}</div>
				<button onclick={onCopyCreatedToken} class="btn btn-xs preset-outlined uppercase tracking-wider text-[10px]">Copy Token</button>
			</div>
		{/if}
	</div>

	{#if botError}
		<div class="p-3 rounded border border-red-500/40 bg-red-500/10 text-red-400 text-sm mb-3">{botError}</div>
	{/if}

	{#if botLoading}
		<div class="text-surface-400 text-sm py-2">Loading tokens…</div>
	{:else if botTokens.length === 0}
		<div class="text-surface-400 text-sm py-2">No active bot tokens yet.</div>
	{:else}
		<div class="space-y-2">
			{#each botTokens as token}
				<div class="sc-card p-3 flex items-center justify-between gap-3">
					<div class="min-w-0">
						<p class="text-sm font-semibold text-surface-800 truncate">{token.name}</p>
						<p class="text-[11px] text-surface-500 font-mono">{token.token_prefix}...</p>
						<p class="text-[11px] text-surface-500 mt-1">
							{token.can_trade ? 'read + trade' : 'read only'} · created {new Date(token.created_at).toLocaleDateString()}
							{#if token.last_used_at} · used {new Date(token.last_used_at).toLocaleDateString()}{/if}
						</p>
					</div>
					<button onclick={() => onRemoveToken(token.id)} class="btn btn-xs preset-outlined text-red-500 border-red-300 hover:bg-red-50 uppercase tracking-wider text-[10px]">
						Revoke
					</button>
				</div>
			{/each}
		</div>
	{/if}
</section>