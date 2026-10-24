package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const SessionLifetime = 7 * 24 * time.Hour

var ErrInvalidSession = errors.New("invalid session")

type SessionCodec struct {
	secret []byte
}

func NewSessionCodec(secret string) (*SessionCodec, error) {
	if len(secret) < 32 {
		return nil, errors.New("session secret must be at least 32 bytes")
	}
	return &SessionCodec{secret: []byte(secret)}, nil
}

func (c *SessionCodec) Encode(userID int64) (string, error) {
	if userID <= 0 {
		return "", errors.New("user ID must be positive")
	}
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%d:%d", userID, time.Now().Add(SessionLifetime).Unix())),
	)
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func (c *SessionCodec) Decode(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, ErrInvalidSession
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrInvalidSession
	}
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, ErrInvalidSession
	}
	values := strings.Split(string(payload), ":")
	if len(values) != 2 {
		return 0, ErrInvalidSession
	}
	userID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil || userID <= 0 {
		return 0, ErrInvalidSession
	}
	expiresAt, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil || time.Now().Unix() >= expiresAt {
		return 0, ErrInvalidSession
	}
	return userID, nil
}
