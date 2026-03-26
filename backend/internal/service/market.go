package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lhw/scolymarket/internal/db"
)

// Valid market categories.
var validCategories = map[string]bool{
	"bug_fixes":        true,
	"feature_delivery": true,
	"patch_timing":     true,
	"community_events": true,
	"meta":             true,
}

// MarketService handles business logic for markets.
type MarketService struct {
	queries  *db.Queries
	sqlDB    *sql.DB
	badgeSvc *BadgeService
}

// NewMarketService creates a MarketService.
func NewMarketService(queries *db.Queries, sqlDB *sql.DB, badgeSvc *BadgeService) *MarketService {
	return &MarketService{queries: queries, sqlDB: sqlDB, badgeSvc: badgeSvc}
}

// CreateMarketInput is the validated input for creating a market.
type CreateMarketInput struct {
	Title               string
	Description         string
	Category            string
	ResolutionCriteria  string
	Deadline            time.Time
	CreatedBy           int64
	ResolutionType      string  // "binary" | "date" | "numeric"
	ResolutionThreshold *string // target date (ISO) or threshold value
}

// CreateMarket validates input, runs auto-filter, and inserts a new market.
func (s *MarketService) CreateMarket(ctx context.Context, inp CreateMarketInput) (*db.Market, error) {
	if err := s.validateMarketInput(ctx, inp); err != nil {
		return nil, err
	}

	resType := inp.ResolutionType
	if resType == "" {
		resType = "binary"
	}

	market, err := s.queries.CreateMarket(ctx, db.CreateMarketParams{
		Title:               inp.Title,
		Description:         inp.Description,
		Category:            inp.Category,
		ResolutionCriteria:  inp.ResolutionCriteria,
		ResolutionDeadline:  inp.Deadline,
		CreatedBy:           inp.CreatedBy,
		LiquidityParam:      100.0,
		ResolutionType:      resType,
		ResolutionThreshold: inp.ResolutionThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("create market: %w", err)
	}
	return &market, nil
}

// ApproveMarket moves a market from pending_review to active (moderator action).
func (s *MarketService) ApproveMarket(ctx context.Context, marketID, modID int64) error {
	market, err := s.queries.GetMarketByID(ctx, marketID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get market: %w", err)
	}
	if market.Status != "pending_review" {
		return fmt.Errorf("market is not pending_review (status: %s)", market.Status)
	}
	if err := s.queries.UpdateMarketStatus(ctx, db.UpdateMarketStatusParams{
		Status: "active",
		ID:     marketID,
	}); err != nil {
		return fmt.Errorf("update market status: %w", err)
	}
	// Award a 50 bUEC creator bonus when the market goes live.
	if err := s.queries.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{
		Balance: 50,
		ID:      market.CreatedBy,
	}); err != nil {
		slog.Warn("creator bonus failed (non-fatal)", "market_id", marketID, "creator_id", market.CreatedBy, "err", err)
	}
	slog.Info("market approved", "market_id", marketID, "mod_id", modID, "creator_id", market.CreatedBy)
	return nil
}

// RejectMarket moves a market to cancelled (moderator action).
func (s *MarketService) RejectMarket(ctx context.Context, marketID, modID int64) error {
	market, err := s.queries.GetMarketByID(ctx, marketID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get market: %w", err)
	}
	if market.Status != "pending_review" {
		return fmt.Errorf("market is not pending_review (status: %s)", market.Status)
	}
	if err := s.queries.UpdateMarketStatus(ctx, db.UpdateMarketStatusParams{
		Status: "cancelled",
		ID:     marketID,
	}); err != nil {
		return fmt.Errorf("update market status: %w", err)
	}
	slog.Info("market rejected", "market_id", marketID, "mod_id", modID)
	return nil
}

// ResolveInput is the input for resolving a market.
type ResolveInput struct {
	MarketID     int64
	Resolution   string // "yes" or "no"
	ModID        int64
	EvidenceLink *string
}

// ResolveMarket resolves a market and pays out winners within a transaction.
func (s *MarketService) ResolveMarket(ctx context.Context, inp ResolveInput) error {
	if inp.Resolution != "yes" && inp.Resolution != "no" {
		return fmt.Errorf("resolution must be 'yes' or 'no'")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qTx := s.queries.WithBoundTx(tx)

	market, err := qTx.GetMarketByID(ctx, inp.MarketID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get market: %w", err)
	}
	if market.Status != "active" && market.Status != "resolution_requested" {
		return fmt.Errorf("market must be active or resolution_requested (status: %s)", market.Status)
	}

	if err := qTx.ResolveMarket(ctx, db.ResolveMarketParams{
		Resolution:         &inp.Resolution,
		ResolvedBy:         &inp.ModID,
		ResolutionEvidence: inp.EvidenceLink,
		ID:                 inp.MarketID,
	}); err != nil {
		return fmt.Errorf("resolve market: %w", err)
	}

	// Pay out winners (each winning share pays 100 ScollyBucks).
	positions, err := qTx.GetPositionsForResolution(ctx, inp.MarketID)
	if err != nil {
		return fmt.Errorf("get positions: %w", err)
	}

	for _, pos := range positions {
		var winShares float64
		if inp.Resolution == "yes" {
			winShares = pos.YesShares
		} else {
			winShares = pos.NoShares
		}
		if winShares <= 0 {
			continue
		}
		payout := int64(winShares * 100)
		if payout <= 0 {
			continue
		}
		if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{
			Balance: payout,
			ID:      pos.UserID,
		}); err != nil {
			return fmt.Errorf("payout user %d: %w", pos.UserID, err)
		}
		slog.Info("payout", "user_id", pos.UserID, "payout", payout)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit resolution: %w", err)
	}

	// Award badges to everyone who had a position — non-blocking.
	if s.badgeSvc != nil {
		for _, pos := range positions {
			userID := pos.UserID
			go s.badgeSvc.CheckAndAward(context.Background(), userID)
		}
	}
	return nil
}

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrForbidden is returned when the caller does not own the resource.
var ErrForbidden = errors.New("forbidden")

// ErrRejected is returned when a market is rejected by the auto-filter.
var ErrRejected = errors.New("market rejected by auto-filter")

// RequestResolutionInput holds the parameters for a resolution request.
type RequestResolutionInput struct {
	MarketID int64
	CallerID int64
	Link     string
	Note     string
}

// RequestResolution flags an active market for moderator resolution.
// Any user who holds shares in the market may call this; a short link and
// explanatory note are stored so mods have context.
func (s *MarketService) RequestResolution(ctx context.Context, inp RequestResolutionInput) error {
	market, err := s.queries.GetMarketByID(ctx, inp.MarketID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get market: %w", err)
	}
	if market.Status != "active" {
		return fmt.Errorf("market must be active to request resolution (status: %s)", market.Status)
	}
	// Require the caller to have a position in the market.
	pos, err := s.queries.GetUserPositionOrZero(ctx, inp.CallerID, inp.MarketID)
	if err != nil {
		return fmt.Errorf("get position: %w", err)
	}
	if pos.YesShares == 0 && pos.NoShares == 0 {
		return ErrForbidden
	}
	if err := validateResolutionRequestMetadata(inp.Link, inp.Note); err != nil {
		return err
	}
	if err := s.queries.UpdateMarketStatus(ctx, db.UpdateMarketStatusParams{
		Status: "resolution_requested",
		ID:     inp.MarketID,
	}); err != nil {
		return fmt.Errorf("update market status: %w", err)
	}
	// Store optional link/note for the mod team.
	var link, note *string
	if inp.Link != "" {
		link = &inp.Link
	}
	if inp.Note != "" {
		note = &inp.Note
	}
	if err := s.queries.UpsertResolutionRequestDetails(ctx, inp.MarketID, inp.CallerID, link, note); err != nil {
		// Non-fatal: status already updated; log and continue.
		slog.Warn("failed to store resolution request details", "err", err)
	}
	slog.Info("resolution requested", "market_id", inp.MarketID, "user_id", inp.CallerID)
	return nil
}

func validateResolutionRequestMetadata(link, note string) error {
	if len(note) > 500 {
		return fmt.Errorf("resolution note too long (max 500 chars)")
	}
	if link == "" {
		return nil
	}
	if len(link) > 2048 {
		return fmt.Errorf("resolution link too long (max 2048 chars)")
	}
	parsed, err := url.ParseRequestURI(link)
	if err != nil {
		return fmt.Errorf("resolution link must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("resolution link must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("resolution link must include a host")
	}
	return nil
}

func (s *MarketService) validateMarketInput(ctx context.Context, inp CreateMarketInput) error {
	if utf8.RuneCountInString(strings.TrimSpace(inp.Title)) < 10 {
		return fmt.Errorf("title must be at least 10 characters")
	}
	if utf8.RuneCountInString(inp.Title) > 200 {
		return fmt.Errorf("title too long (max 200 chars)")
	}
	if !validCategories[inp.Category] {
		return fmt.Errorf("invalid category: %s", inp.Category)
	}
	if inp.Deadline.Before(time.Now().Add(24 * time.Hour)) {
		return fmt.Errorf("deadline must be at least 24 hours in the future")
	}
	if err := s.runAutoFilter(ctx, inp.Title+" "+inp.Description); err != nil {
		return err
	}
	return s.checkDuplicateTitle(ctx, inp.Title)
}

// checkDuplicateTitle queries active and pending markets and rejects if any
// existing title is too similar (Jaccard trigram similarity > 0.6).
func (s *MarketService) checkDuplicateTitle(ctx context.Context, newTitle string) error {
	existing, err := s.queries.GetActivePendingMarketTitles(ctx)
	if err != nil {
		return fmt.Errorf("check duplicates: %w", err)
	}

	newSet := trigramSet(strings.ToLower(newTitle))
	for _, title := range existing {
		if jaccardSimilarity(newSet, trigramSet(strings.ToLower(title))) > 0.6 {
			return fmt.Errorf("%w: a very similar market already exists — %q", ErrRejected, title)
		}
	}
	return nil
}

// trigramSet returns the set of character trigrams for a string.
func trigramSet(s string) map[string]struct{} {
	runes := []rune(s)
	set := make(map[string]struct{}, len(runes))
	for i := 0; i+2 < len(runes); i++ {
		set[string(runes[i:i+3])] = struct{}{}
	}
	return set
}

// jaccardSimilarity returns |A∩B| / |A∪B|, or 0 if both sets are empty.
func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func (s *MarketService) runAutoFilter(ctx context.Context, text string) error {
	// Load enabled auto-filter rules from the DB each time (small table, fine for ~200 users).
	rules, err := s.queries.GetEnabledAutofilterRules(ctx)
	if err != nil {
		return fmt.Errorf("load autofilter rules: %w", err)
	}

	lower := strings.ToLower(text)
	for _, rule := range rules {
		switch rule.RuleType {
		case "keyword":
			if strings.Contains(lower, strings.ToLower(rule.Value)) {
				return fmt.Errorf("%w: contains banned keyword %q", ErrRejected, rule.Value)
			}
		case "regex":
			re, compErr := regexp.Compile(rule.Value)
			if compErr != nil {
				continue // skip bad rules
			}
			if re.MatchString(lower) {
				return fmt.Errorf("%w: matches banned pattern", ErrRejected)
			}
		case "min_length":
			var minLen int
			if _, err2 := fmt.Sscanf(rule.Value, "%d", &minLen); err2 == nil {
				if utf8.RuneCountInString(strings.TrimSpace(text)) < minLen {
					return fmt.Errorf("%w: submission too short (minimum %d characters)", ErrRejected, minLen)
				}
			}
		}
	}
	return nil
}
