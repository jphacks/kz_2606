package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrUserNotFound    = errors.New("user not found")
)

type User struct {
	ID        int64
	Username  string
	CreatedAt time.Time
}

func ParseUsername(value string) (string, error) {
	if len(value) < 3 || len(value) > 20 {
		return "", ErrInvalidUsername
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return "", ErrInvalidUsername
		}
	}
	return strings.ToLower(value), nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrInvalidPassword
	}
	return nil
}
