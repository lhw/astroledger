# ScolyMarket — Copilot Instructions

## Project Overview

ScolyMarket is a satirical prediction market web app for betting fake credits on Star Citizen bug fixes and general events. No real money is involved. The tone is humorous and self-aware.

See `plan.md` for the full design document.

## Architecture

- **Backend:** Go 1.26 with chi router. SQLite database via sqlc (type-safe SQL codegen). OIDC auth via scid.my.
- **Frontend:** SvelteKit with Skeleton UI, TypeScript, Vite. Custom luxury gold theme.
- **Monorepo:** `backend/` and `frontend/` directories at the repo root.

## Code Conventions

### Go (backend)

- Use Go modules. Dependencies managed via `go.mod`.
- Use `golangci-lint` for linting. Enable `errcheck`, `staticcheck`, `govet`, `gosimple`, `unused` at minimum.
- Use `Task` (taskfile.dev) as the task runner. All build/test/lint/migrate commands go in `Taskfile.yml`.
- All exported functions must have Go doc comments. Internal helpers don't need them unless complex.
- Use `internal/` for all private application code (handler, service, middleware, config, database, models).
- Use `cmd/server/main.go` as the application entry point.
- **Handlers** (`internal/handler/`) are thin: parse request, validate input, call service, write JSON response. No business logic.
- **Services** (`internal/service/`) contain all business logic. No HTTP concerns (no `http.Request`, no response writing).
- **Middleware** (`internal/middleware/`) for cross-cutting concerns: auth session check, logging, rate limiting.
- Use `context.Context` as the first parameter for functions doing I/O. Propagate the request context.
- Always handle errors explicitly — never `_` discard errors. Wrap with `fmt.Errorf("context: %w", err)`.
- Use `defer` for cleanup (closing rows, response bodies, files). `defer rows.Close()` immediately after query.
- Use `database/sql` transactions for multi-row mutations (trading, payouts). `defer tx.Rollback()` then commit on success.
- Use `log/slog` (stdlib) for structured logging. Never log tokens, secrets, or sensitive user data.
- SQL queries go in `queries/` as `.sql` files. Run `sqlc generate` to produce Go code in `internal/db/`. Never edit generated files by hand.
- Database migrations go in `migrations/` as numbered SQL files for goose.
- Use `modernc.org/sqlite` (pure Go, no CGo) as the SQLite driver.
- Currency (ScollyBucks™) is always integer arithmetic. Never use float for balances or costs.
- Use HTTP status codes correctly: 200 OK, 201 Created, 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 422 Unprocessable Entity, 429 Too Many Requests, 500 Internal Server Error.
- Tests use `testing` + `testify`. Prefer table-driven tests. Tests go in `tests/` or alongside the code as `_test.go` files.

### TypeScript / Svelte (frontend)

- Use TypeScript strictly — `strict: true` in tsconfig, avoid `any`. Use `unknown` + type guards when needed.
- Use SvelteKit file-based routing under `src/routes/`.
- Use Skeleton UI components wherever possible instead of custom CSS.
- API calls go through a centralized fetch wrapper in `src/lib/api.ts`.
- Shared state uses Svelte stores in `src/lib/stores/`.
- Define TypeScript interfaces for all API response shapes in `src/lib/types.ts`.
- Format with Prettier, lint with ESLint.
- Sanitize user-generated markdown with DOMPurify before rendering with `{@html}`.
- Use `+page.server.ts` load functions for authenticated data fetching (cookies are forwarded server-side).
- The custom luxury gold theme is defined in `src/theme.ts` and applied via Skeleton's theming system.

### General

- Prefer simple, readable code over clever abstractions.
- This is a joke project for ~200 users. Don't over-engineer.
- SQLite is the database — no need for PostgreSQL, Redis, or message queues.
- No real money or crypto is involved. Ever.

## Authentication

- **No local passwords.** All auth is via SCID (scid.my), an OpenID Connect provider.
- **Confidential client** with Authorization Code Flow + PKCE.
- The Go backend handles the full OIDC flow: redirect to scid.my → callback → token exchange → session cookie.
- SCID tokens (`SCID_CLIENT_ID`, `SCID_CLIENT_SECRET`) are stored in environment variables only — never in code or version control.
- After OIDC login, the backend issues its own session as an httpOnly, Secure, SameSite=Lax cookie.
- The frontend determines auth state by calling `GET /api/me` — it never reads or modifies the session cookie directly.
- Validate OIDC `state` parameter, `id_token` signature (via JWKS), and claims (`iss`, `aud`, `exp`) on every callback.

## Key Domain Concepts

- **Market:** A yes/no question that users bet on. Has a deadline and resolution criteria.
- **AMM (LMSR):** Automated Market Maker using Logarithmic Market Scoring Rule. Provides liquidity so users can always buy/sell.
- **ScollyBucks™:** The play currency. Users start with 1,000 and get 200 weekly.
- **Shares:** YES or NO shares in a market. Prices range 1–99. Winning shares pay 100 at resolution.
- **Moderation:** Markets go through auto-filter → mod queue → active. Mods resolve markets.
- **Auto-filter:** Keyword/regex rules that auto-reject markets about banned topics (player kills, harassment, etc.).

## Security Notes

- No local passwords — auth delegated to SCID (OIDC).
- Validate OIDC state, PKCE, id_token signature, and claims on every auth callback.
- Session cookies: httpOnly, Secure, SameSite=Lax. No tokens in localStorage.
- All SQL is parameterized via sqlc — no string concatenation in queries.
- Validate and sanitize all user input at the handler boundary.
- Rate-limit auth callbacks, market submission, and trading endpoints.
- Sanitize markdown rendering on the frontend to prevent XSS (DOMPurify).
- Use integer arithmetic for all currency operations.
- CORS configured to allow only the frontend origin.
- HTTPS enforced in production via Caddy.
