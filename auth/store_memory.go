// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-process TokenStore for tests and single-node development.
// It is NOT suitable for multi-instance deployments because state is not shared
// across processes. Entries expire lazily on read; there is no background sweep.
type MemoryStore struct {
	mu     sync.Mutex
	block  map[string]time.Time // jti -> expiry
	family map[string]famEntry  // fid -> active refresh jti + expiry
}

type famEntry struct {
	jti string
	exp time.Time
}

// NewMemoryStore creates an empty in-memory token store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		block:  make(map[string]time.Time),
		family: make(map[string]famEntry),
	}
}

// Block marks jti revoked until ttl elapses.
func (m *MemoryStore) Block(_ context.Context, jti string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.block[jti] = time.Now().Add(ttl)
	return nil
}

// IsBlocked reports whether jti is currently revoked.
func (m *MemoryStore) IsBlocked(_ context.Context, jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.block[jti]
	if !ok {
		return false, nil
	}
	if time.Now().After(exp) {
		delete(m.block, jti)
		return false, nil
	}
	return true, nil
}

// SetFamily records jti as the active refresh token for family fid until ttl.
func (m *MemoryStore) SetFamily(_ context.Context, fid, jti string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.family[fid] = famEntry{jti: jti, exp: time.Now().Add(ttl)}
	return nil
}

// Family returns the active refresh jti for fid.
func (m *MemoryStore) Family(_ context.Context, fid string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.family[fid]
	if !ok {
		return "", false, nil
	}
	if time.Now().After(e.exp) {
		delete(m.family, fid)
		return "", false, nil
	}
	return e.jti, true, nil
}

// RevokeFamily removes the family record, invalidating all its tokens.
func (m *MemoryStore) RevokeFamily(_ context.Context, fid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.family, fid)
	return nil
}
