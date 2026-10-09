package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Port     string
	LogLevel string
}

func Load() (Config, error) {
	cfg := Config{
		Port:     getenv("PORT", "8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),
	}

	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL invalido: %q", cfg.LogLevel)
	}

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
