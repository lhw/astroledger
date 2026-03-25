package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lhw/scolymarket/internal/db"
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
	UserID   int64
	MarketID int64
	Side     string  // "yes" | "no"
	Action   string  // "buy" | "sell"
	Shares   float64 // number of shares
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
	if inp.Side != "yes" && inp.Side != "no" {
		return nil, fmt.Errorf("side must be 'yes' or 'no'")
	}
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
	defer tx.Rollback()

	qTx := s.queries.WithTx(tx)

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

	user, err := qTx.GetUserByID(ctx, inp.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	sideYes := inp.Side == "yes"
	b := market.LiquidityParam
	qYes := market.YesShares
	qNo := market.NoShares
	price := YESPrice(b, qYes, qNo)

	var cost int64
	var deltaYes, deltaNp float64

	switch inp.Action {
	case "buy":
		cost = BuyCost(b, qYes, qNo, inp.Shares, sideYes)
		if cost <= 0 {
			cost = 1
		}
		if user.Balance < cost {
			return nil, fmt.Errorf("insufficient balance: have %d, need %d", user.Balance, cost)
		}
		if sideYes {
			deltaYes = inp.Shares
		} else {
			deltaNp = inp.Shares
		}
		// Deduct cost from user.
		if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: -cost, ID: inp.UserID}); err != nil {
			return nil, fmt.Errorf("deduct balance: %w", err)
		}

	case "sell":
		// Check the user actually holds enough shares.
		pos, posErr := qTx.GetUserPosition(ctx, db.GetUserPositionParams{UserID: inp.UserID, MarketID: inp.MarketID})
		if errors.Is(posErr, sql.ErrNoRows) {
			return nil, fmt.Errorf("you have no position in this market")
		}
		if posErr != nil {
			return nil, fmt.Errorf("get position: %w", posErr)
		}
		if sideYes && pos.YesShares < inp.Shares {
			return nil, fmt.Errorf("insufficient YES shares: have %.4f, selling %.4f", pos.YesShares, inp.Shares)
		}
		if !sideYes && pos.NoShares < inp.Shares {
			return nil, fmt.Errorf("insufficient NO shares: have %.4f, selling %.4f", pos.NoShares, inp.Shares)
		}

		revenue := SellRevenue(b, qYes, qNo, inp.Shares, sideYes)
		if revenue < 0 {
			revenue = 0
		}
		cost = -revenue // negative cost = money received

		if sideYes {
			deltaYes = -inp.Shares
		} else {
			deltaNp = -inp.Shares
		}
		// Credit user with revenue.
		if revenue > 0 {
			if err := qTx.UpdateUserBalance(ctx, db.UpdateUserBalanceParams{Balance: revenue, ID: inp.UserID}); err != nil {
				return nil, fmt.Errorf("credit balance: %w", err)
			}
		}
	}

	// Update AMM pool state.
	if err := qTx.UpdateMarketAMMState(ctx, db.UpdateMarketAMMStateParams{
		YesShares: qYes + deltaYes,
		NoShares:  qNo + deltaNp,
		ID:        inp.MarketID,
	}); err != nil {
		return nil, fmt.Errorf("update amm state: %w", err)
	}

	// Update user position (negative delta for sells is handled by UpsertPosition's ADD logic).
	if err := qTx.UpsertPosition(ctx, db.UpsertPositionParams{
		UserID:    inp.UserID,
		MarketID:  inp.MarketID,
		YesShares: deltaYes,
		NoShares:  deltaNp,
	}); err != nil {
		return nil, fmt.Errorf("upsert position: %w", err)
	}

	// Record the trade.
	absCost := cost
	if absCost < 0 {
		absCost = -absCost
	}
	trade, err := qTx.CreateTrade(ctx, db.CreateTradeParams{
		UserID:       inp.UserID,
		MarketID:     inp.MarketID,
		Side:         inp.Side,
		Action:       inp.Action,
		Shares:       inp.Shares,
		Cost:         absCost,
		PriceAtTrade: price,
	})
	if err != nil {
		return nil, fmt.Errorf("create trade record: %w", err)
	}

	// Fetch updated balance.
	updatedUser, err := qTx.GetUserByID(ctx, inp.UserID)
	if err != nil {
		return nil, fmt.Errorf("get updated user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	slog.Info("trade executed",
		"trade_id", trade.ID,
		"user_id", inp.UserID,
		"market_id", inp.MarketID,
		"action", inp.Action,
		"side", inp.Side,
		"shares", inp.Shares,
		"cost", cost,
	)

	// Check badge eligibility non-blockingly after the transaction is committed.
	if s.badgeSvc != nil {
		userID := inp.UserID
		go s.badgeSvc.CheckAndAward(context.Background(), userID)
	}

	return &TradeResult{
		TradeID:      trade.ID,
		Cost:         cost,
		Shares:       inp.Shares,
		PriceAtTrade: price,
		NewBalance:   updatedUser.Balance,
	}, nil
}
