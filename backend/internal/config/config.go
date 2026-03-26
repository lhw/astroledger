package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Environment string
	Port        string
	// DBPath is the SQLite database file path. Ignored when DatabaseURL is set.
	DBPath string
	// DatabaseURL is a PostgreSQL connection string (postgres://…).
	// When set, the backend switches to PostgreSQL and DBPath is ignored.
	DatabaseURL        string
	SCIDIssuer         string
	SCIDClientID       string
	SCIDClientSecret   string
	SCIDRedirectURL    string
	FrontendURL        string
	SessionSecret      string
	CORSAllowedOrigins []string
	LogLevel           string
	CookieSecure       bool
	// ModerationAPIKey is the OpenAI API key used for comment abuse detection.
	// Optional — when empty, automated moderation is disabled and all comments pass.
	ModerationAPIKey string

	// GoatCounterURL is the internal base URL of the GoatCounter analytics service.
	// Defaults to the Docker internal service name. Override for non-Docker setups.
	GoatCounterURL string
	// GoatCounterAPIKey is a GoatCounter API Bearer token created in the
	// GoatCounter settings (Settings → API tokens). Required for the
	// /api/admin/analytics endpoint to return data. Leave empty to disable.
	GoatCounterAPIKey string
}

// UsePostgres returns true when a PostgreSQL DATABASE_URL is configured.
func (c *Config) UsePostgres() bool {
	return c.DatabaseURL != ""
}

// Load reads configuration from the environment (and an optional .env file).
func Load() (*Config, error) {
	// Ignore error — .env is optional (not present in production).
	_ = godotenv.Load()

	cfg := &Config{
		Environment:       env("ENVIRONMENT", "development"),
		Port:              env("PORT", "8080"),
		DBPath:            envWithAliases([]string{"DB_PATH", "DATABASE_PATH"}, "./scolymarket.db"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SCIDIssuer:        os.Getenv("SCID_ISSUER"),
		SCIDClientID:      os.Getenv("SCID_CLIENT_ID"),
		SCIDClientSecret:  os.Getenv("SCID_CLIENT_SECRET"),
		SCIDRedirectURL:   os.Getenv("SCID_REDIRECT_URL"),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
		LogLevel:          env("LOG_LEVEL", "info"),
		CookieSecure:      cookieSecureDefault(),
		ModerationAPIKey:  os.Getenv("OPENAI_API_KEY"),
		GoatCounterURL:    env("GOATCOUNTER_URL", "http://goatcounter:8081"),
		GoatCounterAPIKey: os.Getenv("GOATCOUNTER_API_KEY"),
	}

	origins := envWithAliases([]string{"CORS_ALLOWED_ORIGINS", "ALLOWED_ORIGINS"}, "")
	if origins != "" {
		cfg.CORSAllowedOrigins = splitOrigins(origins)
	} else {
		cfg.CORSAllowedOrigins = []string{"http://localhost:5173"}
	}

	cfg.FrontendURL = envWithAliases([]string{"FRONTEND_URL", "SCID_POST_LOGIN_URL"}, "")
	if cfg.FrontendURL == "" && len(cfg.CORSAllowedOrigins) > 0 {
		cfg.FrontendURL = cfg.CORSAllowedOrigins[0]
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	required := map[string]string{
		"SCID_ISSUER":        c.SCIDIssuer,
		"SCID_CLIENT_ID":     c.SCIDClientID,
		"SCID_CLIENT_SECRET": c.SCIDClientSecret,
		"SCID_REDIRECT_URL":  c.SCIDRedirectURL,
		"SESSION_SECRET":     c.SessionSecret,
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("required environment variable %s is not set", name)
		}
	}
	if len(c.SessionSecret) < 32 {
		return fmt.Errorf("SESSION_SECRET must be at least 32 characters")
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.CORSAllowedOrigins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
	}
	parsedFrontend, err := url.Parse(c.FrontendURL)
	if err != nil || parsedFrontend.Scheme == "" || parsedFrontend.Host == "" {
		return fmt.Errorf("invalid FRONTEND_URL %q", c.FrontendURL)
	}
	return nil
}

func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		trimmed = strings.TrimRight(trimmed, "/")
		if trimmed == "" {
			continue
		}
		origins = append(origins, trimmed)
	}
	return origins
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envWithAliases(keys []string, fallback string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}

func cookieSecureDefault() bool {
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		return v != "false"
	}
	return env("ENVIRONMENT", "development") != "development"
}
