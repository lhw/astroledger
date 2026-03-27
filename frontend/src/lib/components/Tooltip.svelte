<!--
  Thin wrapper around bits-ui Tooltip.
  Usage:
    <Tooltip text="Your tooltip text">
      <button>Hover me</button>
    </Tooltip>
-->
<script lang="ts">
	import { Tooltip } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		text,
		children,
		delay = 300,
		side = 'top'
	}: {
		text: string;
		children: Snippet;
		delay?: number;
		side?: 'top' | 'bottom' | 'left' | 'right';
	} = $props();
</script>

<Tooltip.Root delayDuration={delay}>
	<Tooltip.Trigger>
		{@render children()}
	</Tooltip.Trigger>
	<Tooltip.Portal>
		<Tooltip.Content
			{side}
			sideOffset={6}
			class="z-50 rounded px-2.5 py-1.5 text-xs bg-surface-900 text-surface-100
			       border border-surface-700 shadow-lg max-w-xs leading-snug"
		>
			{text}
		</Tooltip.Content>
	</Tooltip.Portal>
</Tooltip.Root>
