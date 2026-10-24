package usecase

import (
	"context"
	"errors"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

const maxPersonalRoomCodeAttempts = 5

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPasswordMismatch   = errors.New("password mismatch")
	ErrUnauthorized       = errors.New("unauthorized")
)

type AuthResult struct {
	User  domain.User
	Room  domain.Room
	Token string
}

type AuthService struct {
	users    UserRepository
	password PasswordHasher
	sessions SessionCodec
}

func NewAuthService(users UserRepository, password PasswordHasher, sessions SessionCodec) *AuthService {
	return &AuthService{users: users, password: password, sessions: sessions}
}

func (s *AuthService) Register(ctx context.Context, username, password string) (AuthResult, error) {
	normalizedUsername, err := domain.ParseUsername(username)
	if err != nil {
		return AuthResult{}, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := s.password.Hash(password)
	if err != nil {
		return AuthResult{}, err
	}
	title := normalizedUsername + "のルーム"
	var user domain.User
	var room domain.Room
	created := false
	for range maxPersonalRoomCodeAttempts {
		code, err := domain.NewRoomCode()
		if err != nil {
			return AuthResult{}, err
		}
		user, room, _, err = s.users.CreateWithPersonalRoom(ctx, normalizedUsername, passwordHash, code, title)
		if errors.Is(err, domain.ErrRoomCodeTaken) {
			continue
		}
		if err != nil {
			return AuthResult{}, err
		}
		created = true
		break
	}
	if !created {
		return AuthResult{}, errors.New("could not allocate a unique room code")
	}
	token, err := s.sessions.Encode(user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Room: room, Token: token}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (AuthResult, error) {
	normalizedUsername, err := domain.ParseUsername(username)
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err := domain.ValidatePassword(password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	user, passwordHash, err := s.users.FindByUsername(ctx, normalizedUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// ユーザーが居なくてもハッシュ比較のコストを払う。
			// これを省くと「不在=即 401 / パスワード違い=bcrypt 1回分」の
			// 応答時間差でユーザー名を列挙できてしまう。
			_ = s.password.Compare(s.password.DummyHash(), password)
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	if err := s.password.Compare(passwordHash, password); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	token, err := s.sessions.Encode(user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	room, err := s.users.FindPersonalRoom(ctx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Room: room, Token: token}, nil
}

func (s *AuthService) Me(ctx context.Context, token string) (domain.User, error) {
	userID, err := s.sessions.Decode(token)
	if err != nil {
		return domain.User{}, ErrUnauthorized
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, ErrUnauthorized
		}
		return domain.User{}, err
	}
	return user, nil
}

func (s *AuthService) PersonalRoom(ctx context.Context, ownerUserID int64) (domain.Room, error) {
	return s.users.FindPersonalRoom(ctx, ownerUserID)
}
