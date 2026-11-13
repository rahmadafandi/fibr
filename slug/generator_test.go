// Copyright 2026 Rahmad Afandi. MIT License.

package slug

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/rahmadafandi/fibr/v2/fibrtest"
)

type post struct {
	bun.BaseModel `bun:"table:posts,alias:p"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Slug          string `bun:"slug"`
}

func newBunDB(t *testing.T) *bun.DB {
	t.Helper()
	db := fibrtest.NewDB(t)
	_, err := db.NewCreateTable().Model((*post)(nil)).Exec(context.Background())
	assert.NoError(t, err)
	return db
}

func TestGenerateUnique(t *testing.T) {
	db := newBunDB(t)
	ctx := context.Background()

	got, err := Generate(ctx, db, "posts", "My First Post")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "my-first-post-"))
	assert.Greater(t, len(got), len("my-first-post-"))
}

func TestGenerateNilDB(t *testing.T) {
	_, err := Generate(context.Background(), nil, "posts", "My Title")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGenerateFormat(t *testing.T) {
	db := newBunDB(t)
	got, err := Generate(context.Background(), db, "posts", "Título de Prueba")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "titulo-de-prueba-"))
	parts := strings.Split(got, "-")
	suffix := parts[len(parts)-1]
	assert.Len(t, suffix, 16) // 10 random bytes -> 16 base32 chars
}

func TestGenerateReadable(t *testing.T) {
	db := newBunDB(t)
	ctx := context.Background()

	// Free: the plain slug, no suffix.
	got, err := GenerateReadable(ctx, db, "posts", "Hello World")
	require.NoError(t, err)
	assert.Equal(t, "hello-world", got)

	// Taken: falls back to the suffixed form.
	_, err = db.NewInsert().Model(&post{Slug: "hello-world"}).Exec(ctx)
	require.NoError(t, err)
	got, err = GenerateReadable(ctx, db, "posts", "Hello World")
	require.NoError(t, err)
	assert.Regexp(t, `^hello-world-[a-z2-7]{16}$`, got)

	// A title with no slug-able characters never returns an empty slug.
	got, err = GenerateReadable(ctx, db, "posts", "!!!")
	require.NoError(t, err)
	assert.NotEmpty(t, got)

	_, err = GenerateReadable(ctx, nil, "posts", "x")
	assert.Error(t, err)
}
