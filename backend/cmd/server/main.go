package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/infrastructure/auth"
	"github.com/jphacks/kz_2606/backend/internal/infrastructure/config"
	"github.com/jphacks/kz_2606/backend/internal/infrastructure/persistence"
	"github.com/jphacks/kz_2606/backend/internal/interface/rest"
	"github.com/jphacks/kz_2606/backend/internal/interface/ws"
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
	if !cfg.CookieSecure {
		slog.Warn("COOKIE_SECURE=false; the session cookie is sent over plain HTTP (development only)")
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
	roomService := usecase.NewRoomService(persistence.NewRoomRepository(pool))
	mux := rest.NewRouter(
		authService,
		roomService,
		rest.CookieOptions{
			Secure:   cfg.CookieSecure,
			Lifetime: auth.SessionLifetime,
		},
	)
	mux.Handle("/ws", ws.NewHandler(authService, roomService, ws.NewRegistry()))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
		// ReadTimeout / WriteTimeout は設定しない。/ws の長時間接続を切ってしまうため。
		// スローロリス対策はヘッダ読み込みとアイドルの打ち切りで行う。
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", server.Addr)
		listenErr <- server.ListenAndServe()
	}()

	select {
	case err := <-listenErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownCtx.Done():
		slog.Info("shutting down")
		graceCtx, cancelGrace := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelGrace()
		return server.Shutdown(graceCtx)
	}
}
