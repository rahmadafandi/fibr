// Copyright 2026 Rahmad Afandi. MIT License.

// Package slug generates unique URL-safe slugs backed by a Bun database.
package slug

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/uptrace/bun"
)

const maxSlugAttempts = 10

// Generate creates a unique slug for the given table using a Bun database. The
// slug always carries a random suffix ("my-title-<16 chars>"), which makes it
// safe under concurrent calls with the same title.
func Generate(ctx context.Context, db *bun.DB, tableName string, title string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("slug: db must not be nil")
	}

	baseSlug := slug.Make(title)

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		randomBytes := make([]byte, 10)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", fmt.Errorf("slug: reading random bytes: %w", err)
		}

		uniqueID := strings.ToLower(strings.TrimRight(base32.StdEncoding.EncodeToString(randomBytes), "="))
		slugCandidate := baseSlug + "-" + uniqueID

		count, err := db.NewSelect().Table(tableName).Where("slug = ?", slugCandidate).Count(ctx)
		if err != nil {
			return "", fmt.Errorf("slug: check uniqueness: %w", err)
		}

		if count == 0 {
			return slugCandidate, nil
		}
	}

	return "", fmt.Errorf("slug: no unique slug for %q after %d attempts", title, maxSlugAttempts)
}

// GenerateReadable returns the plain slug for title ("my-title") when the table
// has no row with it yet, and falls back to Generate's suffixed form otherwise.
//
// The check and your later insert are separate steps, so two concurrent calls
// with the same title can both get the plain slug. Keep a unique index on the
// slug column and, on a unique violation, retry with Generate.
func GenerateReadable(ctx context.Context, db *bun.DB, tableName string, title string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("slug: db must not be nil")
	}
	plain := slug.Make(title)
	if plain != "" {
		count, err := db.NewSelect().Table(tableName).Where("slug = ?", plain).Count(ctx)
		if err != nil {
			return "", fmt.Errorf("slug: check uniqueness: %w", err)
		}
		if count == 0 {
			return plain, nil
		}
	}
	return Generate(ctx, db, tableName, title)
}
