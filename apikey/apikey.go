// Copyright 2026 Rahmad Afandi. MIT License.

// Package apikey provides API-key authentication for Fiber, distinct from the
// JWT-bearer auth package: a presented key is hashed and looked up in a
// pluggable store to resolve its identity and scopes.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Identity is who an API key belongs to.
type Identity struct {
	ID     string
	Scopes []string
	Meta   map[string]any
}

// Store resolves a key's SHA-256 hash to an Identity. It returns (nil, nil) when
// the key is unknown.
type Store interface {
	Lookup(ctx context.Context, keyHash string) (*Identity, error)
}

// mapStore is an in-memory Store keyed by key hash.
type mapStore map[string]Identity

// MapStore returns an in-memory Store keyed by key hash (use Hash to compute the
// keys). The map is copied.
func MapStore(m map[string]Identity) Store {
	cp := make(mapStore, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func (s mapStore) Lookup(_ context.Context, keyHash string) (*Identity, error) {
	id, ok := s[keyHash]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

// Hash returns the SHA-256 hex of a raw key, which is what a Store keys on.
func Hash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Generate returns a new random API key and its hash. Give the key to the
// client and persist the hash.
func Generate() (key, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("apikey: generate: %w", err)
	}
	key = hex.EncodeToString(b)
	return key, Hash(key), nil
}
