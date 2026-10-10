package cachestore

import (
	"context"
	"encoding/json"
	"time"
)

// GetJSON reads key and decodes it into T. A value that no longer decodes (e.g. after a shape
// change between deploys) is reported as a miss, not an error.
func GetJSON[T any](ctx context.Context, s CacheStore, key string) (v T, found bool, err error) {
	raw, ok, err := s.Get(ctx, key)
	if err != nil || !ok {
		return v, false, err
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		var zero T
		return zero, false, nil
	}
	return v, true, nil
}

// SetJSON encodes v and stores it under key for ttl.
func SetJSON[T any](ctx context.Context, s CacheStore, key string, v T, ttl time.Duration) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Set(ctx, key, raw, ttl)
}
