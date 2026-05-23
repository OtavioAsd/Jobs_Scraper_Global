package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "scraper:"

func prefixed(key string) string {
	return keyPrefix + key
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string, target any) (bool, error) {
	val, err := r.client.Get(ctx, prefixed(key)).Result()

	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache.Get %q: %w", key, err)
	}

	if err := json.Unmarshal([]byte(val), target); err != nil {
		return false, fmt.Errorf("cache.Get %q: unmarshal: %w", key, err)
	}

	return true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache.Set %q: marshal: %w", key, err)
	}

	if err := r.client.Set(ctx, prefixed(key), payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache.Set %q: %w", key, err)
	}

	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, prefixed(key)).Err(); err != nil {
		return fmt.Errorf("cache.Delete %q: %w", key, err)
	}
	return nil
}

func (r *RedisCache) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("cache.SetNX %q: marshal: %w", key, err)
	}

	acquired, err := r.client.SetNX(ctx, prefixed(key), payload, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("cache.SetNX %q: %w", key, err)
	}

	return acquired, nil
}

func (r *RedisCache) Status() Status {
	return Status{Provider: "valkey"}
}
