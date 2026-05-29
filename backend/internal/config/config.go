package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort       string
	BaseURL       string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	URLTTL        time.Duration
}

func Load() Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	ttlMinutes, _ := strconv.Atoi(getEnv("URL_TTL_MINUTES", "10"))

	return Config{
		AppPort:       getEnv("APP_PORT", "8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://app:app@localhost:5432/url_shortener?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
		URLTTL:        time.Duration(ttlMinutes) * time.Minute,
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
