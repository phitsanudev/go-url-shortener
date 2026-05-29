package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisURLCache struct {
	client *redis.Client
}

func NewRedisURLCache(client *redis.Client) *RedisURLCache {
	return &RedisURLCache{client: client}
}

func (c *RedisURLCache) Set(ctx context.Context, code string, originalURL string, ttl time.Duration) error {
	return c.client.Set(ctx, cacheKey(code), originalURL, ttl).Err()
}

func (c *RedisURLCache) Get(ctx context.Context, code string) (string, error) {
	value, err := c.client.Get(ctx, cacheKey(code)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

func (c *RedisURLCache) Delete(ctx context.Context, code string) error {
	return c.client.Del(ctx, cacheKey(code)).Err()
}

func cacheKey(code string) string {
	return "short_url:" + code
}
