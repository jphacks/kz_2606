package domain

import "errors"

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrInvalidRoomCode  = errors.New("invalid room code")
	ErrInvalidRoomTitle = errors.New("invalid room title")
	ErrMemberNotFound   = errors.New("member not found")
)
