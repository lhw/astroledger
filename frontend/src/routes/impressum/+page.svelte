<script lang="ts">
	import { onMount } from 'svelte';

	function decodeFrag(arr: number[], key: number): string {
		return (arr || []).slice().reverse().map((c) => String.fromCharCode(c ^ key)).join('');
	}

	// Same operator as scid.my — encoded with the same XOR+reverse scheme
	const nameFrag  = [88, 79, 70, 70, 79, 125, 10, 4, 102];
	const emailFrag = [64, 84, 23, 93, 80, 90, 74, 121, 87, 80, 84, 93, 88];

	let contactMount: HTMLDivElement | null = null;

	function renderContact(el: HTMLDivElement) {
		const name  = decodeFrag(nameFrag,  42);
		const email = decodeFrag(emailFrag, 57);

		while (el.firstChild) el.removeChild(el.firstChild);

		const canvas = document.createElement('canvas');
		canvas.setAttribute('role', 'img');
		canvas.setAttribute('aria-label', 'Impressum name and email');
		canvas.style.width = '100%';
		canvas.style.height = 'auto';
		el.appendChild(canvas);

		const cs        = getComputedStyle(document.documentElement);
		const isDark    = document.documentElement.getAttribute('data-theme') === 'dark';
		const textColor = isDark ? 'oklch(88% 0.008 75)' : 'oklch(20% 0.007 75)';
		const goldColor = 'oklch(69% 0.140 75)'; // --color-primary-500

		const rect      = el.getBoundingClientRect();
		const cssWidth  = rect.width || parseFloat(cs.width) || 700;
		const padding   = 16;
		const fontSize  = Math.max(13, parseFloat(cs.fontSize) || 16);
		const lineH     = Math.round(fontSize * 1.45);
		const dpr       = window.devicePixelRatio || 1;
		const lines     = [name, email];

		canvas.width  = Math.ceil(cssWidth * dpr);
		canvas.height = Math.ceil((lines.length * lineH + padding * 2) * dpr);

		const ctx = canvas.getContext('2d');
		if (!ctx) return;

		ctx.scale(dpr, dpr);
		ctx.clearRect(0, 0, canvas.width, canvas.height);
		ctx.textBaseline = 'top';

		for (let i = 0; i < lines.length; i++) {
			const y      = padding + i * lineH;
			const jitter = Math.round((Math.random() - 0.5) * 6);

			if (i === 0) {
				ctx.font      = `bold ${Math.round(fontSize * 1.05)}px system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial`;
				ctx.fillStyle = goldColor;
			} else {
				ctx.font      = `${fontSize}px system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial`;
				ctx.fillStyle = textColor;
			}

			ctx.fillText(lines[i], padding + jitter, y);
		}
	}

	onMount(() => {
		if (!contactMount) return;

		const render = () => renderContact(contactMount!);
		render();

		let t: ReturnType<typeof setTimeout> | undefined;
		const handleResize = () => {
			if (t) clearTimeout(t);
			t = setTimeout(render, 120);
		};

		// Re-render when theme toggles
		const observer = new MutationObserver(() => render());
		observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });

		window.addEventListener('resize', handleResize);
		return () => {
			window.removeEventListener('resize', handleResize);
			observer.disconnect();
		};
	});
</script>

<svelte:head>
	<title>Impressum — AstroLedger</title>
</svelte:head>

<div class="mx-auto w-full max-w-2xl px-6 py-16">
	<div class="rounded-xl border border-surface-200 bg-[var(--card-bg)] p-8 shadow-sm">
		<h1 class="mb-8 text-2xl font-bold">Impressum</h1>

		<section class="mb-8">
			<h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">
				Angaben gemäß § 5 TMG
			</h2>
			<div bind:this={contactMount} class="min-h-[3.5rem]" aria-label="impressum-contact"></div>
			<noscript>
				<p class="text-sm leading-relaxed text-surface-600">
					Name and email are shown with JavaScript enabled.
				</p>
			</noscript>
		</section>

		<section class="mb-8">
			<h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">Kontakt</h2>
			<p class="text-sm text-surface-600">The email address is shown above for human readers.</p>
		</section>

		<section class="mb-8">
			<h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">
				Disclaimer / Haftungsausschluss
			</h2>
			<p class="text-sm leading-relaxed text-surface-600">
				AstroLedger is an <strong class="text-surface-700 dark:text-surface-300">unofficial</strong>
				fan project and is not affiliated with, authorized by, or endorsed by Cloud Imperium Games
				Corporation or Roberts Space Industries Corp. Star Citizen® is a registered trademark of
				Cloud Imperium Rights LLC.
			</p>
			<p class="mt-3 text-sm leading-relaxed text-surface-600">
				No real money is involved. AstroLedger uses only fictional in-game currency (bUEC) with
				no monetary value. No advertising revenue is generated.
			</p>
		</section>

		<section>
			<h2 class="mb-3 text-xs font-medium uppercase tracking-wider text-surface-500">
				Datenschutz / Privacy
			</h2>
			<p class="text-sm leading-relaxed text-surface-600">
				AstroLedger stores only what is necessary to operate the service: your SCID identity
				(username, RSI handle, citizen record, enlistment date) obtained via OpenID Connect login
				through <a href="https://scid.my" class="text-primary-600 hover:text-primary-500 transition-colors no-underline">scid.my</a>.
				No personal data is sold or shared with third parties. Data is stored on EU-based
				infrastructure. You can request account deletion by contacting the operator.
			</p>
		</section>
	</div>
</div>
