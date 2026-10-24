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

func NewRouter(authService *usecase.AuthService, cookies CookieOptions) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rootHandler)
	handler := &authHandler{service: authService, cookies: cookies}
	mux.HandleFunc("POST /api/auth/register", handler.register)
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.HandleFunc("GET /api/auth/me", handler.me)
	return mux
}
