package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const openAIModerationURL = "https://api.openai.com/v1/moderations"

// shadowThresholds maps OpenAI category names to per-score thresholds above
// which a comment is shadow-hidden. These supplement OpenAI's own "flagged"
// signal and are tuned for a gaming community where mild profanity is normal.
var shadowThresholds = map[string]float64{
	"hate":                   0.70,
	"hate/threatening":       0.60,
	"harassment":             0.80,
	"harassment/threatening": 0.65,
	"violence":               0.75,
	"violence/graphic":       0.70,
	"self-harm":              0.70,
}

const pingCacheTTL = 5 * time.Minute

// ModerationClient calls the OpenAI Moderation API to detect abusive comments.
// When the API key is empty the client is a no-op: all comments pass through.
type ModerationClient struct {
	apiKey   string
	http     *http.Client
	cacheMu  sync.Mutex
	cached   *ModerationStatus
	cachedAt time.Time
}

// NewModerationClient creates a ModerationClient. apiKey may be empty to
// disable automated moderation (useful in development).
func NewModerationClient(apiKey string) *ModerationClient {
	return &ModerationClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

// Enabled reports whether the moderation API key is configured.
func (m *ModerationClient) Enabled() bool { return m.apiKey != "" }

// ModerationResult holds per-category scores and a shadow-hide decision.
type ModerationResult struct {
	// Scores maps OpenAI category names to their [0,1] probability scores.
	// Nil when moderation is disabled.
	Scores map[string]float64

	// Hide is true if OpenAI flagged the content OR any individual score
	// exceeded its per-category shadow threshold.
	Hide bool
}

type openAIRequest struct {
	Input string `json:"input"`
}

type openAIResponse struct {
	Results []struct {
		Flagged        bool               `json:"flagged"`
		CategoryScores map[string]float64 `json:"category_scores"`
	} `json:"results"`
}

type openAIErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ModerationStatus describes the runtime health of the OpenAI moderation integration.
type ModerationStatus struct {
	// Enabled is true when an API key is configured.
	Enabled bool `json:"enabled"`
	// OK is true when a test moderation call returned HTTP 200.
	OK bool `json:"ok"`
	// CreditsOK is false only when OpenAI returned an insufficient_quota error.
	// It is true when OK is true, and also true for non-quota failures (e.g. bad key).
	CreditsOK bool `json:"credits_ok"`
	// Error contains a human-readable failure description, empty on success.
	Error string `json:"error,omitempty"`
}

// Ping makes a minimal test moderation call and returns the integration status.
// Results are cached for pingCacheTTL to avoid burning rate-limit quota on
// every admin page load. Safe to call regardless of whether the client is enabled.
func (m *ModerationClient) Ping(ctx context.Context) *ModerationStatus {
	if !m.Enabled() {
		return &ModerationStatus{Enabled: false, CreditsOK: true}
	}

	m.cacheMu.Lock()
	if m.cached != nil && time.Since(m.cachedAt) < pingCacheTTL {
		result := *m.cached
		m.cacheMu.Unlock()
		return &result
	}
	m.cacheMu.Unlock()

	body, err := json.Marshal(openAIRequest{Input: "test"})
	if err != nil {
		return &ModerationStatus{Enabled: true, Error: err.Error(), CreditsOK: true}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIModerationURL, bytes.NewReader(body))
	if err != nil {
		return &ModerationStatus{Enabled: true, Error: err.Error(), CreditsOK: true}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.http.Do(req)
	if err != nil {
		return &ModerationStatus{Enabled: true, Error: err.Error(), CreditsOK: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return m.cacheAndReturn(&ModerationStatus{Enabled: true, OK: true, CreditsOK: true})
	}

	// HTTP 429 means the API key is valid and the service is reachable — we just
	// hit the rate limit (likely from testing). Treat as OK so repeated status
	// checks don't self-degrade the indicator.
	if resp.StatusCode == http.StatusTooManyRequests {
		return m.cacheAndReturn(&ModerationStatus{Enabled: true, OK: true, CreditsOK: true})
	}

	var oaErr openAIErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&oaErr)

	creditsOK := oaErr.Error.Code != "insufficient_quota" && oaErr.Error.Type != "insufficient_quota"
	msg := oaErr.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
	}
	// Don't cache failures — allow a retry on next page load.
	return &ModerationStatus{Enabled: true, CreditsOK: creditsOK, Error: msg}
}

func (m *ModerationClient) cacheAndReturn(s *ModerationStatus) *ModerationStatus {
	m.cacheMu.Lock()
	m.cached = s
	m.cachedAt = time.Now()
	m.cacheMu.Unlock()
	return s
}

// Moderate calls the OpenAI Moderation API and returns a ModerationResult.
//
// If the API key is not configured, a no-op result is returned immediately.
// If the call fails for any reason, an error is returned alongside a no-op
// result — callers should log the error but not block comment posting.
func (m *ModerationClient) Moderate(ctx context.Context, text string) (*ModerationResult, error) {
	if !m.Enabled() {
		return &ModerationResult{}, nil
	}

	body, err := json.Marshal(openAIRequest{Input: text})
	if err != nil {
		return &ModerationResult{}, fmt.Errorf("moderation: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIModerationURL, bytes.NewReader(body))
	if err != nil {
		return &ModerationResult{}, fmt.Errorf("moderation: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.http.Do(req)
	if err != nil {
		return &ModerationResult{}, fmt.Errorf("moderation: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ModerationResult{}, fmt.Errorf("moderation: unexpected status %d", resp.StatusCode)
	}

	var oaResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaResp); err != nil {
		return &ModerationResult{}, fmt.Errorf("moderation: decode: %w", err)
	}
	if len(oaResp.Results) == 0 {
		return &ModerationResult{}, fmt.Errorf("moderation: empty results")
	}

	result := oaResp.Results[0]
	hide := result.Flagged

	scores := result.CategoryScores
	if scores == nil {
		scores = map[string]float64{}
	}

	// Also check per-category score thresholds (gaming-community tuned).
	if !hide {
		for cat, threshold := range shadowThresholds {
			if score, ok := scores[cat]; ok && score >= threshold {
				hide = true
				break
			}
		}
	}

	return &ModerationResult{Scores: scores, Hide: hide}, nil
}
