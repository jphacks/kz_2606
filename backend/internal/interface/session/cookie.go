package session

import (
	"context"
	"net/http"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

// CookieName はRESTとWebSocketで共有するセッションCookie名。
const CookieName = "session"

// AuthenticateRequest はリクエストCookieのセッションを検証し、対応するユーザーを返す。
func AuthenticateRequest(ctx context.Context, r *http.Request, authService *usecase.AuthService) (domain.User, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return domain.User{}, usecase.ErrUnauthorized
	}
	return authService.Me(ctx, cookie.Value)
}
