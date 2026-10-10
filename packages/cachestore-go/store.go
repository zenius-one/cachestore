package cachestore

import (
	"context"
	"time"
)

// CacheStore is a TTL key-value cache of opaque values.
//
// A miss is (nil, false, nil); an error means the store could not answer, which callers should
// treat as a miss and keep going (a cache must never be the reason a request fails).
type CacheStore interface {
	// Get returns the live value for key.
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	// Set stores value under key for ttl. A ttl <= 0 is rejected with ErrInvalidTTL: an entry that
	// never expires is never what a cache wants.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// SetNX stores value only if key is absent (or expired) and reports whether it did. It is the
	// primitive for leases and request collapsing across replicas.
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (stored bool, err error)
	// Delete removes keys; missing keys are not an error.
	Delete(ctx context.Context, keys ...string) error
	// Close releases the store's resources. It does not close a Redis client it was handed.
	Close() error
}
