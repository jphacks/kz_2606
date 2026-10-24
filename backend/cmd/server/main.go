package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/infrastructure/auth"
	"github.com/jphacks/kz_2606/backend/internal/infrastructure/config"
	"github.com/jphacks/kz_2606/backend/internal/infrastructure/persistence"
	"github.com/jphacks/kz_2606/backend/internal/interface/rest"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.SessionSecretEphemeral {
		slog.Warn("SESSION_SECRET is not set; using a temporary key, sessions will expire when the server restarts")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := persistence.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()

	sessionCodec, err := auth.NewSessionCodec(cfg.SessionSecret)
	if err != nil {
		return err
	}
	authService := usecase.NewAuthService(
		persistence.NewUserRepository(pool),
		auth.Hasher{},
		sessionCodec,
	)
	mux := rest.NewRouter(authService)

	addr := ":" + cfg.Port
	slog.Info("listening", "addr", addr)
	return http.ListenAndServe(addr, mux)
}
