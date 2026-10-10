package cachestore

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a CacheStore backed by Redis, shared by every replica.
type RedisStore struct {
	client redis.Cmdable
	prefix string
}

var _ CacheStore = (*RedisStore)(nil)

// NewRedisStore returns a RedisStore using client. Every key is stored under prefix, so several
// users of one Redis cannot collide and a prefix can be flushed on its own. The client is owned by
// the caller; Close does not close it.
func NewRedisStore(client redis.Cmdable, prefix string) *RedisStore {
	return &RedisStore{client: client, prefix: prefix}
}

func (r *RedisStore) key(k string) string { return r.prefix + k }

// Get implements CacheStore.
func (r *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := r.client.Get(ctx, r.key(key)).Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return v, true, nil
}

// Set implements CacheStore.
func (r *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return r.client.Set(ctx, r.key(key), value, ttl).Err()
}

// SetNX implements CacheStore.
func (r *RedisStore) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	return r.client.SetNX(ctx, r.key(key), value, ttl).Result()
}

// Delete implements CacheStore.
func (r *RedisStore) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = r.key(k)
	}
	return r.client.Del(ctx, full...).Err()
}

// Close implements CacheStore. The client belongs to the caller.
func (r *RedisStore) Close() error { return nil }
