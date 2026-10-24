package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type adminHandler struct {
	rooms *usecase.RoomService
}

func (h *adminHandler) close(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	closedAt, err := h.rooms.Close(r.Context(), r.PathValue("code"), user.ID)
	if err != nil {
		h.writeRoomError(w, err, "close room")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		ClosedAt time.Time `json:"closedAt"`
	}{ClosedAt: closedAt})
}

func (h *adminHandler) reopen(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	if err := h.rooms.Reopen(r.Context(), r.PathValue("code"), user.ID); err != nil {
		h.writeRoomError(w, err, "reopen room")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *adminHandler) ban(w http.ResponseWriter, r *http.Request) {
	h.setBan(w, r, true)
}

func (h *adminHandler) unban(w http.ResponseWriter, r *http.Request) {
	h.setBan(w, r, false)
}

func (h *adminHandler) setBan(w http.ResponseWriter, r *http.Request, banned bool) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	memberID, err := strconv.ParseInt(r.PathValue("memberId"), 10, 64)
	if err != nil || memberID <= 0 {
		writeAPIError(w, http.StatusNotFound, "member_not_found", "メンバーが見つかりません")
		return
	}
	if banned {
		err = h.rooms.Ban(r.Context(), r.PathValue("code"), user.ID, memberID)
	} else {
		err = h.rooms.Unban(r.Context(), r.PathValue("code"), user.ID, memberID)
	}
	if err != nil {
		h.writeMemberError(w, err, "change member ban status")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *adminHandler) deleteRoute(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	routeID, err := strconv.ParseInt(r.PathValue("routeId"), 10, 64)
	if err != nil || routeID <= 0 {
		writeAPIError(w, http.StatusNotFound, "route_not_found", "記録が見つかりません")
		return
	}
	if err := h.rooms.DeleteRoute(r.Context(), r.PathValue("code"), user.ID, routeID); err != nil {
		h.writeRouteError(w, err, "delete route")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *adminHandler) writeRoomError(w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", "この操作を行う権限がありません")
	case errors.Is(err, domain.ErrInvalidRoomCode), errors.Is(err, domain.ErrRoomNotFound):
		writeAPIError(w, http.StatusNotFound, "room_not_found", "部屋が見つかりません")
	default:
		slog.Error(operation+" failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
	}
}

func (h *adminHandler) writeMemberError(w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", "この操作を行う権限がありません")
	case errors.Is(err, domain.ErrCannotBanOwner):
		writeAPIError(w, http.StatusBadRequest, "cannot_ban_owner", "部屋の作成者はBANできません")
	case errors.Is(err, domain.ErrMemberNotFound):
		writeAPIError(w, http.StatusNotFound, "member_not_found", "メンバーが見つかりません")
	case errors.Is(err, domain.ErrRoomNotFound), errors.Is(err, domain.ErrInvalidRoomCode):
		writeAPIError(w, http.StatusNotFound, "room_not_found", "部屋が見つかりません")
	default:
		slog.Error(operation+" failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
	}
}

func (h *adminHandler) writeRouteError(w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, "forbidden", "この操作を行う権限がありません")
	case errors.Is(err, domain.ErrRouteNotFound):
		writeAPIError(w, http.StatusNotFound, "route_not_found", "記録が見つかりません")
	case errors.Is(err, domain.ErrRoomNotFound), errors.Is(err, domain.ErrInvalidRoomCode):
		writeAPIError(w, http.StatusNotFound, "room_not_found", "部屋が見つかりません")
	default:
		slog.Error(operation+" failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
	}
}
