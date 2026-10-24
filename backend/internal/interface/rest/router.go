package rest

import (
	"net/http"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/interface/ws"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

// ServeMux の組み立て。
// 仕様: DATA_MODEL.md 5章

// CookieOptions はセッション Cookie の発行条件。
// Lifetime は SessionCodec が埋め込む失効時刻と必ず揃えること（main.go で渡す）。
type CookieOptions struct {
	Secure   bool
	Lifetime time.Duration
}

func NewRouter(authService *usecase.AuthService, roomService *usecase.RoomService, cookies CookieOptions, registries ...*ws.Registry) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rootHandler)
	registry := ws.NewRegistry()
	if len(registries) > 0 && registries[0] != nil {
		registry = registries[0]
	}
	handler := &authHandler{service: authService, cookies: cookies}
	rooms := &roomHandler{rooms: roomService, registry: registry}
	admin := &adminHandler{rooms: roomService, registry: registry}
	mux.HandleFunc("POST /api/auth/register", handler.register)
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.Handle("GET /api/auth/me", requireUser(authService, http.HandlerFunc(handler.me)))
	mux.Handle("POST /api/rooms", requireUser(authService, http.HandlerFunc(rooms.create)))
	mux.HandleFunc("GET /api/rooms/{code}", rooms.get)
	mux.Handle("POST /api/rooms/{code}/join", requireUser(authService, http.HandlerFunc(rooms.join)))
	mux.Handle("POST /api/rooms/{code}/close", requireUser(authService, http.HandlerFunc(admin.close)))
	mux.Handle("POST /api/rooms/{code}/reopen", requireUser(authService, http.HandlerFunc(admin.reopen)))
	mux.Handle("POST /api/rooms/{code}/members/{memberId}/ban", requireUser(authService, http.HandlerFunc(admin.ban)))
	mux.Handle("DELETE /api/rooms/{code}/members/{memberId}/ban", requireUser(authService, http.HandlerFunc(admin.unban)))
	mux.Handle("DELETE /api/rooms/{code}/routes/{routeId}", requireUser(authService, http.HandlerFunc(admin.deleteRoute)))
	return mux
}
