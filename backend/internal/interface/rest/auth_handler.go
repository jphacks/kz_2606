package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

const sessionCookieName = "session"

type authHandler struct {
	service *usecase.AuthService
	cookies CookieOptions
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type apiUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if err := decodeJSON(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_format", "リクエスト形式が正しくありません")
		return
	}
	result, err := h.service.Register(r.Context(), input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidUsername):
			writeAPIError(w, http.StatusBadRequest, "invalid_username", "ユーザー名は3〜16文字の英数字と _ で入力してください")
		case errors.Is(err, domain.ErrInvalidPassword):
			writeAPIError(w, http.StatusBadRequest, "invalid_password", "パスワードは8〜72バイトで入力してください")
		case errors.Is(err, domain.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "そのユーザー名は使われています")
		default:
			slog.Error("register failed", "err", err)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		}
		return
	}
	h.setSessionCookie(w, result.Token)
	writeUser(w, http.StatusOK, result.User)
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if err := decodeJSON(w, r, &input); err != nil || input.Username == "" || input.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_format", "リクエスト形式が正しくありません")
		return
	}
	result, err := h.service.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "ユーザー名またはパスワードが正しくありません")
			return
		}
		slog.Error("login failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		return
	}
	h.setSessionCookie(w, result.Token)
	writeUser(w, http.StatusOK, result.User)
}

func (h *authHandler) logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}
	user, err := h.service.Me(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
			return
		}
		slog.Error("load current user failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		return
	}
	writeUser(w, http.StatusOK, user)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	// Content-Type を application/json に限定する。text/plain などを許すと
	// CORS の単純リクエストになり、プリフライト無しで他サイトから POST できてしまう
	// （ログインCSRF: 攻撃者のアカウントにログインさせられる）。
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain exactly one JSON value")
	}
	return nil
}

func (h *authHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.cookies.Lifetime.Seconds()),
	})
}

func writeUser(w http.ResponseWriter, status int, user domain.User) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(struct {
		User apiUser `json:"user"`
	}{User: apiUser{ID: user.ID, Username: user.Username}}); err != nil {
		slog.Error("write auth response failed", "err", err)
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	var response apiError
	response.Error.Code = code
	response.Error.Message = message
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("write API error response failed", "err", err)
	}
}
