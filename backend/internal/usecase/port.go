package usecase

import (
	"context"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, username, passwordHash string) (domain.User, error)
	FindByUsername(ctx context.Context, username string) (domain.User, string, error)
	FindByID(ctx context.Context, id int64) (domain.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type SessionCodec interface {
	Encode(userID int64) (string, error)
	Decode(token string) (int64, error)
}
