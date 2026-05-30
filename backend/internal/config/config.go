package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort     string
	BaseURL     string
	DatabaseURL string
	RedisURL    string
	URLTTL      time.Duration
}

func Load() Config {
	ttlMinutes, _ := strconv.Atoi(getEnv("URL_TTL_MINUTES", "10"))

	return Config{
		AppPort:     getEnv("PORT", getEnv("APP_PORT", "8080")),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://app:app@localhost:5432/url_shortener?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		URLTTL:      time.Duration(ttlMinutes) * time.Minute,
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
