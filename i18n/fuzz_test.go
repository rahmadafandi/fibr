// Copyright 2026 Rahmad Afandi. MIT License.

package i18n

import "testing"

// FuzzMatchAcceptLanguage ensures the Accept-Language parser never panics and
// only ever returns a locale the bundle actually has (or "").
func FuzzMatchAcceptLanguage(f *testing.F) {
	b := New(WithFallback("en"))
	b.LoadMap("en", map[string]any{"x": "x"})
	b.LoadMap("id", map[string]any{"x": "x"})

	f.Add("id-ID,id;q=0.9,en;q=0.8")
	f.Add("")
	f.Add(";;;q=,,")
	f.Add("en;q=notanumber")

	f.Fuzz(func(t *testing.T, header string) {
		got := matchAcceptLanguage(b, header)
		if got != "" && !b.Has(got) {
			t.Fatalf("matchAcceptLanguage returned %q which the bundle does not have", got)
		}
	})
}

// FuzzPluralCategory ensures the CLDR resolver never panics and always returns a
// valid category for any locale/n.
func FuzzPluralCategory(f *testing.F) {
	valid := map[string]bool{"zero": true, "one": true, "two": true, "few": true, "many": true, "other": true}
	f.Add("en", 1)
	f.Add("ru-RU", 21)
	f.Add("", -5)
	f.Add("ar", 0)

	f.Fuzz(func(t *testing.T, locale string, n int) {
		if got := pluralCategory(locale, n); !valid[got] {
			t.Fatalf("pluralCategory(%q,%d) = %q, not a valid CLDR category", locale, n, got)
		}
	})
}
