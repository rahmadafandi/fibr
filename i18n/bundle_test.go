// Copyright 2026 Rahmad Afandi. MIT License.

package i18n_test

import (
	"testing"
	"testing/fstest"

	"github.com/rahmadafandi/fibr/i18n"
	"github.com/stretchr/testify/require"
)

func bundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	b := i18n.New(i18n.WithFallback("en"))
	require.NoError(t, b.LoadJSON("en", []byte(`{
		"welcome": "Hello, {name}!",
		"cart": {"items": {"one": "{n} item", "other": "{n} items"}},
		"greeting": {"morning": "Good morning"}
	}`)))
	require.NoError(t, b.LoadJSON("id", []byte(`{
		"welcome": "Halo, {name}!",
		"cart": {"items": {"one": "{n} barang", "other": "{n} barang"}}
	}`)))
	return b
}

func TestTranslateAndPlaceholder(t *testing.T) {
	b := bundle(t)
	require.Equal(t, "Hello, Sam!", b.Translate("en", "welcome", i18n.M{"name": "Sam"}))
	require.Equal(t, "Halo, Sam!", b.Translate("id", "welcome", i18n.M{"name": "Sam"}))
}

func TestFallbackLocale(t *testing.T) {
	b := bundle(t)
	require.Equal(t, "Good morning", b.Translate("id", "greeting.morning", nil))
}

func TestMissingKeyReturnsKey(t *testing.T) {
	b := bundle(t)
	require.Equal(t, "nope.missing", b.Translate("en", "nope.missing", nil))
}

func TestPlural(t *testing.T) {
	b := bundle(t)
	require.Equal(t, "1 item", b.Plural("en", "cart.items", 1, nil))
	require.Equal(t, "5 items", b.Plural("en", "cart.items", 5, nil))
}

func TestHasAndLocales(t *testing.T) {
	b := bundle(t)
	require.True(t, b.Has("en"))
	require.True(t, b.Has("id"))
	require.False(t, b.Has("fr"))
	require.ElementsMatch(t, []string{"en", "id"}, b.Locales())
}

func TestLoadFS(t *testing.T) {
	fsys := fstest.MapFS{
		"en.json":   {Data: []byte(`{"hi":"hi"}`)},
		"id.json":   {Data: []byte(`{"hi":"hai"}`)},
		"notes.txt": {Data: []byte("ignore me")},
	}
	b := i18n.New()
	require.NoError(t, b.LoadFS(fsys, "."))
	require.Equal(t, "hai", b.Translate("id", "hi", nil))
	require.ElementsMatch(t, []string{"en", "id"}, b.Locales())
}
