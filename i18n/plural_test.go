// Copyright 2026 Rahmad Afandi. MIT License.

package i18n

import "testing"

func TestPluralCategory(t *testing.T) {
	cases := []struct {
		locale string
		n      int
		want   string
	}{
		// English (default rule)
		{"en", 0, "other"}, {"en", 1, "one"}, {"en", 2, "other"},
		// Indonesian — no plural
		{"id", 1, "other"}, {"id", 5, "other"},
		// French — 0 and 1 are "one"
		{"fr", 0, "one"}, {"fr", 1, "one"}, {"fr", 2, "other"},
		// Russian
		{"ru", 1, "one"}, {"ru", 21, "one"}, {"ru", 2, "few"}, {"ru", 23, "few"},
		{"ru", 5, "many"}, {"ru", 11, "many"}, {"ru", 111, "many"},
		// Polish
		{"pl", 1, "one"}, {"pl", 2, "few"}, {"pl", 22, "few"}, {"pl", 5, "many"}, {"pl", 12, "many"},
		// Arabic
		{"ar", 0, "zero"}, {"ar", 1, "one"}, {"ar", 2, "two"}, {"ar", 3, "few"},
		{"ar", 11, "many"}, {"ar", 100, "other"},
		// base subtag extraction + unknown -> en rule
		{"ru-RU", 2, "few"}, {"de", 1, "one"}, {"de", 9, "other"},
	}
	for _, c := range cases {
		if got := pluralCategory(c.locale, c.n); got != c.want {
			t.Errorf("pluralCategory(%q, %d) = %q, want %q", c.locale, c.n, got, c.want)
		}
	}
}

func TestPluralResolvesCategory(t *testing.T) {
	b := New(WithFallback("en"))
	_ = b.LoadJSON("ru", []byte(`{"apples":{"one":"{n} яблоко","few":"{n} яблока","many":"{n} яблок"}}`))
	if got := b.Plural("ru", "apples", 1, nil); got != "1 яблоко" {
		t.Errorf("ru 1 = %q", got)
	}
	if got := b.Plural("ru", "apples", 3, nil); got != "3 яблока" {
		t.Errorf("ru 3 = %q", got)
	}
	if got := b.Plural("ru", "apples", 5, nil); got != "5 яблок" {
		t.Errorf("ru 5 = %q", got)
	}
}
