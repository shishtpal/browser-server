package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"browser-server/internal/auth"
)

func TestAdminAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bs-token-admin")
	t.Setenv("SERVER_ADMIN_TOKEN_PATH", path)
	_, _ = auth.AdminDelete()
	defer auth.AdminDelete()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := AdminAuth(next)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unconfigured status = %d, want 403", response.Code)
	}

	token, _, err := auth.AdminGenerate()
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.AdminLoad(); err != nil {
		t.Fatal(err)
	}

	for name, supplied := range map[string]struct {
		token string
		code  int
	}{
		"missing": {code: http.StatusUnauthorized},
		"wrong":   {token: "wrong", code: http.StatusUnauthorized},
		"valid":   {token: token, code: http.StatusNoContent},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
			if supplied.token != "" {
				req.Header.Set("Authorization", "Bearer "+supplied.token)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != supplied.code {
				t.Fatalf("status = %d, want %d", recorder.Code, supplied.code)
			}
		})
	}

	options := httptest.NewRequest(http.MethodOptions, "/api/admin/status", nil)
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, options)
	if optionsResponse.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", optionsResponse.Code)
	}
}
