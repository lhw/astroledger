package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lhw/astroledger/internal/db"
)

type contextKey string

// UserClaimsKey is the context key for storing parsed session claims.
const UserClaimsKey contextKey = "user_claims"

// Claims represents the payload stored in the session JWT cookie.
type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// Session validates the session cookie and stores Claims in the request context.
// Requests without a valid session proceed unauthenticated (claims will be nil).
// Banned users have their session cookie cleared and continue as unauthenticated,
// so the ban takes effect immediately on the next request rather than at next login.
func Session(secret string, queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims := &Claims{}
			_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil {
				// Invalid or expired token — clear the cookie and continue unauthenticated.
				http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
				next.ServeHTTP(w, r)
				return
			}

			// Invalidate the session immediately if the user is banned.
			if banStatus, banErr := queries.GetUserBanStatus(r.Context(), claims.UserID); banErr == nil && banStatus.IsBanned == 1 {
				http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SessionOrToken is like Session but also accepts a Bearer token in the Authorization header.
// If a session cookie is present, it takes precedence. Otherwise, the Bearer token is validated
// and the same Claims structure is set in the context. This allows a single handler to work
// with both browser (session) and API (token) authentication.
func SessionOrToken(secret string, queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try session cookie first.
			cookie, err := r.Cookie("session")
			if err == nil {
				claims := &Claims{}
				_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
					}
					return []byte(secret), nil
				})
				if err == nil {
					// Invalidate the session immediately if the user is banned.
					if banStatus, banErr := queries.GetUserBanStatus(r.Context(), claims.UserID); banErr == nil && banStatus.IsBanned == 1 {
						http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
					} else {
						ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				} else {
					// Invalid or expired token — clear the cookie.
					http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
				}
			}

			// No valid session — try Bearer token.
			if tokenRow, err := authenticateBearerToken(r, queries); err == nil {
				claims := &Claims{UserID: tokenRow.UserID}
				ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
				botInfo := &BotTokenInfo{
					TokenID:          tokenRow.ID,
					UserID:           tokenRow.UserID,
					CanRead:          tokenRow.CanRead == 1,
					CanTrade:         tokenRow.CanTrade == 1,
					CanCreateMarkets: tokenRow.CanCreateMarkets == 1,
				}
				ctx = context.WithValue(ctx, BotTokenInfoCtxKey, botInfo)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No auth — proceed unauthenticated.
			next.ServeHTTP(w, r)
		})
	}
}

// authenticateBearerToken parses and validates a Bearer token from the Authorization header.
func authenticateBearerToken(r *http.Request, queries *db.Queries) (*db.GetAPITokenByHashRow, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("Authorization must be Bearer <token>")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return nil, fmt.Errorf("empty bearer token")
	}

	hash := hashTokenString(token)
	row, err := queries.GetAPITokenByHash(r.Context(), hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("token lookup: %w", err)
	}

	// Check if revoked.
	if row.RevokedAt != nil {
		return nil, fmt.Errorf("token revoked")
	}

	// Touch last_used_at (fire-and-forget).
	_ = queries.TouchAPITokenLastUsed(r.Context(), row.ID)

	return &row, nil
}

// hashTokenString computes the SHA-256 hex digest of a token string.
func hashTokenString(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// BotTokenInfo holds the bot token row when authenticated via Bearer token.
// Set by SessionOrToken when a bot token is used.
type BotTokenInfo struct {
	TokenID          int64
	UserID           int64
	CanRead          bool
	CanTrade         bool
	CanCreateMarkets bool
}

type botTokenInfoKey contextKey

const BotTokenInfoCtxKey contextKey = "bot_token_info"

// SetBotTokenInfo stores bot token info in the request context.
func SetBotTokenInfo(r *http.Request, info *BotTokenInfo) *http.Request {
	ctx := context.WithValue(r.Context(), BotTokenInfoCtxKey, info)
	return r.WithContext(ctx)
}

// GetBotTokenInfo retrieves bot token info from the context. Returns nil if not a bot token request.
func GetBotTokenInfo(r *http.Request) *BotTokenInfo {
	v, _ := r.Context().Value(BotTokenInfoCtxKey).(*BotTokenInfo)
	return v
}

// RequireScope checks that the current auth has the required scope.
// For session cookies, this always passes (session users have full access).
// For bot tokens, it checks the specific scope.
func RequireScope(r *http.Request, scope string) bool {
	info := GetBotTokenInfo(r)
	if info == nil {
		// Session auth — full access.
		return true
	}
	switch scope {
	case "read":
		return info.CanRead
	case "trade":
		return info.CanTrade
	case "create_markets":
		return info.CanCreateMarkets
	default:
		return false
	}
}

// RequireAuth rejects unauthenticated requests with 401.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetClaims(r) == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireMod rejects non-moderator requests with 403 using the current DB role state.
func RequireMod(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			user, err := queries.GetUserByID(r.Context(), claims.UserID)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
				return
			}
			if user.IsModerator != 1 && user.IsAdmin != 1 {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin rejects non-admin requests with 403 using the current DB role state.
func RequireAdmin(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			user, err := queries.GetUserByID(r.Context(), claims.UserID)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
				return
			}
			if user.IsAdmin != 1 {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims retrieves session claims from the context. Returns nil if unauthenticated.
func GetClaims(r *http.Request) *Claims {
	v, _ := r.Context().Value(UserClaimsKey).(*Claims)
	return v
}

// IssueSessionCookie signs and writes a session JWT as an httpOnly cookie.
func IssueSessionCookie(w http.ResponseWriter, secret string, userID int64, secure bool) error {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signed,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   int((72 * time.Hour).Seconds()),
	})
	return nil
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
