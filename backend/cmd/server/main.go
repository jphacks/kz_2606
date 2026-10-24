package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/jphacks/kz_2606/backend/internal/interface/rest"
)

// エントリポイント。ここは配線（DI）だけ。ロジックを書かない。
//
//	cfg      := config.Load()
//	rooms    := persistence.NewRoomRepository(pool)   // or memory.NewRoomRepository()
//	points   := persistence.NewPointRepository(pool)
//	registry := ws.NewRegistry()
//	sessions := usecase.NewSessionService(rooms, points, registry)
//	mux      := rest.NewRouter(usecase.NewRoomService(rooms), sessions, registry, staticFS())
func main() {
	mux := rest.NewRouter()

	// TODO: 上記の配線

	addr := ":" + env("PORT", "8080")
	slog.Info("listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("listen failed", "err", err)
		os.Exit(1)
	}

}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
