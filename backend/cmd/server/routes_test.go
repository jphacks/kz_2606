package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jphacks/kz_2606/backend/internal/interface/rest"
	"github.com/jphacks/kz_2606/backend/internal/interface/ws"
)

// run() が組むのと同じ順序でルートを登録し、ServeMux が衝突で panic しないことを確認する。
// メソッド無しの "/ws" は "GET /"（SPA配信）と衝突すると判定され、起動時に panic する。
func TestRouteRegistrationDoesNotConflict(t *testing.T) {
	files, err := staticFS()
	if err != nil {
		t.Fatalf("create embedded static fs: %v", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ルート登録で panic しました（起動できません）: %v", recovered)
		}
	}()

	mux := rest.NewRouter(nil, nil, rest.CookieOptions{}, ws.NewRegistry())
	mux.Handle("GET /", rest.StaticHandler(files))
	mux.Handle("GET /ws", ws.NewHandler(nil, nil, ws.NewRegistry(), nil))

	// SPAフォールバックが WebSocket より手前で食わないことも確認する
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws?room=KZ26AB", nil)
	mux.ServeHTTP(response, request)
	if response.Code == http.StatusOK && response.Header().Get("Content-Type") == "text/html; charset=utf-8" {
		t.Fatal("/ws が SPA の index.html に吸われています")
	}
}
