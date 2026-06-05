<svelte:head>
	<title>Bot API Docs — AstroLedger</title>
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-12">
	<p class="text-xs font-bold uppercase tracking-[0.2em] text-primary-600 mb-2">Reference</p>
	<h1 class="text-3xl font-bold text-surface-900 tracking-tight mb-2">Bot API</h1>
	<div class="w-10 h-px bg-primary-400 mb-6"></div>
	<p class="text-surface-600 mb-10 max-w-xl">
		AstroLedger exposes a small HTTP API so you can read market data, place trades,
		and manage markets programmatically. Bot tokens work on all existing endpoints —
		generate one on your <a href="/me" class="text-primary-600 hover:underline">profile page</a>.
	</p>

	<!-- Authentication -->
	<section class="mb-12">
		<h2 class="doc-heading">Authentication</h2>
		<p class="doc-body mb-4">
			Pass your token in the <code>Authorization</code> header on every request:
		</p>
		<pre class="doc-code">Authorization: Bearer smt_&lt;your token&gt;</pre>
		<p class="doc-body mt-4">
			Tokens have optional scopes that you set when you create them:
		</p>
		<div class="overflow-x-auto mt-3">
			<table class="doc-table">
				<thead>
					<tr><th>Scope</th><th>Grants</th></tr>
				</thead>
				<tbody>
					<tr><td><code>read</code></td><td>Read account info and market data. Enabled by default.</td></tr>
					<tr><td><code>trade</code></td><td>Place buy/sell orders via <code>POST /api/trades</code>.</td></tr>
					<tr><td><code>create_markets</code></td><td>Create new markets via <code>POST /api/markets</code>. Mod/admin only.</td></tr>
				</tbody>
			</table>
		</div>
		<p class="doc-body mt-4">
			Bot tokens work on the same endpoints as the web app — session cookies and
			Bearer tokens are interchangeable. Session auth has full access; bot tokens
			are restricted to their granted scopes.
		</p>
	</section>

	<!-- Rate limits -->
	<section class="mb-12">
		<h2 class="doc-heading">Rate limits</h2>
		<p class="doc-body">
			Bot endpoints share the same IP-based rate limits as the web app:
		</p>
		<div class="overflow-x-auto mt-3">
			<table class="doc-table">
				<thead>
					<tr><th>Endpoint</th><th>Limit</th></tr>
				</thead>
				<tbody>
					<tr><td><code>GET /api/me</code></td><td>60 req / min</td></tr>
					<tr><td><code>POST /api/trades</code></td><td>30 req / min</td></tr>
					<tr><td><code>POST /api/markets</code></td><td>5 req / min</td></tr>
				</tbody>
			</table>
		</div>
	</section>

	<!-- Error format -->
	<section class="mb-12">
		<h2 class="doc-heading">Errors</h2>
		<p class="doc-body mb-4">
			All errors are JSON with an <code>error</code> string field and a standard HTTP status code.
		</p>
		<pre class="doc-code">{`{ "error": "token missing trade scope" }`}</pre>
	</section>

	<!-- GET /api/me -->
	<section class="mb-10">
		<div class="flex items-center gap-3 mb-3">
			<span class="method-badge method-get">GET</span>
			<h3 class="font-mono text-sm font-semibold text-surface-800">/api/me</h3>
			<span class="scope-badge">read</span>
		</div>
		<p class="doc-body mb-4">Returns your account balance and display name.</p>
		<p class="text-xs font-semibold uppercase tracking-wider text-surface-500 mb-2">Response 200</p>
		<pre class="doc-code">{`{
  "id": 42,
  "display_name": "CitizenHawk",
  "balance": 3750
}`}</pre>
	</section>

	<!-- POST /api/trades -->
	<section class="mb-10">
		<div class="flex items-center gap-3 mb-3">
			<span class="method-badge method-post">POST</span>
			<h3 class="font-mono text-sm font-semibold text-surface-800">/api/trades</h3>
			<span class="scope-badge scope-trade">trade</span>
		</div>
		<p class="doc-body mb-4">
			Executes a buy or sell order. Prices are determined by the LMSR automated market maker —
			you specify how many shares you want, and the AMM calculates the cost.
		</p>
		<p class="text-xs font-semibold uppercase tracking-wider text-surface-500 mb-2">Request body</p>
		<pre class="doc-code">{`{
  "market_id":  123,       // integer — ID of the market
  "outcome_id": 456,       // integer — ID of the outcome (YES / NO / other)
  "action":     "buy",     // "buy" or "sell"
  "shares":     10.0       // positive number, max 10 000 per trade
}`}</pre>
		<p class="text-xs font-semibold uppercase tracking-wider text-surface-500 mt-5 mb-2">Response 200</p>
		<pre class="doc-code">{`{
  "TradeID":      789,
  "Cost":         230,     // bUEC spent (positive = deducted, negative = refunded)
  "Shares":       10.0,
  "PriceAtTrade": 23,      // cents-per-share price at execution
  "NewBalance":   3520     // your balance after the trade
}`}</pre>
		<div class="mt-5 p-4 rounded-lg doc-tip">
			<strong>Tip:</strong> fetch <code>GET /api/markets/&#123;id&#125;</code> first to read the current
			<code>outcomes[].id</code> and <code>outcomes[].price</code> before submitting a trade.
		</div>
	</section>

	<!-- POST /api/markets -->
	<section class="mb-10">
		<div class="flex items-center gap-3 mb-3">
			<span class="method-badge method-post">POST</span>
			<h3 class="font-mono text-sm font-semibold text-surface-800">/api/markets</h3>
			<span class="scope-badge scope-create">create_markets</span>
		</div>
		<p class="doc-body mb-4">
			Create a new prediction market. Only moderators and admins can create markets
			via the bot API. Markets are created in <code>pending_review</code> status.
		</p>
		<p class="text-xs font-semibold uppercase tracking-wider text-surface-500 mb-2">Request body</p>
		<pre class="doc-code">{`{
  "title":               "Will X happen in patch 4.9?",
  "description":         "Details about the market...",
  "category":            "bug_fixes",       // bug_fixes | feature_delivery | patch_timing | community_events | meta
  "resolution_criteria": "YES if... NO if...",
  "deadline":            "4.9.0",           // patch version or RFC3339 date
  "outcomes":            ["YES", "NO"]      // optional, defaults to ["YES", "NO"]
}`}</pre>
		<p class="doc-body mt-4">
			<strong>Deadline formats:</strong> Patch version (e.g., <code>"4.9.0"</code>) sets deadline
			2 years out and prepends "Resolves when patch X ships." to resolution criteria.
			RFC3339 dates (e.g., <code>"2026-12-31T23:59:59Z"</code>) set an exact deadline.
		</p>
		<p class="text-xs font-semibold uppercase tracking-wider text-surface-500 mt-5 mb-2">Response 201</p>
		<pre class="doc-code">{`{
  "id":     17,
  "title":  "Will X happen in patch 4.9?",
  "status": "pending_review"
}`}</pre>
	</section>

	<!-- Public endpoints -->
	<section class="mb-12">
		<h2 class="doc-heading">Public read endpoints</h2>
		<p class="doc-body mb-5">
			These endpoints require no authentication and can be called freely from bots,
			scripts, or dashboards.
		</p>

		<div class="space-y-6">
			<div>
				<div class="flex items-center gap-3 mb-2">
					<span class="method-badge method-get">GET</span>
					<code class="text-sm font-mono text-surface-800">/api/markets</code>
				</div>
				<p class="doc-body">Paginated list of markets. Query params: <code>status</code> (active / resolved / …), <code>category</code>, <code>offset</code>.</p>
			</div>
			<div>
				<div class="flex items-center gap-3 mb-2">
					<span class="method-badge method-get">GET</span>
					<code class="text-sm font-mono text-surface-800">/api/markets/trending</code>
				</div>
				<p class="doc-body">Up to 5 active markets ranked by trade volume in the last 24 h. Cached for 30 s.</p>
			</div>
			<div>
				<div class="flex items-center gap-3 mb-2">
					<span class="method-badge method-get">GET</span>
					<code class="text-sm font-mono text-surface-800">/api/markets/&#123;id&#125;</code>
				</div>
				<p class="doc-body">Full market detail: description, resolution criteria, all outcomes with current LMSR prices, and aggregate trade stats.</p>
			</div>
			<div>
				<div class="flex items-center gap-3 mb-2">
					<span class="method-badge method-get">GET</span>
					<code class="text-sm font-mono text-surface-800">/api/markets/&#123;id&#125;/history</code>
				</div>
				<p class="doc-body">Price history for a market — array of <code>&#123; price_at_trade, outcome_label, created_at &#125;</code> rows.</p>
			</div>
			<div>
				<div class="flex items-center gap-3 mb-2">
					<span class="method-badge method-get">GET</span>
					<code class="text-sm font-mono text-surface-800">/api/leaderboard</code>
				</div>
				<p class="doc-body">Top users by balance + portfolio value. Query param: <code>limit</code> (default 25).</p>
			</div>
		</div>
	</section>

	<!-- Quick example -->
	<section class="mb-4">
		<h2 class="doc-heading">Quick example</h2>
		<p class="doc-body mb-4">Buy 5 YES shares on market 42, outcome 1, using <code>curl</code>:</p>
		<pre class="doc-code">{`# 1. Check current price
curl https://astroledger.de/api/markets/42

# 2. Buy shares
curl -X POST https://astroledger.de/api/trades \\
  -H "Authorization: Bearer smt_YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"market_id":42,"outcome_id":1,"action":"buy","shares":5}'`}</pre>
	</section>

	<!-- OpenAPI spec -->
	<section class="mb-4">
		<h2 class="doc-heading">OpenAPI spec</h2>
		<p class="doc-body">
			<a href="/openapi.yaml" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline">Download OpenAPI spec (YAML)</a>
		</p>
	</section>
</div>

<style>
	.doc-heading {
		font-size: 0.7rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.12em;
		color: var(--color-surface-500);
		border-bottom: 1px solid var(--color-surface-200);
		padding-bottom: 0.5rem;
		margin-bottom: 0.75rem;
	}
	.doc-body {
		font-size: 0.875rem;
		color: var(--app-text);
		line-height: 1.65;
	}
	.doc-code {
		background: var(--card-bg);
		color: var(--app-text);
		border: 1px solid var(--color-surface-200);
		border-radius: 0.5rem;
		padding: 1rem 1.25rem;
		font-size: 0.78rem;
		font-family: ui-monospace, 'Cascadia Code', 'Fira Mono', monospace;
		overflow-x: auto;
		white-space: pre;
		line-height: 1.6;
	}
	.doc-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8rem;
	}
	.doc-table th {
		text-align: left;
		padding: 0.5rem 0.75rem;
		background: var(--color-surface-100);
		color: var(--color-surface-500);
		font-weight: 600;
		text-transform: uppercase;
		font-size: 0.65rem;
		letter-spacing: 0.08em;
		border-bottom: 1px solid var(--color-surface-200);
	}
	.doc-table td {
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid var(--color-surface-100);
		color: var(--app-text);
	}
	.doc-table tr:last-child td {
		border-bottom: none;
	}
	.method-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.15rem 0.55rem;
		border-radius: 0.3rem;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		font-family: ui-monospace, monospace;
	}
	.method-get {
		background: #dbeafe;
		color: #1d4ed8;
		border: 1px solid #bfdbfe;
	}
	.method-post {
		background: #dcfce7;
		color: #15803d;
		border: 1px solid #bbf7d0;
	}
	.scope-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.1rem 0.45rem;
		border-radius: 9999px;
		font-size: 0.65rem;
		font-weight: 600;
		background: var(--color-surface-100);
		color: var(--color-surface-500);
		border: 1px solid var(--color-surface-200);
	}
	.scope-trade {
		background: #fef9c3;
		color: #a16207;
		border-color: #fde047;
	}
	.scope-create {
		background: #fce7f3;
		color: #9d174d;
		border-color: #fbcfe8;
	}

	.doc-tip {
		border: 1px solid #fde047;
		background: #fef9c3;
		color: #92400e;
		font-size: 0.875rem;
	}
	:global(:root[data-theme='dark']) .doc-tip {
		background: #422006;
		border-color: #92400e;
		color: #fcd34d;
	}
	:global(:root[data-theme='dark']) .method-get {
		background: #1e3a5f;
		color: #93c5fd;
		border-color: #1d4ed8;
	}
	:global(:root[data-theme='dark']) .method-post {
		background: #14532d;
		color: #86efac;
		border-color: #15803d;
	}
	:global(:root[data-theme='dark']) .scope-trade {
		background: #422006;
		color: #fbbf24;
		border-color: #92400e;
	}
	:global(:root[data-theme='dark']) .scope-create {
		background: #4a1942;
		color: #f9a8d4;
		border-color: #9d174d;
	}
</style>
