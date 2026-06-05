package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// API base URL from environment or default
func apiBase() string {
	if v := os.Getenv("ASTROLEDGER_API_URL"); v != "" {
		return v
	}
	return "https://astroledger.de"
}

// Bot token from environment
func botToken() string {
	return os.Getenv("ASTROLEDGER_BOT_TOKEN")
}

func getTools() []Tool {
	return []Tool{
		// Best practices
		{
			Name:        "astroledger_best_practices",
			Description: "Get best practices for creating markets, formatting questions, resolution criteria, and using the AstroLedger API effectively.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		// Market tools
		{
			Name:        "astroledger_list_markets",
			Description: "List active markets with optional category filter. Returns paginated market data including current prices.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": "Filter by category: bug_fixes, feature_delivery, patch_timing, community_events, meta",
						"enum":        []string{"bug_fixes", "feature_delivery", "patch_timing", "community_events", "meta"},
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Pagination offset (default 0)",
					},
				},
			},
		},
		{
			Name:        "astroledger_get_market",
			Description: "Get detailed information about a specific market including outcomes, prices, and trade statistics.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "The market ID",
					},
				},
				"required": []string{"market_id"},
			},
		},
		{
			Name:        "astroledger_create_market",
			Description: "Create a new prediction market. Requires can_create_markets scope. Markets are created in pending_review status.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Market question (10-200 chars)",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Detailed description of the market",
					},
					"category": map[string]any{
						"type":        "string",
						"description": "Market category",
						"enum":        []string{"bug_fixes", "feature_delivery", "patch_timing", "community_events", "meta"},
					},
					"resolution_criteria": map[string]any{
						"type":        "string",
						"description": "How the market resolves (YES if... NO if...)",
					},
					"deadline": map[string]any{
						"type":        "string",
						"description": "Patch version (e.g. '4.9.0') or RFC3339 date",
					},
					"outcomes": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Custom outcomes (defaults to ['YES', 'NO'])",
					},
				},
				"required": []string{"title", "description", "category", "resolution_criteria", "deadline"},
			},
		},
		{
			Name:        "astroledger_get_market_trades",
			Description: "Get recent trades for a specific market.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "The market ID",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Pagination offset",
					},
				},
				"required": []string{"market_id"},
			},
		},
		{
			Name:        "astroledger_get_market_history",
			Description: "Get price history for a market (for charting).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "The market ID",
					},
				},
				"required": []string{"market_id"},
			},
		},
		// Trading tools
		{
			Name:        "astroledger_get_me",
			Description: "Get authenticated user info (balance, display name).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "astroledger_trade",
			Description: "Execute a buy or sell order on a market.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "Market ID to trade in",
					},
					"outcome_id": map[string]any{
						"type":        "integer",
						"description": "Outcome ID to buy/sell",
					},
					"action": map[string]any{
						"type":        "string",
						"description": "buy or sell",
						"enum":        []string{"buy", "sell"},
					},
					"shares": map[string]any{
						"type":        "number",
						"description": "Number of shares (1-10000)",
					},
				},
				"required": []string{"market_id", "outcome_id", "action", "shares"},
			},
		},
		{
			Name:        "astroledger_get_positions",
			Description: "Get current user positions across all markets.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "astroledger_get_trades",
			Description: "Get user's trade history.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"offset": map[string]any{
						"type":        "integer",
						"description": "Pagination offset",
					},
				},
			},
		},
		// Moderation tools
		{
			Name:        "astroledger_list_pending_markets",
			Description: "List markets awaiting moderation review. Requires mod/admin role.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "astroledger_approve_market",
			Description: "Approve a pending market (moves to active). Requires mod/admin role.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "Market ID to approve",
					},
				},
				"required": []string{"market_id"},
			},
		},
		{
			Name:        "astroledger_reject_market",
			Description: "Reject a pending market (moves to cancelled). Requires mod/admin role.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"market_id": map[string]any{
						"type":        "integer",
						"description": "Market ID to reject",
					},
				},
				"required": []string{"market_id"},
			},
		},
	}
}

func executeTool(name string, args map[string]any) (CallToolResult, error) {
	switch name {
	case "astroledger_best_practices":
		return bestPractices()
	case "astroledger_list_markets":
		return listMarkets(args)
	case "astroledger_get_market":
		return getMarket(args)
	case "astroledger_create_market":
		return createMarket(args)
	case "astroledger_get_market_trades":
		return getMarketTrades(args)
	case "astroledger_get_market_history":
		return getMarketHistory(args)
	case "astroledger_get_me":
		return getMe()
	case "astroledger_trade":
		return executeTrade(args)
	case "astroledger_get_positions":
		return getPositions()
	case "astroledger_get_trades":
		return getTrades(args)
	case "astroledger_list_pending_markets":
		return listPendingMarkets()
	case "astroledger_approve_market":
		return approveMarket(args)
	case "astroledger_reject_market":
		return rejectMarket(args)
	default:
		return CallToolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
}

func bestPractices() (CallToolResult, error) {
	text := `# AstroLedger Market Best Practices

## Market Question Formatting

### Structure
- Start with "Will" for yes/no questions
- Be specific about what, when, and where
- Include the patch version or timeframe in the question

### Examples
**Good:**
- "Will the Hull C be broken in a brand new way in 4.9?"
- "Will docking ports be fixed in 4.8.1?"
- "Will a new variant of the 30K error appear in 4.9?"

**Bad:**
- "Bug happening again" (too vague)
- "Will CIG fix things?" (unclear what "things" means)

## Categories

| Category | Use For |
|----------|---------|
| bug_fixes | Bugs, glitches, crashes, broken features |
| feature_delivery | New features, content, gameplay additions |
| patch_timing | When patches ship, release dates |
| community_events | Events, contests, community milestones |
| meta | Platform changes, economy, non-gameplay |

## Resolution Criteria

Always specify:
1. **What counts as YES** - concrete, verifiable conditions
2. **What counts as NO** - the alternative or failure case
3. **How to verify** - Issue Council, patch notes, in-game confirmation

### Example
"YES if docking ports function correctly after 4.8.1 maintenance updates. NO if docking remains broken or is not addressed in 4.8.1."

## Deadline Formats

### Patch Version (Recommended for bug/feature markets)
- Format: "4.9.0", "4.8.1"
- Sets deadline 2 years out automatically
- Prepends "Resolves when patch X ships." to criteria

### Date (For time-based markets)
- Format: RFC3339 ("2026-12-31T23:59:59Z")
- Use for end-of-year, quarterly, or specific date markets

## Title Length
- Minimum: 10 characters
- Maximum: 200 characters
- Sweet spot: 50-100 characters

## What Makes a Good Market
1. **Clear resolution** - No ambiguity about outcomes
2. **Verifiable** - Can be checked against official sources
3. **Interesting** - People want to bet on it
4. **Timely** - Relevant to current patch/event cycle
5. **Not already covered** - Check existing markets first`

	return CallToolResult{
		Content: []Content{{Type: "text", Text: text}},
	}, nil
}

func listMarkets(args map[string]any) (CallToolResult, error) {
	url := apiBase() + "/api/markets?status=active"
	if cat, ok := args["category"].(string); ok && cat != "" {
		url += "&category=" + cat
	}
	if off, ok := args["offset"].(float64); ok {
		url += fmt.Sprintf("&offset=%d", int(off))
	}
	return apiGet(url)
}

func getMarket(args map[string]any) (CallToolResult, error) {
	id, ok := args["market_id"].(float64)
	if !ok {
		return CallToolResult{}, fmt.Errorf("market_id required")
	}
	return apiGet(fmt.Sprintf("%s/api/markets/%d", apiBase(), int(id)))
}

func createMarket(args map[string]any) (CallToolResult, error) {
	body := map[string]any{
		"title":               args["title"],
		"description":         args["description"],
		"category":            args["category"],
		"resolution_criteria": args["resolution_criteria"],
		"deadline":            args["deadline"],
	}
	if outcomes, ok := args["outcomes"].([]any); ok {
		body["outcomes"] = outcomes
	}
	return apiPost(apiBase()+"/api/markets", body)
}

func getMarketTrades(args map[string]any) (CallToolResult, error) {
	id, ok := args["market_id"].(float64)
	if !ok {
		return CallToolResult{}, fmt.Errorf("market_id required")
	}
	url := fmt.Sprintf("%s/api/markets/%d/trades", apiBase(), int(id))
	if off, ok := args["offset"].(float64); ok {
		url += fmt.Sprintf("?offset=%d", int(off))
	}
	return apiGet(url)
}

func getMarketHistory(args map[string]any) (CallToolResult, error) {
	id, ok := args["market_id"].(float64)
	if !ok {
		return CallToolResult{}, fmt.Errorf("market_id required")
	}
	return apiGet(fmt.Sprintf("%s/api/markets/%d/history", apiBase(), int(id)))
}

func getMe() (CallToolResult, error) {
	return apiGet(apiBase() + "/api/me")
}

func executeTrade(args map[string]any) (CallToolResult, error) {
	body := map[string]any{
		"market_id":  args["market_id"],
		"outcome_id": args["outcome_id"],
		"action":     args["action"],
		"shares":     args["shares"],
	}
	return apiPost(apiBase()+"/api/trades", body)
}

func getPositions() (CallToolResult, error) {
	return apiGet(apiBase() + "/api/me/positions")
}

func getTrades(args map[string]any) (CallToolResult, error) {
	url := apiBase() + "/api/me/trades"
	if off, ok := args["offset"].(float64); ok {
		url += fmt.Sprintf("?offset=%d", int(off))
	}
	return apiGet(url)
}

func listPendingMarkets() (CallToolResult, error) {
	return apiGet(apiBase() + "/api/mod/markets")
}

func approveMarket(args map[string]any) (CallToolResult, error) {
	id, ok := args["market_id"].(float64)
	if !ok {
		return CallToolResult{}, fmt.Errorf("market_id required")
	}
	return apiPost(fmt.Sprintf("%s/api/mod/markets/%d/approve", apiBase(), int(id)), nil)
}

func rejectMarket(args map[string]any) (CallToolResult, error) {
	id, ok := args["market_id"].(float64)
	if !ok {
		return CallToolResult{}, fmt.Errorf("market_id required")
	}
	return apiPost(fmt.Sprintf("%s/api/mod/markets/%d/reject", apiBase(), int(id)), nil)
}

func apiGet(url string) (CallToolResult, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return CallToolResult{}, fmt.Errorf("create request: %w", err)
	}
	if token := botToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return CallToolResult{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return CallToolResult{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return CallToolResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))}},
			IsError: true,
		}, nil
	}

	// Pretty print JSON
	var pretty json.RawMessage
	if json.Unmarshal(body, &pretty) == nil {
		formatted, _ := json.MarshalIndent(pretty, "", "  ")
		body = formatted
	}

	return CallToolResult{
		Content: []Content{{Type: "text", Text: string(body)}},
	}, nil
}

func apiPost(url string, body any) (CallToolResult, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return CallToolResult{}, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return CallToolResult{}, fmt.Errorf("create request: %w", err)
	}
	if token := botToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", apiBase())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return CallToolResult{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return CallToolResult{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return CallToolResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))}},
			IsError: true,
		}, nil
	}

	// Pretty print JSON
	var pretty json.RawMessage
	if json.Unmarshal(respBody, &pretty) == nil {
		formatted, _ := json.MarshalIndent(pretty, "", "  ")
		respBody = formatted
	}

	return CallToolResult{
		Content: []Content{{Type: "text", Text: string(respBody)}},
	}, nil
}
