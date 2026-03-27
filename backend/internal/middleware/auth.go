package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lhw/scolymarket/internal/db"
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
func Session(secret string) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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
