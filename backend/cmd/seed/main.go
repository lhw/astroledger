// cmd/seed populates a ScolyMarket database with sample markets in every
// lifecycle state so developers can work against realistic data.
//
// Usage:
//
//	go run ./cmd/seed              # seed sample data
//	go run ./cmd/seed -clean       # delete all data then re-seed
//	go run ./cmd/seed -clean-only  # delete all data, do NOT re-seed
//
// Database selection mirrors the server: set DATABASE_URL for PostgreSQL,
// otherwise DB_PATH (or the default scolymarket.db) for SQLite.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lhw/scolymarket/internal/config"
	"github.com/lhw/scolymarket/internal/database"
	"github.com/lhw/scolymarket/internal/db"
)

func main() {
	clean := flag.Bool("clean", false, "delete all data before seeding")
	cleanOnly := flag.Bool("clean-only", false, "delete all data and exit without seeding")
	flag.Parse()

	if err := run(*clean || *cleanOnly, *cleanOnly); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(doClean, cleanOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx := context.Background()

	var sqlDB *sql.DB
	if cfg.UsePostgres() {
		slog.Info("connecting to PostgreSQL")
		sqlDB, err = database.OpenPostgres(ctx, cfg.DatabaseURL)
	} else {
		slog.Info("connecting to SQLite", "path", cfg.DBPath)
		sqlDB, err = database.Open(ctx, cfg.DBPath)
	}
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer sqlDB.Close()

	if doClean {
		slog.Info("cleaning all data")
		if err := cleanData(ctx, sqlDB); err != nil {
			return fmt.Errorf("clean: %w", err)
		}
		slog.Info("data cleaned")
	}

	if cleanOnly {
		return nil
	}

	var queries *db.Queries
	if cfg.UsePostgres() {
		queries = db.NewPostgres(sqlDB)
	} else {
		queries = db.New(sqlDB)
	}

	return seed(ctx, queries)
}

// cleanData deletes every row from every application table in dependency order.
// Schema and migration metadata are preserved.
func cleanData(ctx context.Context, sqlDB *sql.DB) error {
	tables := []string{
		"admin_balance_adjustments",
		"comments",
		"resolution_request_details",
		"moderation_actions",
		"reports",
		"positions",
		"trades",
		"weekly_payout_log",
		"user_badges",
		"detected_patches",
		"markets",
		"users",
	}
	for _, t := range tables {
		if _, err := sqlDB.ExecContext(ctx, "DELETE FROM "+t); err != nil {
			slog.Warn("could not clean table", "table", t, "err", err)
		}
	}
	return nil
}

// seed inserts sample data covering every market lifecycle state.
func seed(ctx context.Context, queries *db.Queries) error {
	bot, err := queries.CreateUser(ctx, db.CreateUserParams{
		ScidSub:     "seed:system",
		DisplayName: "ScolyBot",
		Email:       "",
	})
	if err != nil {
		return fmt.Errorf("create system user: %w", err)
	}
	slog.Info("created system user", "id", bot.ID)

	if err := queries.UpdateUserGroups(ctx, bot.ID, 1, 0, 0); err != nil {
		return fmt.Errorf("set bot as moderator: %w", err)
	}

	type marketSpec struct {
		title      string
		category   string
		criteria   string
		deadline   time.Time
		status     string
		resolution *string
		liquidity  float64
	}

	yes := "yes"
	no := "no"
	now := time.Now().UTC()

	specs := []marketSpec{
		// Active
		{"Will the multi-cargo elevator desync be fixed in patch 4.1?", "bug_fixes",
			"Resolves YES if the elevator desync appears fixed in 4.1 patch notes.",
			now.AddDate(0, 3, 0), "active", nil, 150},
		{"Will the Stanton-Pyro jump point be stable at 4.1 launch?", "feature_delivery",
			"Resolves YES if 4.1 ships without a Known Issue for jump point instability.",
			now.AddDate(0, 2, 0), "active", nil, 120},
		{"Will patch 4.1 ship to live before 30 April 2026?", "patch_timing",
			"Resolves YES if 4.1 is live on or before 30 April 2026 (UTC).",
			time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC), "active", nil, 200},
		{"Will the Hull C be flyable before the end of 2026?", "feature_delivery",
			"Resolves YES if a flyable Hull C appears in any official build before 31 Dec 2026.",
			time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "active", nil, 100},
		{"Will IAE 2026 be held in November as usual?", "community_events",
			"Resolves YES if CIG runs an IAE event during November 2026.",
			time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC), "active", nil, 80},
		{"Will the inventory item duplication exploit be patched before 4.2?", "bug_fixes",
			"Resolves YES if any patch between now and 4.2 explicitly addresses the dupe exploit.",
			now.AddDate(0, 6, 0), "active", nil, 100},
		{"Will SQ42 ship in 2026?", "feature_delivery",
			"Resolves YES if Squadron 42 is commercially released before 31 Dec 2026.",
			time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "active", nil, 250},
		{"Will server frame rate hold above 20 fps average in Q2 2026?", "meta",
			"Resolves YES if community-aggregated server FPS shows a Q2 average above 20 fps.",
			time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC), "active", nil, 90},
		// Pending review
		{"Will server meshing support 4 shards by end of 2026?", "feature_delivery",
			"Resolves YES if CIG demonstrates 4-shard live meshing before 31 Dec 2026.",
			time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "pending_review", nil, 100},
		{"Will the Reclaimer tractor beam stop launching salvage into orbit?", "bug_fixes",
			"Resolves YES if the bug is absent from 4.1 Known Issues.",
			now.AddDate(0, 5, 0), "pending_review", nil, 120},
		{"Will there be a Free Fly event in Q3 2026?", "community_events",
			"Resolves YES if CIG runs any free-fly event Jul-Sep 2026.",
			time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC), "pending_review", nil, 80},
		{"Will the Carrack medical bay be fully functional in 4.1?", "bug_fixes",
			"Resolves YES if the Carrack medical bay operates without server restart in 4.1.",
			now.AddDate(0, 3, 0), "pending_review", nil, 70},
		{"Will CIG release a new flyable multi-crew ship in H1 2026?", "feature_delivery",
			"Resolves YES if a new 3+ crew ship is added as flyable in any build Jan-Jun 2026.",
			time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC), "pending_review", nil, 110},
		// Resolution requested
		{"Will the GI banding artefact be patched by end of Q1 2026?", "bug_fixes",
			"Resolves YES if Q1 2026 patch notes include a Lumen banding fix.",
			time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC), "resolution_requested", nil, 90},
		{"Will the 4.0.1 hotfix release before Christmas 2025?", "patch_timing",
			"Resolves YES if a 4.0.1 build appears on RSI Comm-Link before 25 Dec 2025.",
			time.Date(2025, 12, 25, 23, 59, 59, 0, time.UTC), "resolution_requested", nil, 180},
		// Deadline passed
		{"Will patch 4.0 ship before December 2025?", "patch_timing",
			"Resolves YES if patch 4.0 is live before 30 Nov 2025.",
			time.Date(2025, 11, 30, 23, 59, 59, 0, time.UTC), "deadline_passed", nil, 200},
		{"Will the Lorville cargo elevator be fixed before IAE 2025?", "bug_fixes",
			"Resolves YES if the elevator operates without desync for a full week before IAE 2025.",
			time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), "deadline_passed", nil, 90},
		// Resolved YES
		{"Will the Pyro system ship in any form in 2025?", "feature_delivery",
			"Resolves YES if Pyro becomes accessible in any official build during 2025.",
			time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "resolved", &yes, 300},
		{"Will CIG hold a CitizenCon in 2025?", "community_events",
			"Resolves YES if CIG hosts a CitizenCon during 2025.",
			time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "resolved", &yes, 100},
		{"Will dynamic cargo launch before 4.0?", "feature_delivery",
			"Resolves YES if dynamic cargo ships in any live patch numbered below 4.0.",
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "resolved", &yes, 150},
		{"Will the Mirai Fury MX be added as a flyable ship in 3.23?", "feature_delivery",
			"Resolves YES if the Fury MX appears as flyable in 3.23 live.",
			time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), "resolved", &yes, 80},
		// Resolved NO
		{"Will Star Citizen hit 1.0 before the end of 2025?", "meta",
			"Resolves YES if CIG officially ships Star Citizen 1.0 before 31 Dec 2025.",
			time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "resolved", &no, 500},
		{"Will base building ship before Q3 2025?", "feature_delivery",
			"Resolves YES if base building is playable in any live build before 1 Oct 2025.",
			time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC), "resolved", &no, 120},
		{"Will there be a server wipe in 4.0?", "meta",
			"Resolves NO if player items and aUEC carry over from 3.x to 4.0 live.",
			time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "resolved", &no, 160},
		// Cancelled
		{"Will the Kraken be made player purchasable in 2024?", "feature_delivery",
			"Resolves YES if the Kraken is listed for purchase on the RSI ship page during 2024.",
			time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), "cancelled", nil, 60},
		{"Will Orison get a full city ground level before 4.0?", "feature_delivery",
			"Resolves YES if Orison receives a walk-able ground-level city in any patch before 4.0.",
			time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "cancelled", nil, 70},
	}

	for _, s := range specs {
		m, err := queries.CreateMarket(ctx, db.CreateMarketParams{
			Title:              s.title,
			Description:        "",
			Category:           s.category,
			ResolutionCriteria: s.criteria,
			ResolutionDeadline: s.deadline,
			CreatedBy:          bot.ID,
			LiquidityParam:     s.liquidity,
			ResolutionType:     "binary",
		})
		if err != nil {
			return fmt.Errorf("create market %q: %w", s.title, err)
		}

		if s.status != "pending_review" {
			if err := queries.UpdateMarketStatus(ctx, db.UpdateMarketStatusParams{
				Status: s.status,
				ID:     m.ID,
			}); err != nil {
				return fmt.Errorf("set status %q: %w", s.title, err)
			}
		}

		if s.resolution != nil {
			if err := queries.ResolveMarket(ctx, db.ResolveMarketParams{
				Resolution: s.resolution,
				ResolvedBy: &bot.ID,
				ID:         m.ID,
			}); err != nil {
				return fmt.Errorf("resolve market %q: %w", s.title, err)
			}
		}

		slog.Info("seeded market", "id", m.ID, "status", s.status, "title", s.title)
	}

	slog.Info("seed complete", "markets", len(specs))
	return nil
}
