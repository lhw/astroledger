package main

import (
	"context"
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

	sqlDB, err := database.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer sqlDB.Close()

	queries := db.New(sqlDB)

	// Services
	badgeSvc := service.NewBadgeService(queries)
	marketSvc := service.NewMarketService(queries, sqlDB, badgeSvc)
	tradingSvc := service.NewTradingService(queries, sqlDB, badgeSvc)
	creditsSvc := service.NewCreditsService(queries)

	// Handlers
	frontendURL := cfg.CORSAllowedOrigins[0]
	authH, err := handler.NewAuthHandler(ctx, queries,
		cfg.SCIDIssuer, cfg.SCIDClientID, cfg.SCIDClientSecret, cfg.SCIDRedirectURL,
		cfg.SessionSecret, frontendURL, cfg.CookieSecure,
	)
	if err != nil {
		return fmt.Errorf("auth handler: %w", err)
	}

	userH := handler.NewUserHandler(queries)
	marketH := handler.NewMarketHandler(queries, marketSvc)
	tradeH := handler.NewTradingHandler(tradingSvc)
	modH := handler.NewModerationHandler(queries, badgeSvc)

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
		r.Get("/auth/login", authH.Login)
		r.Get("/auth/callback", authH.Callback)
		r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
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
		})

		r.Get("/users/{id}/badges", modH.GetUserBadges)

		// Markets (read-only is public)
		r.Get("/markets", marketH.List)
		r.Get("/markets/{id}", marketH.Get)
		r.Get("/markets/{id}/history", marketH.GetPriceHistory)
		r.Get("/markets/{id}/trades", marketH.GetTrades)

		// Authenticated market creation (rate-limited)
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

		// Trading (rate-limited)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(30, time.Minute))
			r.Post("/trades", tradeH.Trade)
			r.Post("/trades/quote", tradeH.Quote)
		})

		// Report submission (auth required, rate-limited)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/reports", modH.SubmitReport)
		})

		// Moderation routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireMod(queries))
			r.Use(middleware.RequireTrustedOrigin(cfg.CORSAllowedOrigins))
			r.Get("/mod/markets", marketH.ListPending)
			r.Post("/mod/markets/{id}/approve", marketH.Approve)
			r.Post("/mod/markets/{id}/reject", marketH.Reject)
			r.Post("/mod/markets/{id}/resolve", marketH.Resolve)
			r.Get("/mod/resolution-requests", marketH.ListResolutionRequested)
			r.Post("/mod/markets/{id}/deny-resolution", marketH.DenyResolution)
			r.Get("/mod/reports", modH.ListReports)
			r.Post("/mod/reports/{id}/review", modH.ReviewReport)
			r.Post("/mod/reports/{id}/dismiss", modH.DismissReport)
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
