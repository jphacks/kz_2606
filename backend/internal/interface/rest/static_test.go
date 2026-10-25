package rest

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticHandlerServesAssetsAndFallsBackToSPA(t *testing.T) {
	handler := StaticHandler(fstest.MapFS{
		"index.html":       {Data: []byte("<html>app</html>")},
		"assets/app.js":    {Data: []byte("console.log('app')")},
		"assets/style.css": {Data: []byte("body { color: red }")},
	})

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{name: "root", path: "/", status: http.StatusOK, contentType: "text/html; charset=utf-8", body: "<html>app</html>"},
		{name: "spa route", path: "/login", status: http.StatusOK, contentType: "text/html; charset=utf-8", body: "<html>app</html>"},
		{name: "javascript asset", path: "/assets/app.js", status: http.StatusOK, body: "console.log('app')"},
		{name: "missing asset", path: "/assets/missing.js", status: http.StatusNotFound},
		{name: "api path", path: "/api/unknown", status: http.StatusNotFound},
		{name: "websocket path", path: "/ws/unknown", status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.status, response.Body)
			}
			if test.contentType != "" && !strings.HasPrefix(response.Header().Get("Content-Type"), test.contentType) {
				t.Fatalf("Content-Type = %q, want prefix %q", response.Header().Get("Content-Type"), test.contentType)
			}
			if test.name == "javascript asset" {
				contentType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
				if err != nil {
					t.Fatalf("parse JavaScript Content-Type: %v", err)
				}
				if contentType != "application/javascript" && contentType != "text/javascript" {
					t.Fatalf("JavaScript Content-Type = %q, want application/javascript or text/javascript", contentType)
				}
			}
			if test.body != "" && response.Body.String() != test.body {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.body)
			}
		})
	}
}

func TestStaticHandlerReportsMissingEntrypoint(t *testing.T) {
	handler := StaticHandler(fstest.MapFS{"assets/app.js": {Data: []byte("asset")}})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/some/route", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("missing SPA entrypoint status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
