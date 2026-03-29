import adapterStatic from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
export default {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapterStatic({
			fallback: 'index.html' // SPA mode — unknown paths serve index.html, Go router handles the rest
		})
	}
};

