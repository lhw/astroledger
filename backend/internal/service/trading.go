package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lhw/astroledger/internal/db"
)

// TradingService handles buy/sell operations.
type TradingService struct {
	queries  *db.Queries
	sqlDB    *sql.DB
	badgeSvc *BadgeService
}

// NewTradingService creates a TradingService.
func NewTradingService(queries *db.Queries, sqlDB *sql.DB, badgeSvc *BadgeService) *TradingService {
	return &TradingService{queries: queries, sqlDB: sqlDB, badgeSvc: badgeSvc}
}

// TradeInput is the validated request for a buy or sell.
type TradeInput struct {
	UserID    int64
	MarketID  int64
	OutcomeID int64   // FK into market_outcomes
	Action    string  // "buy" | "sell"
	Shares    float64 // number of shares (positive integer)
}

// TradeResult is the outcome of a trade.
type TradeResult struct {
	TradeID      int64
	Cost         int64 // positive = spent, negative = received (for sell)
	Shares       float64
	PriceAtTrade float64
	NewBalance   int64
}

// Execute performs a buy or sell trade atomically.
func (s *TradingService) Execute(ctx context.Context, inp TradeInput) (*TradeResult, error) {
	if inp.Action != "buy" && inp.Action != "sell" {
		return nil, fmt.Errorf("action must be 'buy' or 'sell'")
	}
	if inp.Shares <= 0 {
		return nil, fmt.Errorf("shares must be positive")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qTx := s.queries.WithBoundTx(tx)

	market, err := qTx.GetMarketByID(ctx, inp.MarketID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get market: %w", err)
	}
	if market.Status != "active" && market.Status != "resolution_requested" {
		return nil, fmt.Errorf("market is not open for trading")
	}
	if market.ResolutionDeadline.Before(time.Now()) {
		return nil, fmt.Errorf("market has passed its resolution deadline")
	}

	// Load all outcomes so we can compute LMSR prices.
	outcomes, err := qTx.GetOutcomesByMarketID(ctx, inp.MarketID)
	if err != nil {
		return nil, fmt.Errorf("get outcomes: %w", err)
	}
	if len(outcomes) < 2 {
		return nil, fmt.Errorf("market has fewer than 2 outcomes")
	}

	// Find the target outcome and its index in the share slice.
	outcomeIdx := -1
	for i, o := range outcomes {
		if o.ID == inp.OutcomeID {
			outcomeIdx = i
			break
		}
	}
	if outcomeIdx == -1 {
		return nil, fmt.Errorf("outcome %d does not belong to market %d", inp.OutcomeID, inp.MarketID)
	}

	allShares := make([]float64, len(outcomes))
	for i, o := range outcomes {
		allShares[i] = o.Shares
	}

	user, err := qTx.GetUserByID(ctx, inp.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	b := market.LiquidityParam
	price := OutcomeProb(b, allShares, outcomeIdx)

	var cost int64

	switch inp.Action {
	case "buy":
		cost = BuyCost(b, allShares, outcomeIdx, inp.Shares)
		if cost <= 0 {
			cost = 1
		}
		if user.Balance < cost {
			return nil, fmt.Errorf("insufficient balance: have %d, need %d", user.Balance, cost)
		}
		if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: -cost, ID: inp.UserID}); err != nil {
			return nil, fmt.Errorf("deduct balance: %w", err)
		}
		// Update share pool for this outcome.
		if err := qTx.UpdateOutcomeShares(ctx, db.UpdateOutcomeSharesParams{
			Shares: allShares[outcomeIdx] + inp.Shares,
			ID:     inp.OutcomeID,
		}); err != nil {
			return nil, fmt.Errorf("update outcome shares: %w", err)
		}
		// Record position.
		if err := qTx.UpsertPosition(ctx, db.UpsertPositionParams{
			UserID:    inp.UserID,
			MarketID:  inp.MarketID,
			OutcomeID: inp.OutcomeID,
			Shares:    inp.Shares,
		}); err != nil {
			return nil, fmt.Errorf("upsert position: %w", err)
		}

	case "sell":
		pos, posErr := qTx.GetUserPosition(ctx, db.GetUserPositionParams{
			UserID: inp.UserID, MarketID: inp.MarketID, OutcomeID: inp.OutcomeID,
		})
		if errors.Is(posErr, sql.ErrNoRows) {
			return nil, fmt.Errorf("you have no position in this outcome")
		}
		if posErr != nil {
			return nil, fmt.Errorf("get position: %w", posErr)
		}
		if pos.Shares < inp.Shares {
			return nil, fmt.Errorf("insufficient shares: have %.4f, selling %.4f", pos.Shares, inp.Shares)
		}

		revenue := SellRevenue(b, allShares, outcomeIdx, inp.Shares)
		if revenue < 0 {
			revenue = 0
		}
		cost = -revenue
		if revenue > 0 {
			if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: revenue, ID: inp.UserID}); err != nil {
				return nil, fmt.Errorf("credit balance: %w", err)
			}
		}
		// Update share pool for this outcome.
		if err := qTx.UpdateOutcomeShares(ctx, db.UpdateOutcomeSharesParams{
			Shares: allShares[outcomeIdx] - inp.Shares,
			ID:     inp.OutcomeID,
		}); err != nil {
			return nil, fmt.Errorf("update outcome shares: %w", err)
		}
		// Reduce position.
		if err := qTx.UpsertPosition(ctx, db.UpsertPositionParams{
			UserID:    inp.UserID,
			MarketID:  inp.MarketID,
			OutcomeID: inp.OutcomeID,
			Shares:    -inp.Shares,
		}); err != nil {
			return nil, fmt.Errorf("reduce position: %w", err)
		}
	}

	// Record trade.
	trade, err := qTx.CreateTrade(ctx, db.CreateTradeParams{
		UserID:       inp.UserID,
		MarketID:     inp.MarketID,
		OutcomeID:    inp.OutcomeID,
		Action:       inp.Action,
		Shares:       inp.Shares,
		Cost:         cost,
		PriceAtTrade: price,
	})
	if err != nil {
		return nil, fmt.Errorf("create trade: %w", err)
	}

	updatedUser, err := qTx.GetUserByID(ctx, inp.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user post-trade: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Non-blocking badge check.
	go func() {
		bCtx := context.Background()
		s.badgeSvc.CheckAndAward(bCtx, inp.UserID)
	}()

	return &TradeResult{
		TradeID:      trade.ID,
		Cost:         cost,
		Shares:       inp.Shares,
		PriceAtTrade: price,
		NewBalance:   updatedUser.Balance,
	}, nil
}
