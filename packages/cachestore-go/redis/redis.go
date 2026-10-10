// Package cacheredis is the Redis implementation of cachestore.CacheStore, kept in its own module
// so a service that never uses Redis never depends on a Redis client (and its transitive
// dependencies never reach that service's build or bundle).
package cacheredis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	cachestore "github.com/zenius-one/cachestore/packages/cachestore-go"
)

// RedisStore is a CacheStore backed by Redis, shared by every replica.
type RedisStore struct {
	client redis.Cmdable
	prefix string
}

var _ cachestore.CacheStore = (*RedisStore)(nil)

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
		return cachestore.ErrInvalidTTL
	}
	return r.client.Set(ctx, r.key(key), value, ttl).Err()
}

// SetNX implements CacheStore.
func (r *RedisStore) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, cachestore.ErrInvalidTTL
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

// New returns the CacheStore to use: a Redis-backed cachestore.Fallback over a MemoryStore when
// client is non-nil, otherwise the MemoryStore alone. prefix namespaces the Redis keys. Degraded
// means "less sharing between replicas", never "wrong" and never "unavailable".
func New(client redis.Cmdable, prefix string, mem cachestore.MemoryOptions, opts ...cachestore.FallbackOptions) cachestore.CacheStore {
	m := cachestore.NewMemoryStore(mem)
	if client == nil {
		return m
	}
	var fo cachestore.FallbackOptions
	if len(opts) > 0 {
		fo = opts[0]
	}
	return cachestore.NewFallback(NewRedisStore(client, prefix), m, fo)
}
