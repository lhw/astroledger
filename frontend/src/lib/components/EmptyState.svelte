<script lang="ts">
	let {
		message,
		card = true,
		padding = 'p-8',
		textSize = 'text-sm'
	}: {
		message: string;
		card?: boolean;
		padding?: string;
		textSize?: string;
	} = $props();
</script>

{#if card}
	<div class={`sc-card ${padding} text-center text-surface-400 ${textSize}`}>{message}</div>
{:else}
	<div class={`text-surface-400 text-center ${padding} ${textSize}`}>{message}</div>
{/if}