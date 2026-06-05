<svelte:head>
	<title>Bot API Docs — AstroLedger</title>
	<meta name="description" content="API documentation for AstroLedger prediction market bots" />
</svelte:head>

<div class="container mx-auto px-4 max-w-3xl py-10">
	<div class="mb-8">
		<h1 class="text-2xl font-bold text-surface-900 mb-2">Bot API Documentation</h1>
		<p class="text-surface-600 text-sm">
			Authenticated endpoints for automated trading and market creation.
			All requests require a <code class="bg-surface-100 px-1 rounded">Bearer</code> token in the Authorization header.
		</p>
		<p class="text-surface-500 text-xs mt-2">
			<a href="/openapi.json" target="_blank" rel="noopener noreferrer" class="hover:text-primary-500">OpenAPI Spec (JSON) ↗</a>
		</p>
	</div>

	<div class="sc-card p-6">
		<h2 class="text-lg font-bold text-surface-900 mb-4">Endpoints</h2>

		<div class="space-y-6">
			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">Authentication</h3>
				<pre class="bg-surface-100 p-3 rounded text-xs overflow-x-auto">Authorization: Bearer smt_YourTokenHere</pre>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">GET /api/bot/me</h3>
				<p class="text-xs text-surface-600 mb-2">Returns user ID, display name, and balance.</p>
				<pre class="bg-surface-100 p-3 rounded text-xs overflow-x-auto">curl -H "Authorization: Bearer $TOKEN" https://astroledger.de/api/bot/me</pre>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">POST /api/bot/trades</h3>
				<p class="text-xs text-surface-600 mb-2">Buy or sell shares. Requires <code class="bg-surface-200 px-1 rounded">can_trade</code> scope.</p>
				<pre class="bg-surface-100 p-3 rounded text-xs overflow-x-auto">curl -X POST https://astroledger.de/api/bot/trades \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{`{"market_id": 1, "outcome_id": 1, "action": "buy", "shares": 10}`}'</pre>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">
					POST /api/bot/markets
					<span class="text-[10px] text-surface-500 font-normal ml-2">mod/admin only</span>
				</h3>
				<p class="text-xs text-surface-600 mb-2">Create a prediction market. Requires <code class="bg-surface-200 px-1 rounded">can_create_markets</code> scope.</p>
				<pre class="bg-surface-100 p-3 rounded text-xs overflow-x-auto">curl -X POST https://astroledger.de/api/bot/markets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{`{
    "title": "Will X happen in patch 4.9?",
    "description": "Details about the market...",
    "category": "bug_fixes",
    "resolution_criteria": "YES if... NO if...",
    "deadline": "4.9.0",
    "outcomes": ["YES", "NO"]
  }`}'</pre>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">Token Scopes</h3>
				<table class="w-full text-xs text-surface-600">
					<thead>
						<tr class="border-b border-surface-200">
							<th class="text-left py-1">Scope</th>
							<th class="text-left py-1">Access</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">can_read</code></td>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">GET /api/bot/me</code> (always enabled)</td>
						</tr>
						<tr>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">can_trade</code></td>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">POST /api/bot/trades</code></td>
						</tr>
						<tr>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">can_create_markets</code></td>
							<td class="py-1"><code class="bg-surface-100 px-1 rounded">POST /api/bot/markets</code> (mod/admin only)</td>
						</tr>
					</tbody>
				</table>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">Market Categories</h3>
				<p class="text-xs text-surface-600">
					<code class="bg-surface-100 px-1 rounded">bug_fixes</code>,
					<code class="bg-surface-100 px-1 rounded">feature_delivery</code>,
					<code class="bg-surface-100 px-1 rounded">patch_timing</code>,
					<code class="bg-surface-100 px-1 rounded">community_events</code>,
					<code class="bg-surface-100 px-1 rounded">meta</code>
				</p>
			</div>

			<div>
				<h3 class="text-sm font-bold text-surface-800 mb-2">Deadline Formats</h3>
				<ul class="text-xs text-surface-600 space-y-1">
					<li><strong>Patch version:</strong> <code class="bg-surface-100 px-1 rounded">"4.9.0"</code> — sets deadline 2 years out, prepends "Resolves when patch 4.9.0 ships."</li>
					<li><strong>ISO date:</strong> <code class="bg-surface-100 px-1 rounded">"2026-12-31T23:59:59Z"</code> — exact deadline</li>
				</ul>
			</div>
		</div>
	</div>
</div>
