# AstroLedger — Plan

> A satirical prediction market for Star Citizen bug fixing and general events.
> No real money. Just vibes, credits, and the collective delusion that 3.x will be stable.

---

## 1. Concept

AstroLedger is a tongue-in-cheek prediction market (à la Polymarket/Manifold) where players bet fake credits on whether Star Citizen bugs will be fixed, features will ship, and other community events will happen. The tone is irreverent and self-aware — this is a joke about the state of the game, not a serious forecasting tool.

### How Prediction Markets Work (simplified)

- A **market** is a yes/no question: *"Will the Reclaimer elevator bug be fixed by 4.0?"*
- Users buy **YES** or **NO** shares. Share prices range from **1–99 credits** and reflect the crowd's estimated probability.
- When the market **resolves** (the question is answered), YES shares pay out 100 credits each if the answer was yes, 0 if no — and vice versa for NO shares.
- The price you pay is your cost basis. Buy YES at 30, outcome is YES → you profit 70. Outcome is NO → you lose 30.
- A simple **automated market maker (AMM)** sets initial prices and provides liquidity so users can always trade. We'll use a straightforward constant-product or logarithmic market scoring rule (LMSR) — nothing fancy.

### Core Loop

1. User logs in via SCID (Star Citizen Identity — OpenID Connect via scid.my).
2. User browses open markets or submits a new one.
3. User buys YES or NO shares with their credit balance.
4. Market resolves at deadline (or manually by moderator).
5. Payouts are distributed. Leaderboard updates.

---

## 2. Scope & Constraints

| Constraint | Decision |
|---|---|
| Real money | **None.** Play credits only. |
| Scale | ~100–300 users max. Single server, SQLite is fine. |
| Lifespan | Short-lived joke project. Minimal ops. |
| Auth | **SCID (scid.my)** — OIDC login. No local passwords. Confidential client with Authorization Code Flow + PKCE. |
| Mobile | Responsive web is enough. No native app. |
| Legal | No real gambling, no real money, no crypto. Satirical fair use. Theme inspired by luxury/gold aesthetics, not copying any CIG IP directly. |

---

## 3. Authentication — SCID (scid.my)

### Overview

All user authentication is delegated to **SCID** (https://scid.my), a community-run OpenID Connect (OIDC) identity provider built on [Pocket ID](https://github.com/pocket-id/pocket-id). SCID verifies users' RSI (Roberts Space Industries) accounts by having them place a short token in their public RSI profile bio — no RSI passwords are shared. Once verified, SCID provides standard OIDC claims including the user's verified RSI handle and org memberships.

AstroLedger has **no local password storage, no registration form, and no email verification**. Users click "Login with SCID", authenticate via passkey on scid.my, and are redirected back. This dramatically simplifies our auth surface and eliminates password-related security concerns entirely.

### Client Type: Confidential Client (Authorization Code Flow + PKCE)

**Why confidential, not public:**

Since AstroLedger has a Go backend server, we use a **confidential client**. This is the correct and most secure choice because:

1. **The backend can securely store the client secret** — it never reaches the browser or client-side code.
2. **Authorization Code Flow** is the most secure standard OIDC flow. The authorization code is exchanged for tokens server-side, so tokens are never exposed in browser URLs or JavaScript.
3. **PKCE (Proof Key for Code Exchange)** is added as defense-in-depth against code interception attacks, even though the confidential client already mitigates this. PKCE is a best practice for all OAuth2 flows per current security recommendations (RFC 9126, OAuth 2.1 draft).
4. A **public client** would only be appropriate for a pure single-page app with no backend. Since we have a server, there is no reason to weaken security by omitting the client secret.

A public client (e.g., PKCE-only SPA flow) is designed for applications that cannot keep a secret — mobile apps, browser-only SPAs. AstroLedger's Go backend is a trusted server that can and should hold the secret.

### OIDC Flow (step by step)

```
┌──────────┐                    ┌──────────────┐                    ┌──────────┐
│  Browser  │                    │  Go Backend   │                    │  scid.my  │
└─────┬────┘                    └──────┬───────┘                    └─────┬────┘
      │  1. Click "Login with SCID"    │                                  │
      │ ──────────────────────────────→│                                  │
      │                                │  2. Generate state + PKCE verifier│
      │                                │     Store in server-side session  │
      │  3. Redirect to scid.my/authorize                                 │
      │    (client_id, redirect_uri, state, code_challenge, scope)        │
      │ ←─────────────────────────────│                                  │
      │ ─────────────────────────────────────────────────────────────────→│
      │                                │                                  │
      │  4. User authenticates via passkey on scid.my                     │
      │                                │                                  │
      │  5. Redirect back to /auth/callback?code=...&state=...           │
      │ ←────────────────────────────────────────────────────────────────│
      │ ──────────────────────────────→│                                  │
      │                                │  6. Verify state matches session  │
      │                                │  7. Exchange code + client_secret │
      │                                │     + code_verifier for tokens    │
      │                                │ ────────────────────────────────→│
      │                                │  8. Receive id_token + access_token│
      │                                │ ←───────────────────────────────│
      │                                │  9. Validate id_token signature   │
      │                                │     via JWKS from scid.my         │
      │                                │  10. Extract claims (sub, handle) │
      │                                │  11. Create/update local user     │
      │                                │  12. Issue session cookie         │
      │  13. Set httpOnly cookie, redirect to app                         │
      │ ←─────────────────────────────│                                  │
```

### SCID Configuration Required

Register AstroLedger as an OIDC client on scid.my with:

| Setting | Value |
|---|---|
| **Client Type** | Confidential |
| **Client Name** | AstroLedger |
| **Redirect URI** | `https://<domain>/auth/callback` |
| **Scopes** | `openid email profile` |
| **Grant Type** | Authorization Code |

The `Client ID` and `Client Secret` from SCID go into backend environment variables (`SCID_CLIENT_ID`, `SCID_CLIENT_SECRET`). **Never commit these to source control.**

### OIDC Claims & Local User Model

When a user logs in via SCID, the `id_token` provides claims like:

- `sub` — unique user identifier (stable across sessions, used as our foreign key)
- `preferred_username` or custom claim — the RSI handle
- `email` — email if available/verified

On first login, the backend creates a local `User` record linked by `sub`. On subsequent logins, the user record is updated if claims have changed (e.g., display name). No passwords are ever stored locally.

### Session Management

After OIDC login, the backend issues its own **session** to the browser:

- **Session cookie:** `httpOnly`, `Secure`, `SameSite=Lax`. Contains a signed session token (JWT or opaque ID mapping to server-side session).
- **Session duration:** 7 days, sliding expiration (refreshed on active use).
- **Logout:** Clears the session cookie. Optionally redirects to SCID's end-session endpoint for full SSO logout.
- **No access tokens stored client-side.** The SCID access token is only used server-side during the code exchange and for optional userinfo calls. It is not sent to the browser.

### Security Considerations for Auth

- **Always validate the `state` parameter** returned from SCID against the server-side session to prevent CSRF.
- **Always validate the `id_token` signature** using SCID's JWKS endpoint. Cache the JWKS keys with a reasonable TTL (e.g., 1 hour).
- **Always validate `id_token` claims:** check `iss` (must be scid.my), `aud` (must be our client_id), `exp` (must not be expired), `nonce` if used.
- **Store `SCID_CLIENT_SECRET` in environment variables only** — never in code, config files committed to git, or client-side code.
- **Use HTTPS everywhere** — SCID requires secure context, and cookies must be `Secure`.
- **Rate-limit the `/auth/callback` endpoint** to prevent abuse.

---

## 4. Features

### 4.1 Credits & Economy

- **Starting balance:** Every new account gets **1,000 ScollyBucks™** on first login.
- **Weekly payout:** All active users receive **200 ScollyBucks™** every Monday. Keeps the economy alive and lets losing players recover.
- **No negative balances.** You can't bet what you don't have.
- **Leaderboard** ranked by total portfolio value (credits + value of held shares).

### 4.2 Markets

A market consists of:

| Field | Description |
|---|---|
| `title` | The question, e.g. *"Will CIG fix the invisible Carrack ramp by 4.1?"* |
| `description` | Context, links to Issue Council, screenshots, lore, shitposting — whatever. Markdown supported. |
| `category` | One of: `Bug Fixes`, `Feature Delivery`, `Patch Timing`, `CIG Drama`, `Community Events`, `Meta / Other` |
| `resolution_criteria` | Clear conditions for YES/NO. e.g. "Fixed means the bug is no longer on the known issues list for the live build." |
| `resolution_deadline` | When the market closes and must be resolved. |
| `created_by` | The submitting user. |
| `status` | `pending_review` → `active` → `resolved` / `cancelled` |

#### Market Lifecycle

```
User submits market
        │
        ▼
  ┌─────────────┐
  │ Auto-filter  │──── rejected (spam, banned topics) ──→ discarded with reason
  └─────┬───────┘
        │ passed
        ▼
  ┌─────────────┐
  │  Mod Queue   │──── moderator rejects ──→ discarded with reason
  └─────┬───────┘
        │ approved
        ▼
  ┌─────────────┐
  │   Active     │  ← trading happens here
  └─────┬───────┘
        │ deadline reached or manual trigger
        ▼
  ┌─────────────┐
  │  Resolution  │  ← moderator picks YES / NO / CANCELLED
  └─────────────┘
        │
        ▼
    Payouts
```

### 4.3 Moderation System

#### Auto-filter (first pass)

Before a submitted market enters the mod queue, it runs through automated checks:

- **Duplicate detection:** Fuzzy match against existing active/pending market titles. Flag if similarity > threshold.
- **Banned topic keywords:** Reject markets about player-specific kills, doxxing, real-world harassment, slurs, or otherwise undesirable content. Configurable keyword/regex list.
- **Rate limiting:** Max N market submissions per user per day to prevent spam floods.
- **Minimum description length:** Reject very low-effort submissions (e.g. title only, no description, no resolution criteria).

Auto-filtered markets are rejected with an explanation shown to the submitter. They can revise and resubmit.

#### Mod Queue (second pass)

Markets that pass auto-filter land in a moderator dashboard:

- Moderators see pending markets with title, description, category, resolution criteria.
- Actions: **Approve**, **Reject** (with reason), **Edit & Approve** (fix wording/criteria).
- Any registered user can be promoted to moderator by an admin.

#### Reporting

- Users can **report** active markets or individual comments for moderator review.
- Reported items surface in the mod dashboard.

### 4.4 Trading

- **AMM-backed:** Every market is initialized with a liquidity pool. We use a simple LMSR (Logarithmic Market Scoring Rule) with a configurable liquidity parameter `b`. This determines how sensitive prices are to trades — lower `b` = more volatile, higher `b` = more stable prices.
- **Buy/Sell:** Users can buy or sell YES/NO shares at any time while the market is active.
- **Price display:** Current price shown as a percentage (e.g., YES at 72 means the crowd thinks there's a 72% chance).
- **Order history:** Each user can see their own trade history per market.
- **No limit orders, no order book.** AMM only. Keeps it simple.

### 4.5 Resolution

- At or after the deadline, a moderator resolves the market as **YES**, **NO**, or **CANCELLED**.
- **YES/NO:** Winning shares pay 100 credits. Losing shares pay 0.
- **CANCELLED:** All shares refunded at purchase price (e.g., if the question becomes unanswerable).
- Resolved markets are archived and visible but no longer tradeable.
- There should be a grace period / reminder system so markets don't languish unresolved.

### 4.6 User Profiles & Leaderboard

- **Profile:** Display name (from SCID / RSI handle), join date, current balance, portfolio (open positions), trade history, submitted markets.
- **Leaderboard:** Ranked by total portfolio value. Updated in near-real-time.
- **Badges / titles** (optional fun): e.g. "Bug Prophet", "Eternal Optimist", "CIG Apologist". Based on trading patterns, win streaks, etc.

### 4.7 UI Pages

| Page | Purpose |
|---|---|
| **Home / Market List** | Browse active markets, filter by category, sort by volume/deadline/newest |
| **Market Detail** | See question, description, price chart, buy/sell widget, comments |
| **Submit Market** | Form to propose a new market |
| **Mod Dashboard** | Review pending markets, handle reports (mod-only) |
| **Leaderboard** | Top traders ranked by portfolio value |
| **Profile** | User's portfolio, trade history, submitted markets |
| **Login** | "Login with SCID" button — redirects to scid.my for OIDC auth |

Global navigation uses a compact fold-out menu on small screens to prevent header overflow while keeping full horizontal nav on desktop.

---

## 5. Tech Stack

### Backend — Go

| Tool | Purpose |
|---|---|
| **Go 1.26** | Language. Statically typed, fast compilation, excellent stdlib, single binary deployment. |
| **chi** (`github.com/go-chi/chi/v5`) | HTTP router & middleware. Lightweight, idiomatic, composable. Perfect for a project of this size — not too bare (stdlib only), not too heavy (Gin/Echo). Chi is built on `net/http` so all stdlib middleware and handlers are compatible. |
| **SQLite** via `modernc.org/sqlite` | Database — pure Go SQLite driver (no CGo required). Single file, zero config, perfect for ~200 users. |
| **sqlc** (`github.com/sqlc-dev/sqlc`) | Generates type-safe Go code from SQL queries. You write SQL, sqlc generates Go structs and query functions. No ORM magic, no reflection, full control over queries. Compile-time type safety. |
| **goose** (`github.com/pressly/goose/v3`) | Database migrations. Simple, well-maintained, supports SQL and Go migrations. |
| **coreos/go-oidc** (`github.com/coreos/go-oidc/v3`) | OIDC client library for authenticating with scid.my. Handles discovery, JWKS validation, and token verification. |
| **golang.org/x/oauth2** | OAuth2 client — used alongside go-oidc for the Authorization Code Flow token exchange. |
| **golang-jwt/jwt** (`github.com/golang-jwt/jwt/v5`) | Creating and validating our own session JWTs (issued after OIDC login). **Not** for validating SCID tokens — go-oidc handles that. |
| **slog** (stdlib `log/slog`) | Structured logging. Built into Go since 1.21 — no external dependency needed. |
| **go-chi/httprate** | Rate limiting middleware for chi. Used on auth callbacks and market submission endpoints. |
| **golangci-lint** | Linting. Runs multiple linters (staticcheck, errcheck, govet, etc.) in one pass. |
| **testify** (`github.com/stretchr/testify`) | Test assertions and mocking helpers. Makes tests more readable. |
| **Task** (`taskfile.dev`) | Task runner. Defines build, test, lint, migrate, and dev commands in a `Taskfile.yml`. Simpler than Makefiles. |

#### Why Go instead of Python/FastAPI?

- **Single binary deployment.** No virtualenvs, no `pip install`, no Python version management on the server. Build → `scp` → run.
- **Compiled type safety.** Catches entire classes of bugs at compile time that Python would only catch at runtime.
- **Lower resource usage.** Go's memory footprint and startup time are ideal for a small VPS.
- **Excellent stdlib.** `net/http`, `encoding/json`, `crypto`, `log/slog` — less dependency on third-party packages.
- **SCID / Pocket ID itself is written in Go**, so the ecosystem is a natural fit.

#### Go Best Practices to Follow

- **Always handle errors explicitly.** Never use `_` to discard errors. Wrap errors with `fmt.Errorf("context: %w", err)` to maintain stack context.
- **Use `context.Context`** as the first parameter of functions that do I/O (database queries, HTTP calls). Propagate the request context from chi handlers.
- **Use `defer` for cleanup** — closing database rows, HTTP response bodies, file handles. Always `defer rows.Close()` immediately after query execution.
- **Use `database/sql` transactions** for any operation that modifies multiple rows (trading, payouts). Call `tx.Rollback()` in a defer and only commit at the end on success.
- **Validate all input at the handler boundary.** Parse and validate request bodies/params before passing to service functions. Return `400 Bad Request` for malformed input, `422 Unprocessable Entity` for valid-but-rejected input.
- **Never use string concatenation for SQL.** Always use `?` placeholders. sqlc enforces this by design.
- **Keep handlers thin.** HTTP handlers parse the request, call service functions, and write the response. Business logic lives in service packages.
- **Use struct embedding and interfaces sparingly.** Keep the type hierarchy flat.
- **Prefer returning errors over panicking.** Reserve `panic` for truly unrecoverable situations (missing required config at startup).
- **Use `golangci-lint`** in CI and locally. Enable at minimum: `errcheck`, `staticcheck`, `govet`, `gosimple`, `unused`.
- **Write table-driven tests.** Go's `testing` package + `testify` assertions. Use subtests for related test cases.

### Frontend — SvelteKit + Skeleton UI

| Tool | Purpose |
|---|---|
| **SvelteKit 2.53+** | Framework — file-based routing, SSR, adapter-auto. **Svelte 5** runes syntax (`$state`, `$derived`, `$props`, `{@render children()}`). |
| **Svelte 5.43+** | Component language. Runes-compatible. Layout uses `$props()` / `{@render children()}`. Route pages can mix Svelte 4-compat syntax. |
| **Vite 7** + **`@sveltejs/vite-plugin-svelte` ^6** | Bundler — embedded in SvelteKit, v6 plugin required for Svelte 5 compatibility. |
| **Tailwind CSS v4** via `@tailwindcss/vite` | Utility-first CSS. No `tailwind.config.js` or PostCSS needed — pure Vite plugin approach. |
| **Skeleton UI v4** (`@skeletonlabs/skeleton`) | Component library — polished, themeable, accessible. Gold luxury theme via CSS custom properties in `app.css`. |
| **TypeScript** | Type safety. Strict mode — no `any` types. |
| **Playwright** (`@playwright/test`) | End-to-end browser tests. Mocked API calls via `page.route()` — tests don't require a running backend. |
| **DOMPurify** | Sanitize user-generated markdown before rendering with `{@html}`. |

#### SvelteKit Best Practices to Follow

- **Use TypeScript strictly.** Set `strict: true` in `tsconfig.json`. Never use `any` — use `unknown` and narrow with type guards if the type is truly uncertain. Define explicit interfaces for all API response shapes.
- **Use SvelteKit's server-side rendering** (`+page.server.ts` load functions) for data that should be fetched before the page renders (market lists, user profile). This improves perceived performance and SEO.
- **All API calls go through `src/lib/api.ts`** — a centralized fetch wrapper that handles base URL, cookies/credentials, error responses, and JSON parsing. Never use raw `fetch` in components.
- **Use Skeleton UI components** (buttons, cards, modals, tables, toasts, drawers, app bars) instead of writing custom HTML/CSS. Theme them via Skeleton's theming system, don't override component styles directly.
- **Store auth state in Svelte stores** (`src/lib/stores/`). The auth store tracks whether the user is logged in and their basic profile info. Populated from the session cookie on page load via a layout load function.
- **Session cookie is `httpOnly`** — the frontend cannot read or modify it. Auth state is determined by calling a backend endpoint (e.g., `GET /api/me`) on initial page load. If the cookie is valid, the backend returns user info; if not, it returns 401 and the frontend shows logged-out state.
- **Sanitize user-generated markdown** before rendering. Use a library like `marked` + `DOMPurify` to parse markdown and strip dangerous HTML. Never render raw user HTML with `{@html}` without sanitization.
- **Use SvelteKit `+page.server.ts`** for server-side authentication checks on protected routes (mod dashboard, profile). Redirect to login if the user is not authenticated.
- **Follow Svelte 5 runes syntax** if using Svelte 5 (`$state`, `$derived`, `$effect`). If on Svelte 4, use reactive declarations (`$:`) and stores.
- **Prefer `+page.server.ts` (server load) over `+page.ts` (universal load)** for API calls that include cookies/auth, since server load runs on the SvelteKit server and can forward cookies to the backend.

### Dev & Deployment

| Tool | Purpose |
|---|---|
| **Docker Compose** | Single command local dev & deployment. Two services: Go backend, SvelteKit frontend. |
| **Caddy** | Reverse proxy + auto HTTPS. Caddy is simpler than nginx for this scale and handles TLS certificates automatically. |
| **GitHub Actions** | CI — lint, test, build for both backend and frontend. |
| **SQLite backup cron** | Simple `cp` of the db file. Enterprise-grade disaster recovery™. |

---

## 6. Visual Theme — Luxury Gold

### Design Philosophy

The visual identity draws inspiration from **luxury spacecraft showrooms and high-end touring vessels** — think sleek, refined, premium materials. The aesthetic is "private yacht captain's lounge" rather than "industrial dockworker." Clean lines, generous whitespace, warm gold accents on deep dark surfaces. The vibe of a company that would describe a dogfighter as "a flowing work of technical art and mankind's most perfect killing machine" — sophistication with a hint of absurdity, which fits the satirical tone of AstroLedger perfectly.

Design cues: BMW's visual language, luxury watch branding, premium automotive configurators, high-end hotel booking sites. Smooth, aerodynamic, designed — "thousands of hours went into the development of every individual element."

**This is NOT a Star Citizen skin.** We don't use any CIG trademarks, logos, ship names, or branded assets in the UI chrome. The gold-and-dark palette is generic luxury design that works in any sci-fi context. Star Citizen references appear only in user-generated market content.

### Color Palette

The theme is implemented via Skeleton UI's custom theme generator (`@skeletonlabs/skeleton`). Key token values:

| Token | Hex | Usage |
|---|---|---|
| **Primary** | `#D4A843` (warm gold) | Buttons, active states, links, price highlights, primary actions |
| **Primary hover** | `#C49B38` (deeper gold) | Hover/pressed states |
| **Primary light** | `#E8C96A` (champagne) | Subtle highlights, badges, progress bars, "YES" share accent |
| **Surface (base)** | `#0C0E14` (near-black with warm undertone) | Page background — deep charcoal, not pure black |
| **Surface (raised)** | `#161921` (dark gunmetal) | Cards, panels, mod dashboard items |
| **Surface (overlay)** | `#1F232E` (elevated charcoal) | Modals, dropdowns, tooltips |
| **Accent / Secondary** | `#A8B4C4` (silver/platinum) | Secondary actions, metadata, ScollyBucks™ currency display |
| **Success** | `#4ADE80` (green-400) | Positive outcomes, "YES" probability going up |
| **Error / Danger** | `#F87171` (red-400) | Negative outcomes, "NO" shares, destructive actions — softer red, not harsh |
| **Warning** | `#FBBF24` (amber-400) | Warnings, auto-filter rejection notices |
| **Text (primary)** | `#F0F0F0` (warm white) | Main body text — slightly warm, not pure clinical white |
| **Text (muted)** | `#8A919E` (cool gray) | Secondary text, timestamps, metadata |
| **Border** | `#252A36` (subtle charcoal) | Card borders, dividers — understated, not heavy |

### Typography

- **Headings:** An elegant, slightly thin sans-serif like **Outfit**, **Sora**, or **DM Sans**. Clean and modern with a premium feel. Slightly wider letter-spacing on uppercase labels.
- **Body text:** Inter or system font stack. Legibility is paramount — luxury means clarity, not ornamentation.
- **Monospace:** JetBrains Mono or Fira Code for data display (share counts, prices, IDs).
- **Currency/numbers:** Tabular (fixed-width) numerals where available, so columns of prices align cleanly.

### UI Characteristics

- **Dark mode only.** No light theme. Deep, warm darks — this is a premium lounge, not a sterile control room.
- **Warm gold on deep dark surfaces.** Gold elements should feel rich and intentional, not garish. Less is more — use gold for primary actions and key data, not everything.
- **Generous whitespace and padding.** Luxury means breathing room. Cards should feel spacious, not cramped. Content areas wide with comfortable margins.
- **Subtle gradient overlays** on hero sections or key cards — a very faint warm-to-cool gradient suggesting depth, like polished metal catching light.
- **Cards with thin gold top-border or left-border accents** for market items. Borders are 1-2px, not heavy. Subtle drop shadows for elevation.
- **Smooth transitions and hover effects.** Buttons and interactive elements should have graceful transitions (150-250ms). No jarring state changes.
- **Clean, airy data layouts** — the leaderboard and mod dashboard should feel like a premium dashboard or portfolio tracker. Well-spaced rows, clear hierarchy, status indicators using small colored dots rather than loud badges.
- **Iconography:** Minimal, line-based icons (e.g., Lucide or Heroicons outline style). No chunky or filled icons.
- **Mobile responsive.** The premium feel translates to mobile — cards stack elegantly, nav collapses to a clean drawer with the same warm gold accents.

### Skeleton UI Theme Implementation

Skeleton UI uses a custom theme format. The theme is defined as a TypeScript object and applied globally. Key steps for the coding agent:

1. Use Skeleton's theme generator or define a custom theme in `src/theme.ts` matching the color tokens above.
2. Apply the theme in `+layout.svelte` via Skeleton's `AppShell` component.
3. Use Skeleton's built-in dark mode — set it as the default and only mode.
4. Override specific component styles only when the default Skeleton styling doesn't achieve the luxury feel (prefer theme tokens over CSS overrides).
5. Consider a self-hosted elegant font (Outfit or DM Sans) loaded in `app.css` for headings. Fall back to Inter/system for body.

---

## 7. Project Structure

```
astroledger/
├── backend/
│   ├── go.mod                          # Go module definition
│   ├── go.sum
│   ├── Taskfile.yml                    # Task runner commands (build, test, lint, dev, migrate)
│   ├── sqlc.yaml                       # sqlc configuration
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                 # Application entry point, config loading, server startup
│   ├── internal/                       # Private application code (not importable by other modules)
│   │   ├── config/
│   │   │   └── config.go              # Environment variable loading, app configuration struct
│   │   ├── database/
│   │   │   └── database.go            # SQLite connection setup, migration runner
│   │   ├── models/                     # Go structs for domain objects (generated by sqlc + hand-written)
│   │   │   ├── user.go
│   │   │   ├── market.go
│   │   │   ├── trade.go
│   │   │   └── moderation.go
│   │   ├── handler/                    # HTTP handlers (thin — parse request, call service, write response)
│   │   │   ├── auth.go                # /auth/login, /auth/callback, /auth/logout, /auth/me
│   │   │   ├── markets.go            # Market CRUD, listing, filtering
│   │   │   ├── trading.go            # Buy/sell shares
│   │   │   ├── users.go              # Profile, leaderboard
│   │   │   └── moderation.go         # Mod queue, approve/reject, reports
│   │   ├── service/                    # Business logic (no HTTP concerns)
│   │   │   ├── auth.go               # OIDC flow, session management, user creation
│   │   │   ├── amm.go                # LMSR cost function, price calculation, trade execution
│   │   │   ├── market.go             # Market lifecycle management
│   │   │   ├── credits.go            # Balance operations, weekly payouts
│   │   │   ├── autofilter.go         # Keyword matching, duplicate detection, rate limiting
│   │   │   └── resolution.go         # Market resolution, payout distribution
│   │   ├── middleware/                 # chi middleware
│   │   │   ├── auth.go               # Session cookie validation, user injection into context
│   │   │   ├── ratelimit.go          # Rate limiting configuration
│   │   │   └── logging.go            # Request logging via slog
│   │   └── db/                         # sqlc generated code (DO NOT EDIT BY HAND)
│   │       ├── query.sql.go           # Generated query functions
│   │       ├── models.go             # Generated model structs
│   │       └── db.go                  # Generated DBTX interface
│   ├── migrations/                     # goose SQL migrations
│   │   ├── 001_initial_schema.sql
│   │   └── ...
│   ├── queries/                        # sqlc SQL query files (source of truth)
│   │   ├── users.sql
│   │   ├── markets.sql
│   │   ├── trades.sql
│   │   └── moderation.sql
│   └── tests/                          # Test files mirror internal/ structure
│       ├── service/
│       │   ├── amm_test.go
│       │   └── ...
│       └── handler/
│           └── ...
├── frontend/
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── src/
│   │   ├── theme.ts                   # Skeleton UI custom theme (Luxury Gold)
│   │   ├── app.html
│   │   ├── app.css                    # Global styles, font imports
│   │   ├── routes/
│   │   │   ├── +layout.svelte        # AppShell, nav bar, theme application
│   │   │   ├── +layout.server.ts     # Root layout server load — check auth via /api/me
│   │   │   ├── +page.svelte          # Home: market list
│   │   │   ├── market/
│   │   │   │   ├── [id]/+page.svelte          # Market detail + trading widget
│   │   │   │   ├── [id]/+page.server.ts       # Load market data server-side
│   │   │   │   └── submit/+page.svelte        # Submit new market form
│   │   │   ├── leaderboard/+page.svelte
│   │   │   ├── profile/+page.svelte
│   │   │   ├── mod/+page.svelte               # Moderator dashboard (protected)
│   │   │   └── login/+page.svelte             # "Login with SCID" button
│   │   ├── lib/
│   │   │   ├── api.ts                # Centralized fetch wrapper (base URL, credentials, error handling)
│   │   │   ├── types.ts              # TypeScript interfaces for API responses
│   │   │   ├── stores/
│   │   │   │   ├── auth.ts           # Auth state store (user info, logged-in flag)
│   │   │   │   └── toast.ts          # Toast notification store
│   │   │   └── components/
│   │   │       ├── MarketCard.svelte  # Market list item card
│   │   │       ├── TradeWidget.svelte # Buy/sell interface
│   │   │       ├── PriceChart.svelte  # Price history chart
│   │   │       └── ...
│   │   └── app.d.ts                   # SvelteKit type declarations
│   └── static/
│       ├── favicon.ico
│       └── fonts/                     # Self-hosted fonts if used
├── docker-compose.yml
├── Caddyfile                          # Reverse proxy config
├── plan.md
├── README.md
└── LICENSE
```

---

## 8. Data Model (overview)

### User

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | Auto-increment local ID |
| `scid_sub` | TEXT UNIQUE NOT NULL | OIDC `sub` claim from scid.my — stable unique identifier |
| `display_name` | TEXT NOT NULL | From `preferred_username` claim (RSI handle) |
| `email` | TEXT | From `email` claim, nullable |
| `balance` | INTEGER NOT NULL DEFAULT 1000 | ScollyBucks™ balance (stored as integer — no float currency) |
| `is_moderator` | BOOLEAN NOT NULL DEFAULT FALSE | |
| `is_admin` | BOOLEAN NOT NULL DEFAULT FALSE | |
| `created_at` | DATETIME NOT NULL | First login timestamp |
| `last_login_at` | DATETIME NOT NULL | Updated on each OIDC login |

**Note:** No `password_hash` column. Auth is fully delegated to SCID.

### Market

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `title` | TEXT NOT NULL | The prediction question |
| `description` | TEXT NOT NULL | Markdown body |
| `category` | TEXT NOT NULL | Enum: bug_fixes, feature_delivery, patch_timing, cig_drama, community_events, meta |
| `resolution_criteria` | TEXT NOT NULL | Clear description of how YES/NO is determined |
| `resolution_deadline` | DATETIME NOT NULL | When the market must be resolved |
| `status` | TEXT NOT NULL DEFAULT 'pending_review' | Enum: pending_review, active, resolved, cancelled |
| `resolution` | TEXT | Enum: yes, no, cancelled — NULL until resolved |
| `created_by` | INTEGER FK → User | |
| `resolved_by` | INTEGER FK → User | Nullable — set when resolved |
| `created_at` | DATETIME NOT NULL | |
| `resolved_at` | DATETIME | Nullable |
| `liquidity_param` | REAL NOT NULL DEFAULT 100.0 | LMSR `b` parameter |
| `yes_shares` | REAL NOT NULL DEFAULT 0.0 | Outstanding YES shares in pool |
| `no_shares` | REAL NOT NULL DEFAULT 0.0 | Outstanding NO shares in pool |

### Trade

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `user_id` | INTEGER FK → User | |
| `market_id` | INTEGER FK → Market | |
| `side` | TEXT NOT NULL | Enum: yes, no |
| `action` | TEXT NOT NULL | Enum: buy, sell |
| `shares` | REAL NOT NULL | Number of shares traded |
| `cost` | INTEGER NOT NULL | ScollyBucks™ paid/received (integer) |
| `price_at_trade` | REAL NOT NULL | Probability at time of trade (0.0–1.0) |
| `created_at` | DATETIME NOT NULL | |

### Position

| Column | Type | Notes |
|---|---|---|
| `user_id` | INTEGER FK → User | Composite PK |
| `market_id` | INTEGER FK → Market | Composite PK |
| `yes_shares` | REAL NOT NULL DEFAULT 0.0 | |
| `no_shares` | REAL NOT NULL DEFAULT 0.0 | |

### ModerationAction

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `market_id` | INTEGER FK → Market | |
| `moderator_id` | INTEGER FK → User | |
| `action` | TEXT NOT NULL | Enum: approve, reject, edit, cancel |
| `reason` | TEXT | |
| `created_at` | DATETIME NOT NULL | |

### Report

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `reporter_id` | INTEGER FK → User | |
| `market_id` | INTEGER FK → Market | |
| `reason` | TEXT NOT NULL | |
| `status` | TEXT NOT NULL DEFAULT 'pending' | Enum: pending, reviewed, dismissed |
| `created_at` | DATETIME NOT NULL | |

### AutoFilterRule

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `rule_type` | TEXT NOT NULL | Enum: keyword, regex, rate_limit, min_length |
| `value` | TEXT NOT NULL | The keyword, regex pattern, number, etc. |
| `enabled` | BOOLEAN NOT NULL DEFAULT TRUE | |
| `created_at` | DATETIME NOT NULL | |

---

## 9. AMM Design — LMSR

We'll use a simple [Logarithmic Market Scoring Rule](https://en.wikipedia.org/wiki/Scoring_rule#Logarithmic_scoring_rule) (Hanson's market maker):

- The market maker holds a pool of YES and NO shares.
- **Cost function:** $C(q) = b \cdot \ln(e^{q_{yes}/b} + e^{q_{no}/b})$
  where $q_{yes}$ and $q_{no}$ are outstanding shares and $b$ is the liquidity parameter.
- **Price of YES:** $p_{yes} = \frac{e^{q_{yes}/b}}{e^{q_{yes}/b} + e^{q_{no}/b}}$
- To buy $\Delta$ YES shares, the user pays $C(q_{yes} + \Delta, q_{no}) - C(q_{yes}, q_{no})$.
- Higher $b$ = more liquidity, less price impact per trade. We'll pick a sensible default (e.g. $b = 100$) and can tune later.

This is well-understood, avoids the complexity of a full order book, and guarantees that every market is always tradeable.

**Implementation note for Go:** Use `math.Exp` and `math.Log` from the stdlib `math` package. Be careful with floating-point overflow — for large share quantities relative to `b`, `exp(q/b)` can overflow `float64`. Use the log-sum-exp trick: $\ln(e^a + e^b) = \max(a,b) + \ln(1 + e^{-|a-b|})$. Implement this in `internal/service/amm.go` and write thorough table-driven tests with known input/output pairs.

---

## 10. Auto-Filter Rules (initial set)

| Rule | Type | Behavior |
|---|---|---|
| Player kill bets | keyword list | Reject markets mentioning specific player kills, bounties, ganking |
| Harassment / doxxing | keyword + regex | Reject anything referencing real names with negative intent, doxxing |
| Slurs & hate speech | keyword list | Reject common slurs and hate speech |
| Duplicate market | fuzzy title match | Flag if >80% similar to existing active market |
| Spam rate limit | rate limit | Max 5 market submissions per user per 24 hours |
| Low effort | min length | Reject if description < 20 chars or no resolution criteria |
| Real money / gambling | keyword list | Reject anything suggesting real-money exchange |

The keyword lists and thresholds should be configurable by admins without code changes (stored in DB via `AutoFilterRule`).

---

## 11. API Design

All API endpoints live under `/api/`. The Go backend serves both the API and proxies to the SvelteKit frontend (or Caddy handles routing).

### Auth Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| GET | `/auth/login` | Initiates OIDC flow — generates state + PKCE, stores in session, redirects to scid.my | No |
| GET | `/auth/callback` | OIDC callback — exchanges code for tokens, creates/updates user, sets session cookie | No |
| POST | `/auth/logout` | Clears session cookie | Yes |
| GET | `/api/me` | Returns current user info from session | Yes |

### Market Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| GET | `/api/markets` | List active markets (paginated, filterable by category, sortable) | No |
| GET | `/api/markets/:id` | Get single market with current prices | No |
| POST | `/api/markets` | Submit a new market (goes through auto-filter → mod queue) | Yes |
| GET | `/api/markets/:id/trades` | Get trade history for a market | No |
| GET | `/api/markets/:id/price-history` | Get price history data points for charting | No |

### Trading Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| POST | `/api/markets/:id/buy` | Buy YES or NO shares | Yes |
| POST | `/api/markets/:id/sell` | Sell YES or NO shares | Yes |
| GET | `/api/markets/:id/position` | Get current user's position in a market | Yes |

### User Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| GET | `/api/users/:id/profile` | Public profile — display name, portfolio, trade count | No |
| GET | `/api/users/me/positions` | All of current user's open positions | Yes |
| GET | `/api/users/me/trades` | Current user's full trade history | Yes |
| GET | `/api/leaderboard` | Top 100 users by portfolio value | No |

### Moderation Endpoints

| Method | Path | Description | Auth |
|---|---|---|---|
| GET | `/api/mod/queue` | Get pending markets for review | Mod |
| POST | `/api/mod/markets/:id/approve` | Approve a market | Mod |
| POST | `/api/mod/markets/:id/reject` | Reject a market (with reason) | Mod |
| POST | `/api/mod/markets/:id/resolve` | Resolve a market (YES/NO/CANCELLED) | Mod |
| GET | `/api/mod/reports` | Get pending reports | Mod |
| POST | `/api/mod/reports/:id/review` | Mark a report as reviewed/dismissed | Mod |
| POST | `/api/reports` | Submit a report against a market | Yes |

### Response Format

All API responses use a consistent JSON envelope:

```json
// Success
{
  "data": { ... }
}

// Error
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "You need 42 more ScollyBucks™ for this trade."
  }
}
```

Use appropriate HTTP status codes: `200` OK, `201` Created, `400` Bad Request, `401` Unauthorized, `403` Forbidden, `404` Not Found, `422` Unprocessable Entity, `429` Too Many Requests, `500` Internal Server Error.

---

## 12. Implementation Phases

### Phase 1 — Foundation ✅ Complete
- [x] Set up monorepo structure (backend + frontend directories)
- [x] Backend: Go module, chi router, basic middleware (logging, CORS, recovery)
- [x] SQLite database setup with goose migrations — initial schema (users table)
- [x] OIDC auth flow: `/auth/login` → scid.my → `/auth/callback` → session cookie → `/api/me`
- [x] sqlc setup: first query file and code generation for users
- [x] Frontend: SvelteKit 2 + Svelte 5 + Tailwind v4 + Skeleton v4, luxury gold theme, layout with nav bar
- [x] "Login with SCID" button → auth flow → display logged-in user
- [x] Taskfile.yml with `task dev` (runs air backend + vite frontend concurrently)
- [x] `.air.toml` for Go hot-reload in development

### Phase 2 — Core Market Flow ✅ Complete
- [x] Database migrations for markets, trades, positions tables
- [x] sqlc queries for market CRUD
- [x] Market CRUD API (create, list, get, update status)
- [x] AMM implementation in `internal/service/amm.go` (LMSR cost function, buy/sell logic)
- [x] Trading API (`POST /api/trades` — unified buy/sell endpoint)
- [x] Market list page with category filtering and pagination
- [x] Market detail page with YES/NO price display
- [x] Buy/sell UI widget
- [x] Market submission form (`/markets/new`)
- [x] Leaderboard page and profile (me) page
- [x] Playwright e2e test setup with mocked API (`tests/helpers/mock-api.ts`)
- [x] `tests/home.spec.ts` and `tests/markets.spec.ts`

### Phase 3 — Moderation & Auto-filter ✅ Complete
- [x] Mod queue API (`GET /api/mod/markets`, approve/reject/resolve)
- [x] Mod dashboard page (`/mod`) — approve, reject, resolve, deny resolution, review reports
- [x] Resolution request flow — user requests resolution, mod approves or denies
- [x] Auto-filter service — keyword/regex matching against `autofilter_rules` table
- [x] Auto-filter — fuzzy duplicate title detection (trigram-based Jaccard similarity, >0.6 threshold)
- [x] Auto-filter — min-length validation enforced from DB `min_length` rules
- [x] Report system (submit + mod review + dismiss)
- [x] Rate limiting on auth, market creation, trading, and report submission
- [x] Integration tests for full market lifecycle, AMM math, mod queue, reports, payout

### Phase 4 — Economy & Social ✅ Complete
- [x] Weekly credit payout (hourly goroutine, idempotent via weekly_payout_log)
- [x] Market resolution flow (mod resolves YES/NO/CANCELLED, atomic payout transaction)
- [x] Price history chart on market detail page (inline SVG sparkline)
- [x] Sell shares UI on market detail page
- [x] Badge system (first_blood, market_maven, bug_prophet, eternal_optimist, doomsayer)
- [x] Profile badges display on `/me` page

### Phase 5 — Polish & Deploy ✅ Complete
- [x] Theme: luxury gold dark mode, Outfit font, luxury typography throughout
- [x] Satirical copy and badge system
- [x] Caddyfile for reverse proxy + HTTPS (`Caddyfile` at repo root)
- [x] Docker Compose production setup (`docker-compose.yml`, `backend/Dockerfile`, `frontend/Dockerfile`, root `.env.example`)
- [x] GitHub Actions CI (`golangci-lint`, `go test -race`, `svelte-check`, `npm run build`, Playwright) — `.github/workflows/ci.yml`
- [ ] Deploy to a small VPS

---

## 13. Security Checklist

- [ ] **No local passwords.** Auth is fully OIDC via scid.my.
- [ ] **OIDC state parameter** validated on every callback to prevent CSRF.
- [ ] **PKCE** (code_challenge / code_verifier) used in auth flow.
- [ ] **id_token signature validated** via JWKS from scid.my's `.well-known` endpoint.
- [ ] **id_token claims validated:** `iss`, `aud`, `exp`.
- [ ] **Client secret** stored in environment variables only, never in code or version control.
- [ ] **Session cookies:** `httpOnly`, `Secure`, `SameSite=Lax`.
- [ ] **All SQL parameterized.** sqlc enforces this. No raw string concatenation in queries.
- [ ] **Input validation** at handler boundary for all user-submitted data (market title, description, trade amounts).
- [ ] **Rate limiting** on `/auth/callback`, `POST /api/markets`, and trading endpoints.
- [ ] **CORS** configured to only allow the frontend origin.
- [ ] **Markdown sanitization** on frontend before rendering user-generated content (DOMPurify).
- [ ] **No sensitive data in logs.** Never log tokens, secrets, or full request bodies containing user data.
- [ ] **Integer arithmetic for currency.** ScollyBucks™ balances and trade costs are integers — no floating-point currency bugs.
- [ ] **Transaction isolation** for trade execution — buy/sell operations wrapped in SQLite transactions with proper locking.
- [ ] **HTTPS only** in production via Caddy auto-TLS.

---

## 14. Open Questions / Future Ideas

- **Comments on markets?** Probably yes — lightweight comment thread per market. Adds engagement.
- **Market creator reward?** Small credit bonus for creating markets that get high volume. Incentivizes good questions.
- **API for bots?** Could be funny. Let people write trading bots on a satirical market. Low priority.
- **Star Citizen patch scraping?** Auto-resolve bug fix markets by scraping patch notes or the Issue Council. Ambitious but fun.
- **Categories expansion?** Add categories for specific ship manufacturers, gameplay loops, etc.
- **Expiring markets auto-resolution?** If no mod resolves within X days of deadline, auto-cancel and refund.
- **RSI handle verification display?** Show a "verified" badge on users whose SCID account has a verified RSI handle, using SCID claims.

---

*"The chances of Star Citizen releasing Squadron 42 on time are roughly the same as the chances of this project being maintained long-term."*
