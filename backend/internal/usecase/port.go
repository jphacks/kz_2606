package usecase

import (
	"context"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

type UserRepository interface {
	CreateWithPersonalRoom(ctx context.Context, username, passwordHash, roomCode, roomTitle string) (domain.User, domain.Room, domain.Member, error)
	FindByUsername(ctx context.Context, username string) (domain.User, string, error)
	FindByID(ctx context.Context, id int64) (domain.User, error)
	FindPersonalRoom(ctx context.Context, ownerUserID int64) (domain.Room, error)
	CreatePersonalRoom(ctx context.Context, owner domain.User, roomCode, roomTitle string) (domain.Room, domain.Member, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
	// DummyHash は存在しないユーザー名でも Compare を走らせるためのダミーハッシュ。
	// 応答時間の差からユーザー名を列挙されるのを防ぐ（DATA_MODEL.md 5章）。
	DummyHash() string
}

type SessionCodec interface {
	Encode(userID int64) (string, error)
	Decode(token string) (int64, error)
}

type SessionRepository interface {
	StartRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, domain.Route, error)
	AppendPoints(ctx context.Context, code string, memberID int64, points []domain.Point) (int64, domain.Member, error)
	StopRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, int64, error)
	UpdateMember(ctx context.Context, code string, memberID int64, displayName, color *string) (domain.Member, bool, error)
	Leave(ctx context.Context, code string, memberID int64) (domain.Member, int64, bool, error)
}
