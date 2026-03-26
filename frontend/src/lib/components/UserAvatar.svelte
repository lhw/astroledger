<script lang="ts">
	/** URL of the avatar image (from OIDC picture claim). Null → initials fallback. */
	let { src, name, size = 24 }: { src?: string | null; name: string; size?: number } = $props();

	// Falls back to initials when the image fails to load (e.g. Firefox blocking
	// third-party images, CORS restrictions, or a bad URL).
	let imgError = $state(false);

	// Reset the error flag whenever the src changes.
	$effect(() => {
		src;
		imgError = false;
	});

	// Up to two initials from the display name words.
	const initials = name
		.split(/\s+/)
		.map((w) => w[0])
		.join('')
		.slice(0, 2)
		.toUpperCase();

	// Deterministic hue from the name so each user gets a consistent colour.
	const hue = [...name].reduce((acc, ch) => acc + ch.charCodeAt(0), 0) % 360;
</script>

{#if src && !imgError}
	<img
		{src}
		alt=""
		width={size}
		height={size}
		class="rounded-full object-cover shrink-0"
		style="width:{size}px;height:{size}px"
		onerror={() => { imgError = true; }}
	/>
{:else}
	<span
		class="rounded-full flex items-center justify-center font-bold shrink-0 text-white select-none"
		style="width:{size}px;height:{size}px;background:hsl({hue},55%,40%);font-size:{Math.round(size * 0.4)}px"
		aria-label={name}
	>
		{initials}
	</span>
{/if}
