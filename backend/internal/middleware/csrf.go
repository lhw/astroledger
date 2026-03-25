package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// RequireTrustedOrigin rejects state-changing browser requests that do not
// originate from the configured frontend origins.
func RequireTrustedOrigin(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		normalized := normalizeOrigin(origin)
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresCSRFValidation(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			origin := normalizeOrigin(r.Header.Get("Origin"))
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, `{"error":"forbidden origin"}`, http.StatusForbidden)
				return
			}

			referer := normalizeOrigin(refererOrigin(r.Header.Get("Referer")))
			if referer != "" {
				if _, ok := allowed[referer]; ok {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, `{"error":"forbidden origin"}`, http.StatusForbidden)
				return
			}

			http.Error(w, `{"error":"missing origin"}`, http.StatusForbidden)
		})
	}
}

func requiresCSRFValidation(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func refererOrigin(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	origin = strings.TrimRight(origin, "/")
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
