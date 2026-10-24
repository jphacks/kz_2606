package auth

import (
	"errors"
	"sync"

	"github.com/jphacks/kz_2606/backend/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

// 初回の DummyHash() で一度だけ生成する。Compare のコストを本物と揃えるため、
// cost は Hash と同じ bcrypt.DefaultCost でなければならない。
var dummyHash = sync.OnceValue(func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-constant-time-login"), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
})

func (Hasher) DummyHash() string { return dummyHash() }

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
