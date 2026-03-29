package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// AnalyticsHandler accepts client-side page-view pings and forwards them to
// GoatCounter. Requests come from the SvelteKit frontend on the same origin,
// so they are invisible to browser extension blocklists.
type AnalyticsHandler struct {
	gcURL    string // GoatCounter internal base URL
	gcAPIKey string // GoatCounter API Bearer token
}

// NewAnalyticsHandler creates an AnalyticsHandler. When gcURL or gcAPIKey is
// empty the handler silently accepts requests without forwarding them.
func NewAnalyticsHandler(gcURL, gcAPIKey string) *AnalyticsHandler {
	return &AnalyticsHandler{gcURL: gcURL, gcAPIKey: gcAPIKey}
}

type hitRequest struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Ref   string `json:"ref"`
}

// Hit accepts a client-side navigation event and forwards it to GoatCounter.
// POST /api/analytics/hit
// Body: {"path":"/markets/42","title":"Market title","ref":"https://..."}
// Always returns 204 — callers should fire-and-forget.
func (h *AnalyticsHandler) Hit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)

	if h.gcURL == "" || h.gcAPIKey == "" {
		return
	}

	var body hitRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 4096))
	if err := dec.Decode(&body); err != nil || body.Path == "" {
		return
	}

	// Sanitise: path must start with / and contain no newlines.
	path := body.Path
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") {
		return
	}

	ua := r.Header.Get("User-Agent")
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	} else {
		ip = strings.SplitN(ip, ",", 2)[0]
	}
	lang := r.Header.Get("Accept-Language")

	payload, err := json.Marshal(map[string]any{
		"no_sessions": false,
		"hits": []map[string]any{{
			"path":       path,
			"title":      body.Title,
			"ref":        body.Ref,
			"user_agent": ua,
			"ip":         ip,
			"language":   lang,
		}},
	})
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		h.gcURL+"/api/v0/count", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.gcAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("analytics: GoatCounter unreachable", "err", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		slog.Warn("analytics: GoatCounter rejected hit", "status", resp.StatusCode,
			"url", fmt.Sprintf("%s/api/v0/count", h.gcURL))
	}
}
