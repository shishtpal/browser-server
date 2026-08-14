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
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func requestToken(r *http.Request) (string, string) {
	if header := r.Header.Get("Authorization"); header != "" {
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", "malformed Authorization header"
		}
		return parts[1], ""
	}
	return r.URL.Query().Get("token"), ""
}

// Auth validates the Authorization: Bearer <token> header against the loaded
// operator token. It returns 401 for missing/invalid tokens, and 503 if the
// server has no token configured.
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

		token, malformed := requestToken(r)
		if malformed != "" {
			writeAuthError(w, http.StatusUnauthorized, malformed)
			return
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

// AdminAuth protects administrator-only routes with the separate opt-in admin
// token. The operator token is intentionally not accepted. A missing admin
// token disables the entire admin API with a stable 403 response.
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if !auth.AdminConfigured() {
			writeAuthError(w, http.StatusForbidden, "admin_disabled")
			return
		}

		token, malformed := requestToken(r)
		if malformed != "" {
			writeAuthError(w, http.StatusUnauthorized, malformed)
			return
		}
		if token == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing admin API token")
			return
		}
		if !auth.AdminValid(token) {
			writeAuthError(w, http.StatusUnauthorized, "invalid admin API token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
