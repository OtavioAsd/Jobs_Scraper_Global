package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type memoryEntry struct {
	Value     []byte
	ExpiresAt time.Time
}

func (e memoryEntry) expired() bool {
	return time.Now().After(e.ExpiresAt)
}

type MemoryCache struct {
	mu    sync.RWMutex
	store map[string]memoryEntry
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		store: make(map[string]memoryEntry),
	}
}

func (m *MemoryCache) Get(_ context.Context, key string, target any) (bool, error) {
	m.mu.RLock()
	entry, exists := m.store[prefixed(key)]
	m.mu.RUnlock()

	if !exists || entry.expired() {
		return false, nil
	}

	if err := json.Unmarshal(entry.Value, target); err != nil {
		return false, fmt.Errorf("cache.Get %q: unmarshal: %w", key, err)
	}

	return true, nil
}

func (m *MemoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache.Set %q: marshal: %w", key, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[prefixed(key)] = memoryEntry{
		Value:     payload,
		ExpiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (m *MemoryCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, prefixed(key))

	return nil
}

func (m *MemoryCache) SetNX(_ context.Context, key string, value any, ttl time.Duration) (bool, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("cache.SetNX %q: marshal: %w", key, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pKey := prefixed(key)

	if entry, exists := m.store[pKey]; exists && !entry.expired() {
		return false, nil
	}

	m.store[pKey] = memoryEntry{
		Value:     payload,
		ExpiresAt: time.Now().Add(ttl),
	}

	return true, nil
}

func (m *MemoryCache) Status() Status {
	return Status{Provider: "memory"}
}
