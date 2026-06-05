// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCompare(t *testing.T) {
	h, err := Hash("s3cret")
	require.NoError(t, err)
	assert.NotEqual(t, "s3cret", h)
	assert.NoError(t, Compare(h, "s3cret"))
	assert.Error(t, Compare(h, "wrong"))
}

func TestHashIsSalted(t *testing.T) {
	h1, err := Hash("same")
	require.NoError(t, err)
	h2, err := Hash("same")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "bcrypt salts each hash")
}

func TestWithContextKey(t *testing.T) {
	c := newConfig(WithContextKey("custom"))
	assert.Equal(t, "custom", c.contextKey)
}

func TestDefaultContextKey(t *testing.T) {
	c := newConfig()
	assert.Equal(t, DefaultClaimsKey, c.contextKey)
}

func TestWithContextKeyEmptyIgnored(t *testing.T) {
	c := newConfig(WithContextKey(""))
	assert.Equal(t, DefaultClaimsKey, c.contextKey)
}
