// Copyright 2026 Rahmad Afandi. MIT License.

package i18n

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	localeKey = "fibr_i18n_locale"
	bundleKey = "fibr_i18n_bundle"
)

type mwConfig struct {
	queryKey  string
	cookieKey string
}

// MWOption configures the middleware.
type MWOption func(*mwConfig)

// WithQueryKey sets the query parameter checked for the locale (default "lang").
func WithQueryKey(k string) MWOption { return func(c *mwConfig) { c.queryKey = k } }

// WithCookieKey sets the cookie checked for the locale (default "lang").
func WithCookieKey(k string) MWOption { return func(c *mwConfig) { c.cookieKey = k } }

// Middleware resolves the request locale (query > cookie > Accept-Language >
// fallback) and stores it, with the bundle, for T/N/Locale.
func Middleware(b *Bundle, opts ...MWOption) fiber.Handler {
	cfg := mwConfig{queryKey: "lang", cookieKey: "lang"}
	for _, o := range opts {
		o(&cfg)
	}
	return func(c *fiber.Ctx) error {
		c.Locals(localeKey, resolve(b, c, cfg))
		c.Locals(bundleKey, b)
		return c.Next()
	}
}

func resolve(b *Bundle, c *fiber.Ctx, cfg mwConfig) string {
	if v := c.Query(cfg.queryKey); v != "" && b.Has(v) {
		return v
	}
	if v := c.Cookies(cfg.cookieKey); v != "" && b.Has(v) {
		return v
	}
	if v := matchAcceptLanguage(b, c.Get(fiber.HeaderAcceptLanguage)); v != "" {
		return v
	}
	return b.fallback
}

// matchAcceptLanguage returns the first locale (by q-value) present in b, trying
// each tag and its base subtag (e.g. "id-ID" -> "id").
func matchAcceptLanguage(b *Bundle, header string) string {
	if header == "" {
		return ""
	}
	type pref struct {
		tag string
		q   float64
	}
	var prefs []pref
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			if qi := strings.Index(part[i:], "q="); qi >= 0 {
				if f, err := strconv.ParseFloat(part[i+qi+2:], 64); err == nil {
					q = f
				}
			}
		}
		prefs = append(prefs, pref{strings.ToLower(tag), q})
	}
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].q > prefs[j].q })
	for _, p := range prefs {
		if b.Has(p.tag) {
			return p.tag
		}
		if base, _, ok := strings.Cut(p.tag, "-"); ok && b.Has(base) {
			return base
		}
	}
	return ""
}

// Locale returns the resolved request locale (or the bundle fallback / "").
func Locale(c *fiber.Ctx) string {
	if v, ok := c.Locals(localeKey).(string); ok && v != "" {
		return v
	}
	if b, ok := c.Locals(bundleKey).(*Bundle); ok && b != nil {
		return b.fallback
	}
	return ""
}

// T translates key for the request locale.
func T(c *fiber.Ctx, key string, args M) string {
	b, ok := c.Locals(bundleKey).(*Bundle)
	if !ok || b == nil {
		return key
	}
	return b.Translate(Locale(c), key, args)
}

// N translates the plural form of key for the request locale and count n.
func N(c *fiber.Ctx, key string, n int, args M) string {
	b, ok := c.Locals(bundleKey).(*Bundle)
	if !ok || b == nil {
		return key
	}
	return b.Plural(Locale(c), key, n, args)
}
