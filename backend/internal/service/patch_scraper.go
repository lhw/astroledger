package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lhw/astroledger/internal/db"
)

const statusRSSURL = "https://status.robertsspaceindustries.com/index.xml"

// liveVersionRe matches "SC Alpha X.Y.Z..." or "Star Citizen Alpha X.Y.Z..." in status posts.
// Only the semver triple (MAJOR.MINOR.PATCH) is captured; build suffixes like -live.11592622 are dropped.
var liveVersionRe = regexp.MustCompile(`(?:SC|Star Citizen)\s+Alpha\s+(\d+\.\d+\.\d+)`)

// rssChannel holds items parsed from the RSI status RSS feed.
type rssChannel struct {
	Items []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

// PatchScraper fetches and stores new LIVE patch versions from the RSI status RSS feed.
type PatchScraper struct {
	queries *db.Queries
	client  *http.Client
}

// NewPatchScraper creates a PatchScraper.
func NewPatchScraper(queries *db.Queries) *PatchScraper {
	return &PatchScraper{
		queries: queries,
		client: &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Block redirects to external domains (SSRF protection).
				if !strings.HasPrefix(req.URL.String(), "https://status.robertsspaceindustries.com/") {
					return fmt.Errorf("redirect to external domain blocked: %s", req.URL.Host)
				}
				return nil
			},
		},
	}
}

// Scrape fetches the RSI status RSS feed and returns insert params for each live deployment found.
func (s *PatchScraper) Scrape(ctx context.Context) ([]db.InsertPatchParams, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, statusRSSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "AstroLedger/1.0 patch-watcher")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch status RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from RSS feed: %d", resp.StatusCode)
	}

	var feed rssChannel
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}

	var patches []db.InsertPatchParams
	for _, item := range feed.Items {
		link := item.Link
		// Security: only accept links from the expected domain.
		if !strings.HasPrefix(link, "https://status.robertsspaceindustries.com/") {
			continue
		}
		// Skip PTU / EPTU / test deployments.
		titleUpper := strings.ToUpper(item.Title)
		if strings.Contains(titleUpper, "PTU") || strings.Contains(titleUpper, "EPTU") || strings.Contains(titleUpper, "TEST") {
			continue
		}
		// HTML-decode the description to get readable text, then extract the version.
		text := html.UnescapeString(item.Description)
		version := extractLiveVersion(text)
		if version == "" {
			continue
		}
		patches = append(patches, db.InsertPatchParams{
			Title:        cleanRSSTitle(item.Title),
			PatchVersion: version,
			ThreadUrl:    link,
		})
	}

	return patches, nil
}

// Run loops forever, calling Scrape every 30 minutes until ctx is cancelled.
func (s *PatchScraper) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	s.runOnce(ctx)
	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *PatchScraper) runOnce(ctx context.Context) {
	patches, err := s.Scrape(ctx)
	if err != nil {
		slog.Warn("patch scraper: scrape failed", "err", err)
		return
	}
	for _, p := range patches {
		if err := s.queries.InsertPatch(ctx, p); err != nil {
			slog.Warn("patch scraper: insert failed", "title", p.Title, "err", err)
		}
	}
	slog.Info("patch scraper: run complete", "found", len(patches))
}

// extractLiveVersion extracts the live SC version from an RSI status post description.
// It matches patterns like "SC Alpha 4.7.1-live.11592622" or "Star Citizen Alpha 4.7.0-live.11518367".
func extractLiveVersion(text string) string {
	m := liveVersionRe.FindStringSubmatch(text)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// cleanRSSTitle strips the "[Resolved] " prefix the RSS feed prepends to resolved incidents.
func cleanRSSTitle(title string) string {
	title = strings.TrimPrefix(title, "[Resolved] ")
	return strings.TrimSpace(title)
}
