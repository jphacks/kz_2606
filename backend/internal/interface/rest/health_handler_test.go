package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jphacks/kz_2606/backend/internal/interface/ws"
)

func TestHealthEndpoint(t *testing.T) {
	router := NewRouter(nil, nil, CookieOptions{}, ws.NewRegistry())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", contentType)
	}
	if body := response.Body.String(); body != "{\"ok\":true}\n" {
		t.Fatalf("body = %q, want %q", body, "{\"ok\":true}\n")
	}
}
