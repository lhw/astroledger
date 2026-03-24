package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	"cig_drama":        true,
	"community_events": true,
	"meta":             true,
}

// MarketService handles business logic for markets.
type MarketService struct {
	queries *db.Queries
	sqlDB   *sql.DB
}

// NewMarketService creates a MarketService.
func NewMarketService(queries *db.Queries, sqlDB *sql.DB) *MarketService {
	return &MarketService{queries: queries, sqlDB: sqlDB}
}

// CreateMarketInput is the validated input for creating a market.
type CreateMarketInput struct {
	Title              string
	Description        string
	Category           string
	ResolutionCriteria string
	Deadline           time.Time
	CreatedBy          int64
}

// CreateMarket validates input, runs auto-filter, and inserts a new market.
func (s *MarketService) CreateMarket(ctx context.Context, inp CreateMarketInput) (*db.Market, error) {
	if err := s.validateMarketInput(ctx, inp); err != nil {
		return nil, err
	}

	market, err := s.queries.CreateMarket(ctx, db.CreateMarketParams{
		Title:              inp.Title,
		Description:        inp.Description,
		Category:           inp.Category,
		ResolutionCriteria: inp.ResolutionCriteria,
		ResolutionDeadline: inp.Deadline,
		CreatedBy:          inp.CreatedBy,
		LiquidityParam:     100.0,
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
	slog.Info("market approved", "market_id", marketID, "mod_id", modID)
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
	MarketID   int64
	Resolution string // "yes" or "no"
	ModID      int64
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

	qTx := s.queries.WithTx(tx)

	market, err := qTx.GetMarketByID(ctx, inp.MarketID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get market: %w", err)
	}
	if market.Status != "active" {
		return fmt.Errorf("market is not active (status: %s)", market.Status)
	}

	if err := qTx.ResolveMarket(ctx, db.ResolveMarketParams{
		Resolution: &inp.Resolution,
		ResolvedBy: &inp.ModID,
		ID:         inp.MarketID,
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

	return tx.Commit()
}

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrRejected is returned when a market is rejected by the auto-filter.
var ErrRejected = errors.New("market rejected by auto-filter")

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
	return s.runAutoFilter(ctx, inp.Title+" "+inp.Description)
}

func (s *MarketService) runAutoFilter(ctx context.Context, text string) error {
	// Load enabled auto-filter rules from the DB each time (small table, fine for ~200 users).
	rows, err := s.sqlDB.QueryContext(ctx,
		"SELECT rule_type, value FROM autofilter_rules WHERE enabled = 1")
	if err != nil {
		return fmt.Errorf("load autofilter rules: %w", err)
	}
	defer rows.Close()

	lower := strings.ToLower(text)
	for rows.Next() {
		var ruleType, value string
		if err := rows.Scan(&ruleType, &value); err != nil {
			continue
		}
		switch ruleType {
		case "keyword":
			if strings.Contains(lower, strings.ToLower(value)) {
				return fmt.Errorf("%w: contains banned keyword %q", ErrRejected, value)
			}
		case "regex":
			re, compErr := regexp.Compile(value)
			if compErr != nil {
				continue // skip bad rules
			}
			if re.MatchString(lower) {
				return fmt.Errorf("%w: matches banned pattern", ErrRejected)
			}
		}
	}
	return rows.Err()
}
