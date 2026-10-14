package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port          string
	LogLevel      string
	DatabaseURL   string
	CamaraBaseURL string
	CamaraTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:          getenv("PORT", "8080"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		CamaraBaseURL: getenv("CAMARA_BASE_URL", "https://dadosabertos.camara.leg.br/api/v2"),
	}

	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL invalido: %q", cfg.LogLevel)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL e obrigatoria")
	}

	timeout, err := time.ParseDuration(getenv("CAMARA_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("CAMARA_TIMEOUT invalido: %w", err)
	}
	cfg.CamaraTimeout = timeout

	return cfg, nil
}

func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
