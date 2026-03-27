import { browser } from '$app/environment';
import { writable } from 'svelte/store';

export type ThemeMode = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'scolymarket.theme';

export const themeMode = writable<ThemeMode>('system');

function prefersDark(): boolean {
	return browser && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme(mode: ThemeMode) {
	if (!browser) return;
	const effective = mode === 'system' ? (prefersDark() ? 'dark' : 'light') : mode;
	document.documentElement.setAttribute('data-theme', effective);
}

export function initTheme() {
	if (!browser) return;
	const raw = localStorage.getItem(STORAGE_KEY);
	const saved: ThemeMode = raw === 'light' || raw === 'dark' || raw === 'system' ? raw : 'system';
	themeMode.set(saved);
	applyTheme(saved);

	const media = window.matchMedia('(prefers-color-scheme: dark)');
	const onChange = () => {
		let current: ThemeMode = 'system';
		themeMode.update((v) => {
			current = v;
			return v;
		});
		if (current === 'system') applyTheme('system');
	};
	media.addEventListener('change', onChange);
}

export function setThemeMode(mode: ThemeMode) {
	themeMode.set(mode);
	if (browser) {
		localStorage.setItem(STORAGE_KEY, mode);
		applyTheme(mode);
	}
}

export function cycleThemeMode() {
	let next: ThemeMode = 'system';
	themeMode.update((mode) => {
		next = mode === 'light' ? 'dark' : mode === 'dark' ? 'system' : 'light';
		return next;
	});
	setThemeMode(next);
}
