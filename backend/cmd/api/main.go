package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/bell/go-url-shortener/backend/internal/adapters/cache"
	httpadapter "github.com/bell/go-url-shortener/backend/internal/adapters/http"
	"github.com/bell/go-url-shortener/backend/internal/adapters/postgres"
	"github.com/bell/go-url-shortener/backend/internal/application"
	"github.com/bell/go-url-shortener/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	urlRepo := postgres.NewURLRepository(dbpool)
	urlCache := cache.NewRedisURLCache(redisClient)

	service := application.NewURLService(urlRepo, urlCache, cfg.BaseURL, cfg.URLTTL)

	router := httpadapter.NewRouter(service)

	addr := ":" + cfg.AppPort
	log.Printf("server started on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
