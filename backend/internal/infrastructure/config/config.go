package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

const defaultDatabaseURL = "postgres://gpsart:gpsart@localhost:5432/gpsart?sslmode=disable"

type Config struct {
	Port                   string
	DatabaseURL            string
	SessionSecret          string
	SessionSecretEphemeral bool
}

func Load() (Config, error) {
	cfg := Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(env("DATABASE_URL", defaultDatabaseURL)),
	}
	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
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
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
