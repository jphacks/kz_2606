package domain

import "errors"

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrInvalidRoomCode  = errors.New("invalid room code")
	ErrInvalidRoomTitle = errors.New("invalid room title")
	ErrMemberNotFound   = errors.New("member not found")
	ErrRouteNotFound    = errors.New("route not found")
	ErrForbidden        = errors.New("forbidden")
	ErrCannotBanOwner   = errors.New("cannot ban room owner")
	ErrPrivateRoom      = errors.New("private room")
	ErrRoomClosed       = errors.New("room is closed")
	ErrNoActiveRoute    = errors.New("member has no active route")
	ErrAlreadyRecording = errors.New("member already has an active route")
	ErrInvalidPoints    = errors.New("invalid points")
)
