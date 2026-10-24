package usecase

import (
	"context"
	"testing"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

type recordingHasher struct {
	compares int
}

func (h *recordingHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (h *recordingHasher) Compare(passwordHash, password string) error {
	h.compares++
	if passwordHash != "hash:"+password {
		return ErrPasswordMismatch
	}
	return nil
}

func (h *recordingHasher) DummyHash() string { return "hash:" }

type emptyUsers struct{}

func (emptyUsers) CreateWithPersonalRoom(context.Context, string, string, string, string) (domain.User, domain.Room, domain.Member, error) {
	return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrUsernameTaken
}

func (emptyUsers) FindByUsername(context.Context, string) (domain.User, string, error) {
	return domain.User{}, "", domain.ErrUserNotFound
}

func (emptyUsers) FindByID(context.Context, int64) (domain.User, error) {
	return domain.User{}, domain.ErrUserNotFound
}

type stubSessions struct{}

func (stubSessions) Encode(int64) (string, error) { return "token", nil }
func (stubSessions) Decode(string) (int64, error) { return 1, nil }

// 存在しないユーザー名でもハッシュ比較が走ること（応答時間での列挙対策）。
func TestLoginHashesEvenWhenUserIsMissing(t *testing.T) {
	hasher := &recordingHasher{}
	service := NewAuthService(emptyUsers{}, hasher, stubSessions{})
	if _, err := service.Login(context.Background(), "missing", "password123"); err != ErrInvalidCredentials {
		t.Fatalf("Login err = %v", err)
	}
	if hasher.compares != 1 {
		t.Fatalf("Compare calls = %d, want 1", hasher.compares)
	}
}
