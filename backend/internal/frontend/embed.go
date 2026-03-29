package frontend

import "embed"

// FS holds the compiled SvelteKit static build, embedded at compile time.
// In Docker the dist/ directory is populated by the frontend build stage before
// go build runs. For local development a placeholder index.html is used; run
// `task dev:frontend` in the repo root for the real Vite dev server instead.

//go:embed all:dist
var FS embed.FS
