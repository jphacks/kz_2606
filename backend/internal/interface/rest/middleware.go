package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type userContextKey struct{}

func requireUser(authService *usecase.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
			return
		}
		user, err := authService.Me(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, usecase.ErrUnauthorized) {
				writeAPIError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
				return
			}
			slog.Error("authenticate request failed", "err", err)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFrom(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(domain.User)
	return user, ok
}

func userFromRequest(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	user, ok := userFrom(r.Context())
	if !ok {
		slog.Error("authenticated user missing from request context")
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		return domain.User{}, false
	}
	return user, true
}
