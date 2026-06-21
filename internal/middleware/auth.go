package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"browser-server/internal/auth"
)

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Auth validates the Authorization: Bearer <token> header against the loaded
// operator token. It returns 401 for missing/invalid tokens, and 503 if the
// server has no token configured (so the operator knows to run
// `server token generate`).
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow CORS preflight through; the CORS middleware already handles it.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if !auth.Configured() {
			writeAuthError(w, http.StatusServiceUnavailable, "server has no API token configured; run 'server token generate'")
			return
		}

		// Prefer the Authorization: Bearer header. Fall back to a ?token=
		// query param, which is needed for resources loaded via <img src>
		// (e.g. screenshots) that cannot set request headers.
		token := ""
		if header := r.Header.Get("Authorization"); header != "" {
			if bearer, ok := strings.CutPrefix(header, "Bearer "); ok {
				token = strings.TrimSpace(bearer)
			} else {
				writeAuthError(w, http.StatusUnauthorized, "malformed Authorization header")
				return
			}
		} else if q := r.URL.Query().Get("token"); q != "" {
			token = q
		}

		if token == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing API token")
			return
		}

		if !auth.Valid(token) {
			writeAuthError(w, http.StatusUnauthorized, "invalid API token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
