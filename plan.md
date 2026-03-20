# ScollyMarket — Plan

> A satirical prediction market for Star Citizen bug fixing and general events.
> No real money. Just vibes, credits, and the collective delusion that 3.x will be stable.

---

## 1. Concept

ScollyMarket is a tongue-in-cheek prediction market (à la Polymarket/Manifold) where players bet fake credits on whether Star Citizen bugs will be fixed, features will ship, and other community events will happen. The tone is irreverent and self-aware — this is a joke about the state of the game, not a serious forecasting tool.

### How Prediction Markets Work (simplified)

- A **market** is a yes/no question: *"Will the Reclaimer elevator bug be fixed by 4.0?"*
- Users buy **YES** or **NO** shares. Share prices range from **1–99 credits** and reflect the crowd's estimated probability.
- When the market **resolves** (the question is answered), YES shares pay out 100 credits each if the answer was yes, 0 if no — and vice versa for NO shares.
- The price you pay is your cost basis. Buy YES at 30, outcome is YES → you profit 70. Outcome is NO → you lose 30.
- A simple **automated market maker (AMM)** sets initial prices and provides liquidity so users can always trade. We'll use a straightforward constant-product or logarithmic market scoring rule (LMSR) — nothing fancy.

### Core Loop

1. User browses open markets or submits a new one.
2. User buys YES or NO shares with their credit balance.
3. Market resolves at deadline (or manually by moderator).
4. Payouts are distributed. Leaderboard updates.

---

## 2. Scope & Constraints

| Constraint | Decision |
|---|---|
| Real money | **None.** Play credits only. |
| Scale | ~100–300 users max. Single server, SQLite is fine. |
| Lifespan | Short-lived joke project. Minimal ops. |
| Auth | Simple username/password registration. No OAuth, no email verification. |
| Mobile | Responsive web is enough. No native app. |
| Legal | No real gambling, no real money, no crypto. Satirical fair use. |

---

## 3. Features

### 3.1 Credits & Economy

- **Starting balance:** Every new account gets **1,000 ScollyBucks™**.
- **Weekly payout:** All active users receive **200 ScollyBucks™** every Monday. Keeps the economy alive and lets losing players recover.
- **No negative balances.** You can't bet what you don't have.
- **Leaderboard** ranked by total portfolio value (credits + value of held shares).

### 3.2 Markets

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

### 3.3 Moderation System

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

### 3.4 Trading

- **AMM-backed:** Every market is initialized with a liquidity pool. We use a simple LMSR (Logarithmic Market Scoring Rule) with a configurable liquidity parameter `b`. This determines how sensitive prices are to trades — lower `b` = more volatile, higher `b` = more stable prices.
- **Buy/Sell:** Users can buy or sell YES/NO shares at any time while the market is active.
- **Price display:** Current price shown as a percentage (e.g., YES at 72 means the crowd thinks there's a 72% chance).
- **Order history:** Each user can see their own trade history per market.
- **No limit orders, no order book.** AMM only. Keeps it simple.

### 3.5 Resolution

- At or after the deadline, a moderator resolves the market as **YES**, **NO**, or **CANCELLED**.
- **YES/NO:** Winning shares pay 100 credits. Losing shares pay 0.
- **CANCELLED:** All shares refunded at purchase price (e.g., if the question becomes unanswerable).
- Resolved markets are archived and visible but no longer tradeable.
- There should be a grace period / reminder system so markets don't languish unresolved.

### 3.6 User Profiles & Leaderboard

- **Profile:** Username, join date, current balance, portfolio (open positions), trade history, submitted markets.
- **Leaderboard:** Ranked by total portfolio value. Updated in near-real-time.
- **Badges / titles** (optional fun): e.g. "Bug Prophet", "Eternal Optimist", "CIG Apologist". Based on trading patterns, win streaks, etc.

### 3.7 UI Pages

| Page | Purpose |
|---|---|
| **Home / Market List** | Browse active markets, filter by category, sort by volume/deadline/newest |
| **Market Detail** | See question, description, price chart, buy/sell widget, comments |
| **Submit Market** | Form to propose a new market |
| **Mod Dashboard** | Review pending markets, handle reports (mod-only) |
| **Leaderboard** | Top traders ranked by portfolio value |
| **Profile** | User's portfolio, trade history, submitted markets |
| **Login / Register** | Simple auth forms |

---

## 4. Tech Stack

### Backend — Python

| Tool | Purpose |
|---|---|
| **Python 3.12+** | Language |
| **uv** | Project & package management (replaces pip/venv/poetry) |
| **FastAPI** | Web framework — async, fast, auto-generated OpenAPI docs |
| **SQLite** | Database — single file, zero config, perfect for this scale |
| **SQLModel** | ORM — SQLAlchemy + Pydantic in one, pairs well with FastAPI |
| **Alembic** | Database migrations |
| **bcrypt** (via passlib) | Password hashing |
| **python-jose** | JWT token auth |
| **APScheduler** | Scheduled tasks (weekly payouts, resolution reminders) |
| **Ruff** | Linting & formatting |
| **pytest** | Testing |

### Frontend — Svelte + Skeleton

| Tool | Purpose |
|---|---|
| **SvelteKit** | Framework — SSR, routing, file-based routes |
| **Skeleton UI** | Component library for Svelte — polished, themeable, accessible |
| **TypeScript** | Type safety |
| **Vite** | Bundler (built into SvelteKit) |
| **Chart.js** or **Layerchart** | Price history charts for markets |
| **date-fns** | Date formatting/manipulation |
| **ESLint + Prettier** | Linting & formatting |

### Dev & Deployment

| Tool | Purpose |
|---|---|
| **Docker Compose** | Single command local dev & deployment |
| **Caddy** or **nginx** | Reverse proxy + auto HTTPS (if deployed) |
| **GitHub Actions** | CI — lint, test, build |
| **SQLite backup cron** | Simple `cp` of the db file. Enterprise-grade disaster recovery™ |

---

## 5. Project Structure

```
scollymarket/
├── backend/
│   ├── pyproject.toml          # uv project config
│   ├── alembic/                # db migrations
│   ├── src/
│   │   └── scollymarket/
│   │       ├── __init__.py
│   │       ├── main.py         # FastAPI app entry
│   │       ├── config.py       # settings & env vars
│   │       ├── database.py     # SQLite engine & session
│   │       ├── models/         # SQLModel models
│   │       │   ├── user.py
│   │       │   ├── market.py
│   │       │   ├── trade.py
│   │       │   └── moderation.py
│   │       ├── routers/        # API route modules
│   │       │   ├── auth.py
│   │       │   ├── markets.py
│   │       │   ├── trading.py
│   │       │   ├── users.py
│   │       │   └── moderation.py
│   │       ├── services/       # business logic
│   │       │   ├── amm.py      # market maker math
│   │       │   ├── credits.py  # balance & payout logic
│   │       │   ├── autofilter.py
│   │       │   └── resolution.py
│   │       └── schemas/        # request/response schemas (if separate from models)
│   └── tests/
├── frontend/
│   ├── package.json
│   ├── svelte.config.js
│   ├── src/
│   │   ├── routes/             # SvelteKit file-based routing
│   │   │   ├── +page.svelte    # home / market list
│   │   │   ├── +layout.svelte  # global layout, nav
│   │   │   ├── market/
│   │   │   │   ├── [id]/+page.svelte
│   │   │   │   └── submit/+page.svelte
│   │   │   ├── leaderboard/+page.svelte
│   │   │   ├── profile/+page.svelte
│   │   │   ├── mod/+page.svelte
│   │   │   ├── login/+page.svelte
│   │   │   └── register/+page.svelte
│   │   ├── lib/
│   │   │   ├── api.ts          # fetch wrapper for backend
│   │   │   ├── stores/         # Svelte stores (auth, user, etc.)
│   │   │   └── components/     # reusable UI components
│   │   └── app.html
│   └── static/
│       └── favicon.ico
├── docker-compose.yml
├── plan.md
├── README.md
└── LICENSE
```

---

## 6. Data Model (overview)

### User
- `id`, `username`, `password_hash`, `balance`, `is_moderator`, `is_admin`, `created_at`

### Market
- `id`, `title`, `description`, `category`, `resolution_criteria`, `resolution_deadline`
- `status` (pending_review | active | resolved | cancelled)
- `resolution` (yes | no | cancelled — null until resolved)
- `created_by` (FK → User), `resolved_by` (FK → User), `created_at`, `resolved_at`
- AMM state: `liquidity_param`, `yes_shares`, `no_shares` (pool quantities for LMSR)

### Trade
- `id`, `user_id` (FK), `market_id` (FK)
- `side` (yes | no), `action` (buy | sell)
- `shares`, `cost`, `price_at_trade`
- `created_at`

### Position (user's current holding in a market)
- `user_id`, `market_id`, `yes_shares`, `no_shares`, `average_cost_yes`, `average_cost_no`

### ModerationAction
- `id`, `market_id` (FK), `moderator_id` (FK)
- `action` (approve | reject | edit | cancel)
- `reason`, `created_at`

### Report
- `id`, `reporter_id` (FK), `market_id` (FK)
- `reason`, `status` (pending | reviewed | dismissed)
- `created_at`

### AutoFilterRule
- `id`, `rule_type` (keyword | regex | rate_limit | min_length)
- `value`, `enabled`, `created_at`

---

## 7. AMM Design — LMSR

We'll use a simple [Logarithmic Market Scoring Rule](https://en.wikipedia.org/wiki/Scoring_rule#Logarithmic_scoring_rule) (Hanson's market maker):

- The market maker holds a pool of YES and NO shares.
- **Cost function:** $C(q) = b \cdot \ln(e^{q_{yes}/b} + e^{q_{no}/b})$
  where $q_{yes}$ and $q_{no}$ are outstanding shares and $b$ is the liquidity parameter.
- **Price of YES:** $p_{yes} = \frac{e^{q_{yes}/b}}{e^{q_{yes}/b} + e^{q_{no}/b}}$
- To buy $\Delta$ YES shares, the user pays $C(q_{yes} + \Delta, q_{no}) - C(q_{yes}, q_{no})$.
- Higher $b$ = more liquidity, less price impact per trade. We'll pick a sensible default (e.g. $b = 100$) and can tune later.

This is well-understood, avoids the complexity of a full order book, and guarantees that every market is always tradeable.

---

## 8. Auto-Filter Rules (initial set)

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

## 9. Implementation Phases

### Phase 1 — Foundation
- [ ] Set up monorepo structure (backend + frontend)
- [ ] Backend: FastAPI app skeleton with uv, SQLite, SQLModel
- [ ] Database models & migrations
- [ ] Auth: register, login, JWT tokens
- [ ] Frontend: SvelteKit + Skeleton scaffolding, layout, basic navigation

### Phase 2 — Core Market Flow
- [ ] Market CRUD API (create, list, get, update status)
- [ ] AMM implementation (LMSR cost function, buy/sell endpoints)
- [ ] Trading API (buy/sell shares, position tracking)
- [ ] Market list page, market detail page with price display
- [ ] Buy/sell UI widget
- [ ] Market submission form

### Phase 3 — Moderation & Auto-filter
- [ ] Auto-filter service (keyword matching, duplicate detection, rate limiting)
- [ ] Mod queue API and dashboard page
- [ ] Market approval/rejection flow
- [ ] Report system (submit + review)

### Phase 4 — Economy & Social
- [ ] Weekly credit payout (scheduled job)
- [ ] Leaderboard API and page
- [ ] User profile page (portfolio, history)
- [ ] Market resolution flow (mod resolves, payouts distributed)
- [ ] Price history chart on market detail

### Phase 5 — Polish & Deploy
- [ ] Theming: dark space theme, Star Citizen flavor
- [ ] Satirical copy, Easter eggs, badge system
- [ ] Docker Compose setup
- [ ] Basic CI (lint + test)
- [ ] Deploy to a small VPS

---

## 10. Open Questions / Future Ideas

- **Comments on markets?** Probably yes — lightweight comment thread per market. Adds engagement.
- **Market creator reward?** Small credit bonus for creating markets that get high volume. Incentivizes good questions.
- **API for bots?** Could be funny. Let people write trading bots on a satirical market. Low priority.
- **Star Citizen patch scraping?** Auto-resolve bug fix markets by scraping patch notes or the Issue Council. Ambitious but fun.
- **Categories expansion?** Add categories for specific ship manufacturers, gameplay loops, etc.
- **Expiring markets auto-resolution?** If no mod resolves within X days of deadline, auto-cancel and refund.

---

*"The chances of Star Citizen releasing Squadron 42 on time are roughly the same as the chances of this project being maintained long-term."*
