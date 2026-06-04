// Copyright 2026 Rahmad Afandi. MIT License.

package slug

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type post struct {
	bun.BaseModel `bun:"table:posts,alias:p"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Slug          string `bun:"slug"`
}

func newBunDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	assert.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.Close() })
	_, err = db.NewCreateTable().Model((*post)(nil)).Exec(context.Background())
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
