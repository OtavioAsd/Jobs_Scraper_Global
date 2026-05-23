package cache

import "context"

func GetAs[T any](c Cache, ctx context.Context, key string) (T, bool, error) {
	var target T

	found, err := c.Get(ctx, key, &target)
	if err != nil {
		var zero T
		return zero, false, err
	}

	return target, found, nil
}
