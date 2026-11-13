// Copyright 2026 Rahmad Afandi. MIT License.

// Package randhex generates random hex strings from crypto/rand.
package randhex

import (
	"crypto/rand"
	"encoding/hex"
)

// String returns n cryptographically random bytes, hex-encoded (2n characters).
func String(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
