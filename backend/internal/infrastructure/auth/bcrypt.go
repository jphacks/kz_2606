package auth

import (
	"errors"

	"github.com/jphacks/kz_2606/backend/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

func (Hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (Hasher) Compare(passwordHash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return usecase.ErrPasswordMismatch
	}
	return err
}
