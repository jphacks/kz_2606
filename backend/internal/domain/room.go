package domain

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrRoomCodeTaken = errors.New("room code already taken")

const roomCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Room struct {
	ID            int64
	Code          string
	Title         string
	OwnerMemberID int64
	ClosedAt      *time.Time
}

func NormalizeRoomTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 40 {
		return "", ErrInvalidRoomTitle
	}
	return title, nil
}

func NormalizeRoomCode(code string) (string, error) {
	code = strings.ToUpper(code)
	if len(code) != 6 {
		return "", ErrInvalidRoomCode
	}
	for _, char := range code {
		if !strings.ContainsRune(roomCodeAlphabet, char) {
			return "", ErrInvalidRoomCode
		}
	}
	return code, nil
}

func NewRoomCode() (string, error) {
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	code := make([]byte, len(randomBytes))
	for i, value := range randomBytes {
		code[i] = roomCodeAlphabet[value&31]
	}
	return string(code), nil
}
