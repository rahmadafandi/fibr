// Copyright 2026 Rahmad Afandi. MIT License.

package migrate

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahmadafandi/fibr/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

//go:embed testdata
var testMigrationsFS embed.FS

func testMigrations(t *testing.T) *Migrations {
	t.Helper()
	sub, err := fs.Sub(testMigrationsFS, "testdata")
	require.NoError(t, err)
	ms := NewMigrations()
	require.NoError(t, ms.Discover(sub))
	return ms
}

func openTestDB(t *testing.T) *bun.DB {
	t.Helper()
	db, err := database.NewBun("file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func tableExists(t *testing.T, db *bun.DB) bool {
	t.Helper()
	var n int
	err := db.NewRaw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='widgets'").Scan(context.Background(), &n)
	require.NoError(t, err)
	return n == 1
}

func TestUpCreatesTableThenIdempotent(t *testing.T) {
	db := openTestDB(t)
	ms := testMigrations(t)
	ctx := context.Background()

	msg, err := Up(ctx, db, ms)
	require.NoError(t, err)
	assert.Contains(t, msg, "migrated")
	assert.True(t, tableExists(t, db), "widgets table should exist after Up")

	msg, err = Up(ctx, db, ms)
	require.NoError(t, err)
	assert.Contains(t, msg, "no new migrations")
}

func TestStatusReportsApplied(t *testing.T) {
	db := openTestDB(t)
	ms := testMigrations(t)
	ctx := context.Background()
	_, err := Up(ctx, db, ms)
	require.NoError(t, err)

	out, err := Status(ctx, db, ms)
	require.NoError(t, err)
	assert.Contains(t, out, "applied")
}

func TestDownRollsBack(t *testing.T) {
	db := openTestDB(t)
	ms := testMigrations(t)
	ctx := context.Background()
	_, err := Up(ctx, db, ms)
	require.NoError(t, err)

	msg, err := Down(ctx, db, ms)
	require.NoError(t, err)
	assert.Contains(t, msg, "rolled back")
	assert.False(t, tableExists(t, db), "widgets table should be gone after Down")
}

func TestCreateWritesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "migrations")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	ms := NewMigrations(WithMigrationsDirectory(dir))
	path, err := Create(ms, "add_things")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, "_add_things.go"))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), "package migrations")
}
