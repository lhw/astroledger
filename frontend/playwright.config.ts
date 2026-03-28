import { defineConfig, devices } from '@playwright/test';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

// Load backend/.env so analytics tests can reach GoatCounter with the API key
// when running against the full dev stack (task dev).
function loadBackendEnv() {
	const __filename = fileURLToPath(import.meta.url);
	const __dirname = path.dirname(__filename);
	const envFile = path.resolve(__dirname, '../backend/.env');
	if (!fs.existsSync(envFile)) return;
	for (const line of fs.readFileSync(envFile, 'utf-8').split('\n')) {
		const trimmed = line.trim();
		if (!trimmed || trimmed.startsWith('#')) continue;
		const eq = trimmed.indexOf('=');
		if (eq < 0) continue;
		const key = trimmed.slice(0, eq).trim();
		const val = trimmed.slice(eq + 1).trim();
		// Don't overwrite values already set in the shell environment.
		if (!(key in process.env)) process.env[key] = val;
	}
}
loadBackendEnv();

export default defineConfig({
	testDir: './tests',
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: process.env.CI ? 1 : undefined,
	reporter: 'html',
	use: {
		baseURL: 'http://localhost:5173',
		trace: 'on-first-retry',
		screenshot: 'only-on-failure'
	},
	projects: [
        {
        name: 'firefox',
        use: { ...devices['Desktop Firefox'] },
        },
	],
	webServer: {
		command: 'npm run dev',
		url: 'http://localhost:5173',
		reuseExistingServer: !process.env.CI
	}
});
