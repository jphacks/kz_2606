package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultDatabaseURL = "postgres://gpsart:gpsart@localhost:5432/gpsart?sslmode=disable"

type Config struct {
	Port                   string
	DatabaseURL            string
	SessionSecret          string
	SessionSecretEphemeral bool
	CookieSecure           bool
}

func Load() (Config, error) {
	cfg := Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(env("DATABASE_URL", defaultDatabaseURL)),
	}
	if secret := strings.TrimSpace(os.Getenv("SESSION_SECRET")); secret != "" {
		cfg.SessionSecret = secret
	} else {
		randomSecret := make([]byte, 32)
		if _, err := rand.Read(randomSecret); err != nil {
			return Config{}, errors.New("generate temporary session secret: " + err.Error())
		}
		cfg.SessionSecret = base64.RawURLEncoding.EncodeToString(randomSecret)
		cfg.SessionSecretEphemeral = true
	}
	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("SESSION_SECRET must be at least 32 bytes")
	}
	// HTTP の開発サーバー（スマホ実機を LAN でつなぐ場合など）では Secure Cookie が
	// ブラウザに捨てられてログインできないため、明示的に false にできるようにする。
	cookieSecure, err := envBool("COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	cfg.CookieSecure = cookieSecure
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New(key + " must be a boolean (true/false)")
	}
	return parsed, nil
}
