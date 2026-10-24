package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type roomHandler struct {
	rooms *usecase.RoomService
}

type createRoomRequest struct {
	Title string `json:"title"`
}

type joinRoomRequest struct {
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
}

type roomResponse struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Title         string     `json:"title"`
	OwnerMemberID int64      `json:"ownerMemberId"`
	ClosedAt      *time.Time `json:"closedAt"`
}

type memberResponse struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
}

func (h *roomHandler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	var input createRoomRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_format", "リクエスト形式が正しくありません")
		return
	}
	room, _, err := h.rooms.Create(r.Context(), user, input.Title)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRoomTitle) {
			writeAPIError(w, http.StatusBadRequest, "invalid_title", "タイトルは1〜40文字で入力してください")
			return
		}
		slog.Error("create room failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Room roomResponse `json:"room"`
	}{Room: toRoomResponse(room)})
}

func (h *roomHandler) get(w http.ResponseWriter, r *http.Request) {
	room, err := h.rooms.FindByCode(r.Context(), r.PathValue("code"))
	if err != nil {
		if errors.Is(err, domain.ErrRoomNotFound) || errors.Is(err, domain.ErrInvalidRoomCode) {
			writeAPIError(w, http.StatusNotFound, "room_not_found", "部屋が見つかりません")
			return
		}
		slog.Error("get room failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Room roomResponse `json:"room"`
	}{Room: toRoomResponse(room)})
}

func (h *roomHandler) join(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromRequest(w, r)
	if !ok {
		return
	}
	var input joinRoomRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_format", "リクエスト形式が正しくありません")
		return
	}
	room, member, err := h.rooms.Join(r.Context(), r.PathValue("code"), user.ID, input.DisplayName, input.Color)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidDisplayName):
			writeAPIError(w, http.StatusBadRequest, "invalid_display_name", "表示名は1〜16文字で入力してください")
		case errors.Is(err, domain.ErrInvalidColor):
			writeAPIError(w, http.StatusBadRequest, "invalid_color", "色は #rrggbb 形式で入力してください")
		case errors.Is(err, domain.ErrMemberBanned):
			writeAPIError(w, http.StatusForbidden, "banned", "この部屋への参加は禁止されています")
		case errors.Is(err, domain.ErrRoomNotFound), errors.Is(err, domain.ErrInvalidRoomCode):
			writeAPIError(w, http.StatusNotFound, "room_not_found", "部屋が見つかりません")
		default:
			slog.Error("join room failed", "err", err)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました")
		}
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Member memberResponse `json:"member"`
		Room   roomResponse   `json:"room"`
	}{
		Member: toMemberResponse(member),
		Room:   toRoomResponse(room),
	})
}

func toRoomResponse(room domain.Room) roomResponse {
	return roomResponse{
		ID:            room.ID,
		Code:          room.Code,
		Title:         room.Title,
		OwnerMemberID: room.OwnerMemberID,
		ClosedAt:      room.ClosedAt,
	}
}

func toMemberResponse(member domain.Member) memberResponse {
	return memberResponse{
		ID:          member.ID,
		DisplayName: member.DisplayName,
		Color:       member.Color,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response failed", "err", err)
	}
}
