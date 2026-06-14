// Copyright 2026 Rahmad Afandi. MIT License.

package apikey_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/apikey"
)

func TestGenerateAndHash(t *testing.T) {
	key, hash, err := apikey.Generate()
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.Equal(t, apikey.Hash(key), hash)

	key2, _, err := apikey.Generate()
	require.NoError(t, err)
	assert.NotEqual(t, key, key2, "keys should be unique")
}

func TestMapStoreLookup(t *testing.T) {
	hash := apikey.Hash("secret")
	store := apikey.MapStore(map[string]apikey.Identity{
		hash: {ID: "svc1", Scopes: []string{"read"}},
	})
	ctx := context.Background()

	id, err := store.Lookup(ctx, hash)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, "svc1", id.ID)

	missing, err := store.Lookup(ctx, apikey.Hash("nope"))
	require.NoError(t, err)
	assert.Nil(t, missing)
}
