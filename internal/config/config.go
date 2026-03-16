package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	DatabasePath string
	TickInterval time.Duration
	LogLevel     string
	OpenRouter   OpenRouterConfig
	Telegram     TelegramConfig
}

type OpenRouterConfig struct {
	BaseURL       string
	APIKey        string
	PrimaryModel  string
	FallbackModel string
	SiteURL       string
	AppName       string
	Timeout       time.Duration
}

type TelegramConfig struct {
	BotToken string
	Mode     string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	tickInterval, err := durationFromEnv("TICK_INTERVAL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	openRouterTimeout, err := durationFromEnv("OPENROUTER_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = filepath.Join("data", "diaryhero.db")
	}

	return Config{
		AppEnv:       stringFromEnv("APP_ENV", "development"),
		DatabasePath: databasePath,
		TickInterval: tickInterval,
		LogLevel:     stringFromEnv("LOG_LEVEL", "info"),
		OpenRouter: OpenRouterConfig{
			BaseURL:       stringFromEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			APIKey:        os.Getenv("OPENROUTER_API_KEY"),
			PrimaryModel:  stringFromEnv("OPENROUTER_PRIMARY_MODEL", "openrouter/auto"),
			FallbackModel: stringFromEnv("OPENROUTER_FALLBACK_MODEL", "openrouter/auto"),
			SiteURL:       os.Getenv("OPENROUTER_SITE_URL"),
			AppName:       stringFromEnv("OPENROUTER_APP_NAME", "DiaryHero"),
			Timeout:       openRouterTimeout,
		},
		Telegram: TelegramConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
			Mode:     stringFromEnv("TELEGRAM_MODE", "polling"),
		},
	}, nil
}

func stringFromEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return parsed, nil
}
