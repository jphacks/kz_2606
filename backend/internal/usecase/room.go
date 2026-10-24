package usecase

import (
	"context"
	"errors"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

const maxRoomCodeAttempts = 5

type RoomRepository interface {
	Create(ctx context.Context, owner domain.User, code, title string) (domain.Room, domain.Member, error)
	FindByCode(ctx context.Context, code string) (domain.Room, error)
	Join(ctx context.Context, code string, userID int64, displayName, color string) (domain.Room, domain.Member, error)
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
	if err := domain.ValidateDisplayName(displayName); err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	normalizedColor, err := domain.NormalizeColor(color)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	return s.rooms.Join(ctx, normalizedCode, userID, displayName, normalizedColor)
}
