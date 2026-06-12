// Copyright 2026 Rahmad Afandi. MIT License.

// Package i18n is a small, dependency-free internationalization helper: a
// message catalog (Bundle) loaded from nested JSON, with {placeholder}
// substitution, one/other pluralization, and a fallback locale. Request-scoped
// locale detection lives in Middleware.
package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// M is a placeholder argument set: {key} in a message is replaced by args[key].
type M map[string]any

// Bundle is a catalog of messages per locale.
type Bundle struct {
	messages map[string]map[string]string
	fallback string
}

// Option configures a Bundle.
type Option func(*Bundle)

// WithFallback sets the locale used when a key is missing in the requested
// locale (default "en").
func WithFallback(locale string) Option { return func(b *Bundle) { b.fallback = locale } }

// New creates an empty bundle.
func New(opts ...Option) *Bundle {
	b := &Bundle{messages: map[string]map[string]string{}, fallback: "en"}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Fallback returns the fallback locale.
func (b *Bundle) Fallback() string { return b.fallback }

// Has reports whether the bundle has any messages for locale.
func (b *Bundle) Has(locale string) bool { return len(b.messages[locale]) > 0 }

// Locales returns the loaded locales, sorted.
func (b *Bundle) Locales() []string {
	out := make([]string, 0, len(b.messages))
	for l := range b.messages {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// LoadMap merges a nested message map into locale (flattened to dotted keys).
func (b *Bundle) LoadMap(locale string, nested map[string]any) {
	m := b.messages[locale]
	if m == nil {
		m = map[string]string{}
		b.messages[locale] = m
	}
	flatten("", nested, m)
}

// LoadJSON merges nested JSON messages into locale.
func (b *Bundle) LoadJSON(locale string, data []byte) error {
	var nested map[string]any
	if err := json.Unmarshal(data, &nested); err != nil {
		return fmt.Errorf("i18n: parse %s: %w", locale, err)
	}
	b.LoadMap(locale, nested)
	return nil
}

// LoadFS loads every <locale>.json file in dir of fsys (file base name is the
// locale).
func (b *Bundle) LoadFS(fsys fs.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("i18n: read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("i18n: read %s: %w", e.Name(), err)
		}
		if err := b.LoadJSON(strings.TrimSuffix(e.Name(), ".json"), data); err != nil {
			return err
		}
	}
	return nil
}

// Translate resolves key in locale, then the fallback locale, then returns key
// itself if missing. args fill {placeholders}.
func (b *Bundle) Translate(locale, key string, args M) string {
	if s, ok := b.lookup(locale, key); ok {
		return substitute(s, args)
	}
	if locale != b.fallback {
		if s, ok := b.lookup(b.fallback, key); ok {
			return substitute(s, args)
		}
	}
	return key
}

// Plural resolves "<key>.one" when n == 1 else "<key>.other"; {n} is set to n
// (unless already provided in args).
func (b *Bundle) Plural(locale, key string, n int, args M) string {
	suffix := "other"
	if n == 1 {
		suffix = "one"
	}
	merged := M{"n": n}
	for k, v := range args {
		merged[k] = v
	}
	return b.Translate(locale, key+"."+suffix, merged)
}

func (b *Bundle) lookup(locale, key string) (string, bool) {
	m := b.messages[locale]
	if m == nil {
		return "", false
	}
	s, ok := m[key]
	return s, ok
}

func substitute(s string, args M) string {
	if len(args) == 0 {
		return s
	}
	for k, v := range args {
		s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprint(v))
	}
	return s
}

func flatten(prefix string, in map[string]any, out map[string]string) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			flatten(key, child, out)
		} else {
			out[key] = fmt.Sprint(v)
		}
	}
}
