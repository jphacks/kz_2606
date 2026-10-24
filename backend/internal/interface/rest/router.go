package rest

import (
	"net/http"

	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

// ServeMux の組み立て。
// 仕様: DATA_MODEL.md 5章

func NewRouter(authService *usecase.AuthService) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rootHandler)
	handler := &authHandler{service: authService}
	mux.HandleFunc("POST /api/auth/register", handler.register)
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.HandleFunc("GET /api/auth/me", handler.me)
	return mux
}
