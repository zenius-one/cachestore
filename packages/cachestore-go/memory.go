package cachestore

import (
	"context"
	"sync"
	"time"
)

// MemoryOptions configures a MemoryStore.
type MemoryOptions struct {
	// MaxEntries bounds the store; 0 means 10000. When full, expired entries are dropped first and
	// then the entry closest to expiry, so a flood of distinct keys cannot grow memory.
	MaxEntries int
	// Now is overridable for tests.
	Now func() time.Time
}

type memoryEntry struct {
	value   []byte
	expires time.Time
}

// MemoryStore is a bounded in-process CacheStore. It is per-process: replicas do not share it.
// Safe for concurrent use.
type MemoryStore struct {
	max int
	now func() time.Time

	mu      sync.Mutex
	entries map[string]memoryEntry
}

var _ CacheStore = (*MemoryStore)(nil)

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore(opts MemoryOptions) *MemoryStore {
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 10000
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &MemoryStore{max: opts.MaxEntries, now: opts.Now, entries: make(map[string]memoryEntry)}
}

func (m *MemoryStore) live(key string, now time.Time) (memoryEntry, bool) {
	e, ok := m.entries[key]
	if !ok {
		return memoryEntry{}, false
	}
	if !now.Before(e.expires) {
		delete(m.entries, key)
		return memoryEntry{}, false
	}
	return e, true
}

// Get implements CacheStore. The returned slice is a copy.
func (m *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.live(key, m.now())
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), e.value...), true, nil
}

// Set implements CacheStore. The value is copied.
func (m *MemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.put(key, value, ttl)
	return nil
}

// SetNX implements CacheStore.
func (m *MemoryStore) SetNX(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.live(key, m.now()); ok {
		return false, nil
	}
	m.put(key, value, ttl)
	return true, nil
}

// put requires m.mu held.
func (m *MemoryStore) put(key string, value []byte, ttl time.Duration) {
	now := m.now()
	if _, exists := m.entries[key]; !exists && len(m.entries) >= m.max {
		m.evict(now)
	}
	m.entries[key] = memoryEntry{value: append([]byte(nil), value...), expires: now.Add(ttl)}
}

// evict makes room for one entry. m.mu must be held.
func (m *MemoryStore) evict(now time.Time) {
	var soonestKey string
	var soonest time.Time
	first := true
	for k, e := range m.entries {
		if !now.Before(e.expires) {
			delete(m.entries, k)
			continue
		}
		if first || e.expires.Before(soonest) {
			soonestKey, soonest, first = k, e.expires, false
		}
	}
	if len(m.entries) >= m.max && !first {
		delete(m.entries, soonestKey)
	}
}

// Delete implements CacheStore.
func (m *MemoryStore) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.entries, k)
	}
	return nil
}

// Len reports the number of stored entries, including any not yet reaped. For tests and metrics.
func (m *MemoryStore) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

// Close implements CacheStore.
func (m *MemoryStore) Close() error { return nil }
