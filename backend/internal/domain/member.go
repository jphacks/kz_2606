package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrMemberBanned       = errors.New("member is banned")
	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrInvalidColor       = errors.New("invalid color")
)

const FirstMemberColor = "#00e5ff"

var memberColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type Member struct {
	ID          int64
	DisplayName string
	Color       string
	BannedAt    *time.Time
}

func NormalizeDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if utf8.RuneCountInString(displayName) < 1 || utf8.RuneCountInString(displayName) > 16 {
		return "", ErrInvalidDisplayName
	}
	return displayName, nil
}

func NormalizeColor(color string) (string, error) {
	if !memberColorPattern.MatchString(color) {
		return "", ErrInvalidColor
	}
	return strings.ToLower(color), nil
}
