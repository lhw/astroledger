package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/lhw/scolymarket/internal/config"
	"github.com/lhw/scolymarket/internal/database"
	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/handler"
	"github.com/lhw/scolymarket/internal/middleware"
	"github.com/lhw/scolymarket/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	setupLogger(cfg.LogLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var sqlDB *sql.DB
	if cfg.UsePostgres() {
		slog.Info("connecting to PostgreSQL", "url", "DATABASE_URL")
		sqlDB, err = database.OpenPostgres(ctx, cfg.DatabaseURL)
	} else {
		sqlDB, err = database.Open(ctx, cfg.DBPath)
	}
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer sqlDB.Close()

	var queries *db.Queries
	if cfg.UsePostgres() {
		queries = db.NewPostgres(sqlDB)
	} else {
		queries = db.New(sqlDB)
	}

	// Services
	badgeSvc := service.NewBadgeService(queries)
	marketSvc := service.NewMarketService(queries, sqlDB, badgeSvc)
	tradingSvc := service.NewTradingService(queries, sqlDB, badgeSvc)
	creditsSvc := service.NewCreditsService(queries)
	patchScraper := service.NewPatchScraper(queries)
	modClient := service.NewModerationClient(cfg.ModerationAPIKey)
	commentSvc := service.NewCommentService(queries, modClient)

	// Handlers
	authH, err := handler.NewAuthHandler(ctx, queries,
		cfg.SCIDIssuer, cfg.SCIDClientID, cfg.SCIDClientSecret, cfg.SCIDRedirectURL,
		cfg.SessionSecret, cfg.FrontendURL, cfg.CookieSecure,
	)
	if err != nil {
		return fmt.Errorf("auth handler: %w", err)
	}

	userH := handler.NewUserHandler(queries)
	marketH := handler.NewMarketHandler(queries, marketSvc)
	tradeH := handler.NewTradingHandler(tradingSvc)
	modH := handler.NewModerationHandler(queries, badgeSvc)
	commentH := handler.NewCommentHandler(commentSvc)
	adminH := handler.NewAdminHandler(queries, creditsSvc, cfg.GoatCounterURL, cfg.GoatCounterAPIKey)
	patchH := handler.NewPatchHandler(queries)

	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Session(cfg.SessionSecret))

	// Auth routes (rate-limited)
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(10, time.Minute))
		r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
		r.Get("/auth/login", authH.Login)
		r.Get("/auth/callback", authH.Callback)
		r.Post("/auth/logout", authH.Logout)
	})

	// Public API routes
	r.Route("/api", func(r chi.Router) {
		// Users
		r.Get("/me", userH.Me)
		r.Get("/users/{id}", userH.GetUser)
		r.Get("/leaderboard", userH.Leaderboard)

		// Authenticated user routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Get("/me/positions", userH.GetUserPositions)
			r.Get("/me/trades", userH.GetUserTrades)
			r.Get("/me/badges", modH.GetMyBadges)
			r.Put("/me/badge", modH.SetActiveBadge)
		})

		r.Get("/users/{id}/badges", modH.GetUserBadges)

		// FOMO store
		r.Get("/fomo", modH.GetStoreBadges)
		r.Get("/admiral", modH.GetAdmiralRanks)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/fomo/purchase", modH.PurchaseBadge)
		})

		// Patches (public)
		r.Get("/patches", patchH.List)

		// Markets (read-only, public)
		r.Get("/markets", marketH.List)
		r.Get("/markets/{id}", marketH.Get)
		r.Get("/markets/{id}/history", marketH.GetPriceHistory)
		r.Get("/markets/{id}/trades", marketH.GetTrades)
		r.Get("/markets/{id}/comments", commentH.List)

		// Authenticated market creation (rate-limited to 5/min)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(5, time.Minute))
			r.Post("/markets", marketH.Create)
		})

		// Creator requests mod resolution
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Post("/markets/{id}/request-resolution", marketH.RequestResolution)
		})

		// Trading (rate-limited to 30/min)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(30, time.Minute))
			r.Post("/trades", tradeH.Trade)
		})

		// Comment submission (rate-limited to 10/min)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/markets/{id}/comments", commentH.Post)
		})

		// Report submission (rate-limited to 10/min)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/reports", modH.SubmitReport)
		})

		// Moderation routes (mod-only; includes comment deletion)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireMod(queries))
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Delete("/comments/{id}", commentH.Delete)
			r.Get("/mod/markets", marketH.ListPending)
			r.Post("/mod/markets/{id}/approve", marketH.Approve)
			r.Post("/mod/markets/{id}/reject", marketH.Reject)
			r.Post("/mod/markets/{id}/resolve", marketH.Resolve)
			r.Get("/mod/resolution-requests", marketH.ListResolutionRequested)
			r.Post("/mod/markets/{id}/deny-resolution", marketH.DenyResolution)
			r.Get("/mod/reports", modH.ListReports)
			r.Post("/mod/reports/{id}/review", modH.ReviewReport)
			r.Post("/mod/reports/{id}/dismiss", modH.DismissReport)
			r.Post("/mod/patches/{id}/notify", patchH.MarkNotified)
		})

		// Admin routes (admin check performed inside handlers)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Post("/admin/weekly-payout", adminH.TriggerWeeklyPayout)
			r.Get("/admin/users/search", adminH.SearchUsers)
			r.Post("/admin/users/{id}/balance", adminH.AdjustUserBalance)
			r.Get("/admin/analytics", adminH.AnalyticsProxy)
		})
	})
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	// Background market expiry job — runs hourly; idempotent.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		runExpiry := func() {
			if err := queries.ExpirePendingMarkets(context.Background()); err != nil {
				slog.Warn("expiry: pending markets", "err", err)
			}
			if err := queries.ExpireOverdueActiveMarkets(context.Background()); err != nil {
				slog.Warn("expiry: overdue active markets", "err", err)
			}
		}
		runExpiry()
		for {
			select {
			case <-ticker.C:
				runExpiry()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Patch scraper goroutine — checks Spectrum forum every 30 minutes.
	go patchScraper.Run(ctx)

	// Weekly payout goroutine — checks every hour; idempotent so safe to run often.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		// Run immediately at startup to catch any missed payouts.
		if _, err := creditsSvc.WeeklyPayout(ctx); err != nil {
			slog.Warn("weekly payout error", "err", err)
		}
		for {
			select {
			case <-ticker.C:
				if _, err := creditsSvc.WeeklyPayout(ctx); err != nil {
					slog.Warn("weekly payout error", "err", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
	}
	return nil
}

func setupLogger(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
