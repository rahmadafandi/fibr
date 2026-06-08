// Copyright 2026 Rahmad Afandi. MIT License.

package migrate

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rahmadafandi/fibr/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestNewCommandHasChildren(t *testing.T) {
	cmd := NewCommand(func(ctx context.Context) (*bun.DB, error) { return nil, nil }, NewMigrations())
	assert.Equal(t, "migrate", cmd.Name())

	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"up", "down", "status", "create"} {
		assert.True(t, names[want], "missing subcommand %q", want)
	}
}

func TestNewCommandUpExecutesCleanly(t *testing.T) {
	db, err := database.NewBun("file::memory:?cache=shared")
	require.NoError(t, err)
	// command closes db via defer; no t.Cleanup to avoid double-close.

	var out strings.Builder
	cmd := NewCommand(func(ctx context.Context) (*bun.DB, error) { return db, nil }, testMigrations(t))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"up"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "migrated")
}

func TestNewCommandCreateRuns(t *testing.T) {
	dir := t.TempDir() + "/migrations"
	require.NoError(t, os.MkdirAll(dir, 0o755))
	ms := NewMigrations(WithMigrationsDirectory(dir))
	cmd := NewCommand(func(ctx context.Context) (*bun.DB, error) { return nil, nil }, ms)
	cmd.SetArgs([]string{"create", "add_stuff"})
	require.NoError(t, cmd.Execute())
}
