package rest

import (
	"net/http"
	"time"

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

func NewRouter(authService *usecase.AuthService, roomService *usecase.RoomService, cookies CookieOptions) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rootHandler)
	handler := &authHandler{service: authService, cookies: cookies}
	rooms := &roomHandler{auth: authService, rooms: roomService}
	admin := &adminHandler{auth: authService, rooms: roomService}
	mux.HandleFunc("POST /api/auth/register", handler.register)
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.HandleFunc("GET /api/auth/me", handler.me)
	mux.HandleFunc("POST /api/rooms", rooms.create)
	mux.HandleFunc("GET /api/rooms/{code}", rooms.get)
	mux.HandleFunc("POST /api/rooms/{code}/join", rooms.join)
	mux.HandleFunc("POST /api/rooms/{code}/close", admin.close)
	mux.HandleFunc("POST /api/rooms/{code}/reopen", admin.reopen)
	mux.HandleFunc("POST /api/rooms/{code}/members/{memberId}/ban", admin.ban)
	mux.HandleFunc("DELETE /api/rooms/{code}/members/{memberId}/ban", admin.unban)
	mux.HandleFunc("DELETE /api/rooms/{code}/routes/{routeId}", admin.deleteRoute)
	return mux
}
