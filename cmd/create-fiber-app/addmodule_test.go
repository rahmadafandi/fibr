// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.25\n"), 0o644))
}

func TestReadGoModule(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/me/app")
	got, err := readGoModule(dir)
	require.NoError(t, err)
	assert.Equal(t, "github.com/me/app", got)
}

func TestReadGoModuleMissing(t *testing.T) {
	_, err := readGoModule(t.TempDir())
	assert.Error(t, err)
}

func TestDetectLayout(t *testing.T) {
	ddd := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(ddd, "internal/domain"), 0o755))
	got, err := detectLayout(ddd)
	require.NoError(t, err)
	assert.Equal(t, "ddd", got)

	layered := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(layered, "internal/model"), 0o755))
	got, err = detectLayout(layered)
	require.NoError(t, err)
	assert.Equal(t, "layered", got)

	_, err = detectLayout(t.TempDir())
	assert.Error(t, err)
}

func TestAddModuleDDD(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/me/app")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal/domain"), 0o755))

	var out strings.Builder
	require.NoError(t, AddModule(AddModuleOptions{Name: "product", Dir: dir}, &out))

	assertFileContains(t, filepath.Join(dir, "internal/interface/http/product_module.go"), "func NewProductModule(db *bun.DB) bootstrap.Module")
	assertFileContains(t, filepath.Join(dir, "internal/domain/product/product.go"), "type Product struct")
	assert.Contains(t, out.String(), "app.Mount(httpiface.NewProductModule(db))")
}

func TestAddModuleLayoutFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/me/app")
	// no layout dirs present; flag forces layered
	require.NoError(t, AddModule(AddModuleOptions{Name: "product", Dir: dir, Layout: "layered"}, &strings.Builder{}))
	assertFileContains(t, filepath.Join(dir, "internal/handler/product_module.go"), "func NewProductModule(db *bun.DB) bootstrap.Module")
}

func TestAddModuleRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/me/app")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal/domain/product"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal/domain/product/product.go"), []byte("package product\n"), 0o644))

	err := AddModule(AddModuleOptions{Name: "product", Dir: dir, Layout: "ddd"}, &strings.Builder{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddModuleNotAGoModule(t *testing.T) {
	err := AddModule(AddModuleOptions{Name: "product", Dir: t.TempDir(), Layout: "ddd"}, &strings.Builder{})
	assert.Error(t, err)
}

func TestAddModuleInvalidName(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/me/app")
	err := AddModule(AddModuleOptions{Name: "../evil", Dir: dir, Layout: "ddd"}, &strings.Builder{})
	assert.Error(t, err)
}
