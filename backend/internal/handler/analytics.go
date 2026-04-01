package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// AnalyticsHandler accepts client-side page-view pings and forwards them to
// GoatCounter in batches to avoid rate-limit (429) responses from GoatCounter's
// /api/v0/count endpoint. Hits are buffered in a channel and flushed either
// every flushInterval or when the buffer reaches maxBatch entries.
type AnalyticsHandler struct {
	gcURL    string // GoatCounter internal base URL
	gcAPIKey string // GoatCounter API Bearer token
	queue    chan gcOutboundHit
}

const (
	flushInterval = 10 * time.Second
	maxBatch      = 25
	queueCap      = 500
)

type gcOutboundHit struct {
	Path      string `json:"path"`
	Title     string `json:"title,omitempty"`
	Ref       string `json:"ref,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	IP        string `json:"ip,omitempty"`
}

// NewAnalyticsHandler creates an AnalyticsHandler. When gcURL or gcAPIKey is
// empty the handler silently accepts requests without forwarding them.
func NewAnalyticsHandler(gcURL, gcAPIKey string) *AnalyticsHandler {
	return &AnalyticsHandler{
		gcURL:    gcURL,
		gcAPIKey: gcAPIKey,
		queue:    make(chan gcOutboundHit, queueCap),
	}
}

// Run starts the background flush loop. It blocks until ctx is cancelled and
// should be called in a goroutine. It performs one final flush on shutdown.
func (h *AnalyticsHandler) Run(ctx context.Context) {
	if h.gcURL == "" || h.gcAPIKey == "" {
		// Drain the queue silently so it never blocks callers.
		for {
			select {
			case <-h.queue:
			case <-ctx.Done():
				return
			}
		}
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	buf := make([]gcOutboundHit, 0, maxBatch)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		h.sendBatch(context.Background(), buf)
		buf = buf[:0]
	}

	for {
		select {
		case hit := <-h.queue:
			buf = append(buf, hit)
			if len(buf) >= maxBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// Drain anything still queued before exiting.
		drain:
			for {
				select {
				case hit := <-h.queue:
					buf = append(buf, hit)
				default:
					break drain
				}
			}
			flush()
			return
		}
	}
}

func (h *AnalyticsHandler) sendBatch(ctx context.Context, hits []gcOutboundHit) {
	payload, err := json.Marshal(map[string]any{
		"no_sessions": true,
		"hits":        hits,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
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
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		slog.Warn("analytics: GoatCounter rejected batch", "status", resp.StatusCode,
			"url", fmt.Sprintf("%s/api/v0/count", h.gcURL), "body", string(respBody), "hits", len(hits))
	}
}

type hitRequest struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Ref   string `json:"ref"`
}

// Hit accepts a client-side navigation event and enqueues it for batched
// forwarding to GoatCounter. Always returns 204 — callers fire-and-forget.
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

	hit := gcOutboundHit{
		Path:      path,
		Title:     body.Title,
		Ref:       body.Ref,
		UserAgent: ua,
		IP:        ip,
	}

	// Non-blocking enqueue — drop if the buffer is full (burst protection).
	select {
	case h.queue <- hit:
	default:
		slog.Warn("analytics: hit queue full, dropping hit", "path", path)
	}
}
