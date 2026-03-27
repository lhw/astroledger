package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/lhw/astroledger/internal/db"
)

const spectrumPatchForumURL = "https://robertsspaceindustries.com/spectrum/community/SC/forum/190048"

var patchVersionRe = regexp.MustCompile(`Alpha\s+(\d+\.\d+[\w.]*)`)

// PatchScraper fetches and stores new LIVE patch notes from the RSI Spectrum forum.
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
				if !strings.HasPrefix(req.URL.String(), "https://robertsspaceindustries.com/") {
					return fmt.Errorf("redirect to external domain blocked: %s", req.URL.Host)
				}
				return nil
			},
		},
	}
}

// Scrape fetches the forum page and returns insert params for each LIVE thread found.
func (s *PatchScraper) Scrape(ctx context.Context) ([]db.InsertPatchParams, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, spectrumPatchForumURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AstroLedger/1.0; patch-watcher)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch forum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from forum: %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var patches []db.InsertPatchParams
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href string
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
				}
			}
			title := strings.TrimSpace(nodeText(n))
			if strings.Contains(href, "/spectrum/community/SC/forum/190048/thread/") && isLivePatchThread(title) {
				threadURL := href
				if strings.HasPrefix(threadURL, "/") {
					threadURL = "https://robertsspaceindustries.com" + href
				}
				// Security: must be on the expected domain.
				if !strings.HasPrefix(threadURL, "https://robertsspaceindustries.com/spectrum/") {
					return
				}
				patches = append(patches, db.InsertPatchParams{
					Title:        title,
					PatchVersion: extractPatchVersion(title),
					ThreadUrl:    threadURL,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

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

// isLivePatchThread returns true if the title is a LIVE patch note and not PTU/EPTU/etc.
func isLivePatchThread(title string) bool {
	upper := strings.ToUpper(title)
	if !strings.Contains(upper, "LIVE") {
		return false
	}
	for _, excl := range []string{"PTU", "EPTU", "EVOCATI", "TEST"} {
		if strings.Contains(upper, excl) {
			return false
		}
	}
	return true
}

// extractPatchVersion extracts the version number from a patch title, e.g. "4.0.2".
func extractPatchVersion(title string) string {
	m := patchVersionRe.FindStringSubmatch(title)
	if len(m) >= 2 {
		return m[1]
	}
	return "unknown"
}

// nodeText returns all text node content within n, concatenated.
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeText(c))
	}
	return sb.String()
}
