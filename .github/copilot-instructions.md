# ScolyMarket — Copilot Instructions

## Project Overview

ScolyMarket is a satirical prediction market web app for betting fake credits on Star Citizen bug fixes and general events. No real money is involved. The tone is humorous and self-aware.

See `plan.md` for the full design document.

## Architecture

- **Backend:** Python 3.12+ with FastAPI, managed by `uv`. SQLite database via SQLModel. JWT auth.
- **Frontend:** SvelteKit with Skeleton UI, TypeScript, Vite.
- **Monorepo:** `backend/` and `frontend/` directories at the repo root.

## Code Conventions

### Python (backend)

- Use `uv` for all dependency management. Never use `pip install` directly.
- Follow PEP 8. Use `ruff` for linting and formatting.
- Type-annotate all function signatures.
- Use `async def` for all FastAPI route handlers.
- Models use SQLModel (which combines SQLAlchemy and Pydantic).
- Keep business logic in `services/`, route handlers in `routers/`, and data models in `models/`.
- Use dependency injection (`Depends()`) for database sessions, auth, and services.
- Passwords must be hashed with bcrypt. Never store or log plaintext passwords.
- All API responses should use Pydantic response models — never return raw ORM objects.
- Tests go in `backend/tests/` using pytest.

### TypeScript / Svelte (frontend)

- Use TypeScript strictly — avoid `any`.
- Use SvelteKit file-based routing under `src/routes/`.
- Use Skeleton UI components wherever possible instead of custom CSS.
- API calls go through a centralized fetch wrapper in `src/lib/api.ts`.
- Shared state uses Svelte stores in `src/lib/stores/`.
- Format with Prettier, lint with ESLint.

### General

- Prefer simple, readable code over clever abstractions.
- This is a joke project for ~200 users. Don't over-engineer.
- SQLite is the database — no need for PostgreSQL, Redis, or message queues.
- No real money or crypto is involved. Ever.

## Key Domain Concepts

- **Market:** A yes/no question that users bet on. Has a deadline and resolution criteria.
- **AMM (LMSR):** Automated Market Maker using Logarithmic Market Scoring Rule. Provides liquidity so users can always buy/sell.
- **ScollyBucks™:** The play currency. Users start with 1,000 and get 200 weekly.
- **Shares:** YES or NO shares in a market. Prices range 1–99. Winning shares pay 100 at resolution.
- **Moderation:** Markets go through auto-filter → mod queue → active. Mods resolve markets.
- **Auto-filter:** Keyword/regex rules that auto-reject markets about banned topics (player kills, harassment, etc.).

## Security Notes

- Hash passwords with bcrypt.
- Use JWT tokens with expiration for auth. Store tokens in httpOnly cookies on the frontend.
- Validate and sanitize all user input server-side.
- Use parameterized queries (SQLModel handles this, but be careful with any raw SQL).
- Rate-limit auth endpoints and market submission.
- Sanitize markdown rendering on the frontend to prevent XSS.
