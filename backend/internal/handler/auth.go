package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/lhw/scolymarket/internal/db"
	"github.com/lhw/scolymarket/internal/middleware"
)

// AuthHandler handles OIDC login, callback, and logout.
type AuthHandler struct {
	queries       *db.Queries
	oauth2Config  oauth2.Config
	verifier      *oidc.IDTokenVerifier
	sessionSecret string
	cookieSecure  bool
	frontendURL   string
}

// NewAuthHandler creates a new AuthHandler by discovering OIDC endpoints.
func NewAuthHandler(
	ctx context.Context,
	queries *db.Queries,
	issuer, clientID, clientSecret, redirectURL, sessionSecret, frontendURL string,
	cookieSecure bool,
) (*AuthHandler, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider discovery: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	oauth2Cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	return &AuthHandler{
		queries:       queries,
		oauth2Config:  oauth2Cfg,
		verifier:      verifier,
		sessionSecret: sessionSecret,
		cookieSecure:  cookieSecure,
		frontendURL:   frontendURL,
	}, nil
}

// Login redirects the user to the OIDC provider with state + PKCE cookies set.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomHex(16)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate state")
		return
	}

	codeVerifier, err := randomBase64URL(32)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate pkce verifier")
		return
	}
	codeChallenge := pkceChallenge(codeVerifier)

	setShortCookie(w, "oauth_state", state, h.cookieSecure)
	setShortCookie(w, "oauth_verifier", codeVerifier, h.cookieSecure)

	authURL := h.oauth2Config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the OIDC provider redirect, exchanges the code, and issues a session.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate state to prevent CSRF.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		respondError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	verifierCookie, err := r.Cookie("oauth_verifier")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing pkce verifier")
		return
	}

	// Clear the one-time auth cookies.
	clearShortCookie(w, "oauth_state")
	clearShortCookie(w, "oauth_verifier")

	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	tokens, err := h.oauth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifierCookie.Value),
	)
	if err != nil {
		slog.Error("oauth2 token exchange failed", "err", err)
		respondError(w, http.StatusBadRequest, "token exchange failed")
		return
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "missing id_token in response")
		return
	}

	idToken, err := h.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		slog.Error("id_token verification failed", "err", err)
		respondError(w, http.StatusUnauthorized, "id_token verification failed")
		return
	}

	// Parse standard typed claims. These are safe well-known OIDC fields (always strings/arrays).
	var claims struct {
		Sub     string   `json:"sub"`
		Name    string   `json:"name"`
		Email   string   `json:"email"`
		Groups  []string `json:"groups"`
		Picture string   `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to parse id_token claims")
		return
	}

	// Parse RSI-specific claims into a raw map so we tolerate providers that
	// return rsi_verified_at as a Unix timestamp (number) instead of a string.
	var rawClaims map[string]interface{}
	_ = idToken.Claims(&rawClaims) // best-effort; won't fail on type mismatches
	rsiHandle := stringClaim(rawClaims["rsi_handle"])
	rsiVerifiedAt := stringClaim(rawClaims["rsi_verified_at"])
	rsiEnlisted := stringClaim(rawClaims["rsi_enlisted"])
	rsiCitizenRecord := stringClaim(rawClaims["rsi_citizen_record"])

	// Determine mod/admin/rsi_verified from OIDC group membership.
	// scolymod = moderator access; admin = admin access; verified = RSI identity confirmed.
	isMod := containsGroup(claims.Groups, "scolymod")
	isAdmin := containsGroup(claims.Groups, "admin")
	isRsiVerified := containsGroup(claims.Groups, "verified")

	// Upsert user in the database.
	user, err := h.queries.GetUserBySub(ctx, claims.Sub)
	if errors.Is(err, sql.ErrNoRows) {
		user, err = h.queries.CreateUser(ctx, db.CreateUserParams{
			ScidSub:     claims.Sub,
			DisplayName: claims.Name,
			Email:       claims.Email,
		})
	}
	if err != nil {
		slog.Error("db user upsert failed", "err", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Update display name / email, RSI profile data, and last login timestamp.
	rsiVerifiedInt := int64(0)
	if isRsiVerified {
		rsiVerifiedInt = 1
	}
	if err := h.queries.UpdateUserLastLogin(ctx, db.UpdateUserLastLoginParams{
		DisplayName:      claims.Name,
		Email:            claims.Email,
		RsiHandle:        nullableString(rsiHandle),
		RsiVerifiedAt:    nullableString(rsiVerifiedAt),
		RsiEnlisted:      nullableString(rsiEnlisted),
		RsiCitizenRecord: nullableString(rsiCitizenRecord),
		AvatarUrl:        nullableString(claims.Picture),
		IsRsiVerified:    rsiVerifiedInt,
		ID:               user.ID,
	}); err != nil {
		slog.Warn("failed to update last_login", "err", err)
	}

	// Sync group-derived mod/admin/rsi_verified flags to the DB so /api/me reflects current membership.
	modInt := int64(0)
	if isMod {
		modInt = 1
	}
	adminInt := int64(0)
	if isAdmin {
		adminInt = 1
	}
	if err := h.queries.UpdateUserGroups(ctx, user.ID, modInt, adminInt, rsiVerifiedInt); err != nil {
		slog.Warn("failed to update user groups", "err", err)
	}

	if err := middleware.IssueSessionCookie(w, h.sessionSecret, user.ID, h.cookieSecure); err != nil {
		slog.Error("failed to issue session cookie", "err", err)
		respondError(w, http.StatusInternalServerError, "session error")
		return
	}

	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

// Logout clears the session cookie and redirects to the frontend.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearSessionCookie(w)
	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

// --- helpers ---

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// containsGroup returns true if group is present in the OIDC groups claim slice.
func containsGroup(groups []string, group string) bool {
	for _, g := range groups {
		if g == group {
			return true
		}
	}
	return false
}

func randomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func setShortCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   300, // 5-minute window
	})
}

func clearShortCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, MaxAge: -1, Path: "/"})
}

// nullableString returns a *string pointer for non-empty strings, nil otherwise.
// Used to convert empty OIDC claim strings into NULL for optional DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// stringClaim converts a raw JSON claim value (which may be a string or a number)
// into a string. Returns "" if the value is nil or an unsupported type.
func stringClaim(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// JSON numbers unmarshal as float64; format integers without decimals.
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprintf("%v", t)
	}
}
