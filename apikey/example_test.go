// Copyright 2026 Rahmad Afandi. MIT License.

package apikey_test

import (
	"context"
	"fmt"

	"github.com/rahmadafandi/fibr/apikey"
)

// Generate a key for a client, store its hash, then authenticate with it.
func ExampleGenerate() {
	key, hash, _ := apikey.Generate()

	// Persist hash (here in a MapStore); hand `key` to the client.
	store := apikey.MapStore(map[string]apikey.Identity{
		hash: {ID: "service-a", Scopes: []string{"read"}},
	})

	// On a request, hash the presented key and look it up.
	id, _ := store.Lookup(context.Background(), apikey.Hash(key))
	fmt.Println(id.ID, id.Scopes)
	// Output: service-a [read]
}
