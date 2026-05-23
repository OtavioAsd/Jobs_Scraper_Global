package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string, target any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error

	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

	Status() Status
}
