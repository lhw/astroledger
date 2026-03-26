# ScolyMarket — Update Plan

This document tracks remaining work, known issues, and future ideas. See `plan.md` for the original design.

---

## Recently Completed

- **Payout balance refresh**: The frontend now refreshes the user's balance when visiting a resolved market, so the updated bUEC shows immediately instead of waiting for a page reload.
- **Resolved markets visible**: Markets are no longer hidden after resolution. The market list includes a "Resolved" filter, and resolved markets show their result badge. Cancelled markets are also now filterable.
- **Buy by bUEC budget**: The trade widget now has a "By Shares / By Budget" toggle. In budget mode, users enter how much bUEC they want to spend, and the UI shows the estimated share count using the LMSR `maxAffordable` function.
- **Market resolution types**: Added `binary` (yes/no), `date` (resolves YES if an event occurs before a target date), and `numeric` (resolves YES if a value reaches a threshold). The creation form has a Market Type selector, and the market detail page shows the type context.
- **Market stats on detail page**: The sidebar now shows total volume (bUEC), unique trader count, and total trade count.
- **Probability bar on market list**: Each active market card now shows the LMSR-derived YES probability as a percentage label and a green-over-red bar, similar to Polymarket.
- **Comments / Discussion**: Users can post and read comments on each market page. Comments support Markdown (rendered with DOMPurify-sanitized HTML). Mods can hard-delete comments. Comment count shown on market list cards.
- **Automated abuse detection (OpenAI Moderation API)**: Comment submissions are scored by the OpenAI Moderation API (`omni-moderation-latest`). Comments that exceed per-category score thresholds (tuned for a gaming community) or are flagged by OpenAI are shadow-hidden — invisible to other users but still visible to the author with an `⚠ Under review` notice. Configured via `OPENAI_API_KEY` env var; graceful no-op when key is absent. *(Replaced Perspective API, which is being shut down.)*
- **Market Creator Recognition**: A `Creator` badge is shown next to the submitter's name on the market detail page. When a mod approves a market, the creator automatically receives a 50 bUEC bonus.
- **Expiring Pending Markets**: A background goroutine (runs hourly at startup) auto-cancels markets stuck in `pending_review` for more than 14 days, and moves active markets past their resolution deadline to `deadline_passed` status. New DB indexes added for both queries.
- **Better Resolution Flow**: When resolving a market in the mod queue, mods can now attach an optional evidence link (issue tracker URL, patch notes, etc.) that is stored in the DB and displayed on the resolved market's detail page. The resolver's display name and resolution date are now shown on the detail page. Evidence link input added to the mod queue resolution form.

---

## High Priority

### ~~Comments / Discussion~~ ✅ Done
- ~~Allow users to post short comments on each market page.~~
- ~~Show comment count on market list cards.~~
- ~~Comments should support markdown (sanitized with DOMPurify).~~
- ~~Mods can delete comments.~~

### ~~Market Creator Recognition~~ ✅ Done
- ~~Show a "Creator" badge next to the submitter's name on the market detail page.~~
- ~~Optionally award a small bUEC bonus (e.g., 50 bUEC) when a creator's market goes live.~~

### ~~Expiring Pending Markets~~ ✅ Done
- ~~Markets that stay in `pending_review` for longer than a configurable threshold (e.g., 14 days) should auto-cancel with a nightly background job.~~
- ~~Similarly, active markets past their deadline without a resolution request should move to a `deadline_passed` status so mods are prompted.~~

### ~~Better Resolution Flow~~ ✅ Done
- ~~When resolving, mods should be able to attach an evidence link (issue tracker URL, patch notes link, etc.) that is stored and displayed on the market detail page.~~
- ~~Show who resolved the market and when (the `resolved_by` / `resolved_at` columns exist already but aren't surfaced on the detail page yet beyond a label).~~

---

## Medium Priority

### True Multi-Outcome Markets
- The current `date` and `numeric` resolution types are still binary (YES/NO against a threshold). Consider a true multi-choice market structure for questions like "Which patch will Big Ben finally get fixed in? 4.1 / 4.2 / 4.3 / Never".
- Requires a new DB schema (`market_outcomes` table, per-outcome share pools), a different AMM formulation (multi-dimensional LMSR or simple equal-liquidity pools), and a redesigned trade UI.

### Fractional Share Display / UX
- Budget mode calculates `budgetShares` as a float (e.g., 3.72 shares). The actual trade submits `Math.floor(budgetShares)` whole shares. The UI should make it clear that only whole shares are purchased and show the exact spend.
- Consider allowing the backend to support fractional shares if ever needed.

### Price History Chart Improvements
- The current chart shows only the last N trades. Add time-axis labels and a resolution marker line.
- Allow switching between linear and log scale.
- Show NO price as a secondary line (they are complementary, but visual context is useful).

### User Portfolio Page Improvements
- Show unrealized P&L per position (current share value at market price vs. cost basis — requires storing average cost, not currently tracked).
- Show resolved markets with payout received.
- Badges section already exists; add a description tooltip for each badge.

### ~~RSI Handle Verification~~ ✅ Done
- Stores `rsi_handle`, `rsi_verified_at`, `rsi_enlisted`, `rsi_citizen_record` from SCID OIDC claims into the `users` table on every login (migration 009).
- The `verified` OIDC group is mirrored as `is_rsi_verified` in the DB and surfaced in `/api/me`.
- Profile page shows an RSI Identity card with the handle (linked to RSI profile), citizen record number, enlistment date, and a "RSI Verified" badge for confirmed accounts.
- `picture` claim from the OIDC token is stored as `avatar_url` when present; a deterministic colour-initial avatar is shown as fallback.
- `UserAvatar` component displays a circular photo or initials badge wherever user names appear; navbar now shows the avatar next to the display name.

### Admin / Mod Panel Improvements
- Add a bulk-action UI for mod queue (approve/reject multiple at once).
- Add a search + filter to the mod queue by category, reporter, or keyword.
- Show auto-filter rule matches on each pending market card.
- Log all mod actions to an audit table (`mod_audit` with action type, target, mod user, timestamp, note).

---

## Low Priority / Ideas

### Notifications
- Email or in-app notification when a market the user holds shares in is resolved.
- Notification when a user's submitted market goes live or is rejected.
- Could use a simple `notifications` table + polling, no push infrastructure needed for ~200 users.

### Weekly Credit Automation
- Currently handled by a cron/background job. Add a simple admin endpoint to manually trigger the weekly payout (useful for testing and edge cases around cron timing).

### Star Citizen Patch Scraper
- A background job that scrapes the SC roadmap or patch notes RSS and attempts to match market titles to recent patches.
- Auto-creates a resolution request when a relevant patch ships, reducing mod workload.
- Risk: false positives. Should require mod confirmation before resolving.

### Bot Trading API
- A simple API token system (scoped to read/trade) that allows bots to trade on markets.
- Could be used to test AMM behavior, run simulated market scenarios, or allow community bots.
- Must be rate-limited and not grant admin/mod access.

### Mobile UX
- The trade widget sidebar collapses awkwardly on small screens. Consider a bottom-sheet / drawer for the trade UI on mobile.
- Ensure the probability bar and price pill on list cards are readable on small screens.

### Pagination Improvements
- The current limit/offset pagination loads full pages. Add infinite scroll or a "Load more" button as an alternative.
- Market list should show total result count even when filtered.

### Performance / Caching
- Market list server load function currently hits the DB every SSR request. Add short-lived (e.g., 5 s) in-memory caching for the list query.
- Price history endpoint could be cached per market since trades are append-only.

### Dark Mode
- Skeleton UI supports dark mode via the `dark` class. Wire it to the OS preference or a user toggle.
- The custom gold theme needs dark-mode variant colors defined in `src/theme.ts`.

---

## Known Issues / Bugs

- **Fractional budget shares**: Budget mode shows a float share estimate but the actual purchase is floored to an integer. The button label says "Spend X bUEC" but the actual cost may differ slightly. The receipt after trading should show the real cost.
- **`cancelled` status filter on market list**: The filter exists in the UI but the backend `ListMarkets` query may need to explicitly handle `cancelled` status if it differs from how inactive markets are stored. Verify with a cancelled market in the DB.
- **Resolution type display on resolved markets**: The "Resolved" card on the detail page shows YES/NO but doesn't yet contextualize it for `date` or `numeric` markets (e.g., "The event DID occur before [date]" vs just "YES").
- **Price history on resolved markets**: The chart shows trades up to resolution; it would be cleaner to mark the resolution point on the chart with a vertical line.
- **`deadline_passed` not a filter option in market list**: The new `deadline_passed` status exists in the DB and TypeScript types but isn't yet a selectable filter in the market list UI. Mods need a way to surface these markets.
- **`deadline_passed` not shown in mod queue**: The mod queue currently shows `pending_review` and `resolution_requested` markets but not `deadline_passed`. Mods should be able to see and action expired markets.
