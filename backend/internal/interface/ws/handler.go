package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/interface/session"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type roomSnapshotter interface {
	Snapshot(context.Context, string, int64) (domain.RoomSnapshot, domain.Member, error)
}

// Handler はCookieで接続者を認証し、部屋の参加状態を確認してから初期snapshotを送る。
type Handler struct {
	auth     *usecase.AuthService
	rooms    roomSnapshotter
	registry *Registry
	upgrader websocket.Upgrader
}

func NewHandler(authService *usecase.AuthService, rooms roomSnapshotter, registry *Registry) *Handler {
	return &Handler{
		auth:     authService,
		rooms:    rooms,
		registry: registry,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := session.AuthenticateRequest(r.Context(), r, h.auth)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			http.Error(w, "ログインが必要です", http.StatusUnauthorized)
			return
		}
		slog.Error("authenticate websocket request failed", "err", err)
		http.Error(w, "サーバーエラーが発生しました", http.StatusInternalServerError)
		return
	}

	roomCode := r.URL.Query().Get("room")
	if roomCode == "" {
		http.Error(w, "room が必要です", http.StatusBadRequest)
		return
	}
	snapshot, you, err := h.rooms.Snapshot(r.Context(), roomCode, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrMemberNotFound), errors.Is(err, domain.ErrMemberBanned):
			http.Error(w, "この部屋に接続する権限がありません", http.StatusForbidden)
		case errors.Is(err, domain.ErrInvalidRoomCode), errors.Is(err, domain.ErrRoomNotFound):
			http.Error(w, "部屋が見つかりません", http.StatusNotFound)
		default:
			slog.Error("load websocket snapshot failed", "err", err)
			http.Error(w, "サーバーエラーが発生しました", http.StatusInternalServerError)
		}
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Debug("websocket upgrade failed", "err", err)
		return
	}
	connection := &client{
		conn:     conn,
		out:      make(chan []byte, sendBufferSize),
		done:     make(chan struct{}),
		registry: h.registry,
		roomCode: snapshot.Room.Code,
		memberID: you.ID,
	}
	h.registry.Add(connection)
	payload, err := json.Marshal(makeSnapshotDTO(snapshot, you, h.registry))
	if err != nil {
		slog.Error("encode websocket snapshot failed", "err", err)
		connection.shutdown()
		return
	}
	go connection.writePump(payload)
	connection.readPump()
}
