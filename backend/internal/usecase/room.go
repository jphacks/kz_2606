package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

const maxRoomCodeAttempts = 5

type RoomRepository interface {
	Create(ctx context.Context, owner domain.User, code, title string) (domain.Room, domain.Member, error)
	FindByCode(ctx context.Context, code string) (domain.Room, error)
	Join(ctx context.Context, code string, userID int64, displayName, color string) (domain.Room, domain.Member, error)
	Snapshot(ctx context.Context, code string, userID int64) (domain.RoomSnapshot, domain.Member, error)
	Close(ctx context.Context, code string, userID int64) (time.Time, error)
	Reopen(ctx context.Context, code string, userID int64) error
	Ban(ctx context.Context, code string, userID, memberID int64) error
	Unban(ctx context.Context, code string, userID, memberID int64) error
	DeleteRoute(ctx context.Context, code string, userID, routeID int64) error
}

type RoomService struct {
	rooms RoomRepository
}

func NewRoomService(rooms RoomRepository) *RoomService {
	return &RoomService{rooms: rooms}
}

func (s *RoomService) Create(ctx context.Context, owner domain.User, title string) (domain.Room, domain.Member, error) {
	normalizedTitle, err := domain.NormalizeRoomTitle(title)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	for range maxRoomCodeAttempts {
		code, err := domain.NewRoomCode()
		if err != nil {
			return domain.Room{}, domain.Member{}, err
		}
		room, ownerMember, err := s.rooms.Create(ctx, owner, code, normalizedTitle)
		if errors.Is(err, domain.ErrRoomCodeTaken) {
			continue
		}
		return room, ownerMember, err
	}
	return domain.Room{}, domain.Member{}, errors.New("could not allocate a unique room code")
}

func (s *RoomService) FindByCode(ctx context.Context, code string) (domain.Room, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Room{}, err
	}
	return s.rooms.FindByCode(ctx, normalizedCode)
}

func (s *RoomService) Join(ctx context.Context, code string, userID int64, displayName, color string) (domain.Room, domain.Member, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	normalizedDisplayName, err := domain.NormalizeDisplayName(displayName)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	normalizedColor, err := domain.NormalizeColor(color)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	return s.rooms.Join(ctx, normalizedCode, userID, normalizedDisplayName, normalizedColor)
}

func (s *RoomService) Snapshot(ctx context.Context, code string, userID int64) (domain.RoomSnapshot, domain.Member, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	return s.rooms.Snapshot(ctx, normalizedCode, userID)
}

func (s *RoomService) Close(ctx context.Context, code string, userID int64) (time.Time, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return time.Time{}, err
	}
	return s.rooms.Close(ctx, normalizedCode, userID)
}

func (s *RoomService) Reopen(ctx context.Context, code string, userID int64) error {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return err
	}
	return s.rooms.Reopen(ctx, normalizedCode, userID)
}

func (s *RoomService) Ban(ctx context.Context, code string, userID, memberID int64) error {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return err
	}
	if memberID <= 0 {
		return domain.ErrMemberNotFound
	}
	return s.rooms.Ban(ctx, normalizedCode, userID, memberID)
}

func (s *RoomService) Unban(ctx context.Context, code string, userID, memberID int64) error {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return err
	}
	if memberID <= 0 {
		return domain.ErrMemberNotFound
	}
	return s.rooms.Unban(ctx, normalizedCode, userID, memberID)
}

func (s *RoomService) DeleteRoute(ctx context.Context, code string, userID, routeID int64) error {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return err
	}
	if routeID <= 0 {
		return domain.ErrRouteNotFound
	}
	return s.rooms.DeleteRoute(ctx, normalizedCode, userID, routeID)
}
