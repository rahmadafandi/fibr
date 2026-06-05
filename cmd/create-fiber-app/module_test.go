// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveModuleNames(t *testing.T) {
	md, err := deriveModuleNames("product")
	require.NoError(t, err)
	assert.Equal(t, "Product", md.Type)
	assert.Equal(t, "product", md.Pkg)
	assert.Equal(t, "products", md.Plural)
}

func TestDeriveModuleNamesUpcasesAndLowers(t *testing.T) {
	md, err := deriveModuleNames("User")
	require.NoError(t, err)
	assert.Equal(t, "User", md.Type)
	assert.Equal(t, "user", md.Pkg)
}

func TestDeriveModuleNamesPreservesCamelCase(t *testing.T) {
	md, err := deriveModuleNames("OrderItem")
	require.NoError(t, err)
	assert.Equal(t, "OrderItem", md.Type)
	assert.Equal(t, "orderitem", md.Pkg)
	assert.Equal(t, "orderitems", md.Plural)
}

func TestDeriveModuleNamesRejectsInvalid(t *testing.T) {
	for _, bad := range []string{"", "a/b", "../evil", "9lives", "has space", "x-y"} {
		_, err := deriveModuleNames(bad)
		assert.Error(t, err, "expected %q to be rejected", bad)
	}
}

func TestPlanModuleDDD(t *testing.T) {
	md := ModuleData{Layout: "ddd", Type: "Product", Pkg: "product", Plural: "products"}
	var dests []string
	for _, s := range planModule(md) {
		dests = append(dests, s.dest)
	}
	assert.Contains(t, dests, "internal/domain/product/product.go")
	assert.Contains(t, dests, "internal/domain/product/repository.go")
	assert.Contains(t, dests, "internal/application/product/service.go")
	assert.Contains(t, dests, "internal/infrastructure/persistence/product_repository_bun.go")
	assert.Contains(t, dests, "internal/interface/http/product_handler.go")
	assert.Contains(t, dests, "internal/interface/http/product_module.go")
}

func TestRenderModuleDDDGofmtClean(t *testing.T) {
	dir := t.TempDir()
	md := ModuleData{Module: "github.com/me/app", Layout: "ddd", Type: "Product", Pkg: "product", Plural: "products"}
	for _, s := range planModule(md) {
		require.NoError(t, renderFile(s, md, dir))
	}
	assertFileContains(t, dir+"/internal/interface/http/product_module.go", "func NewProductModule(db *bun.DB) bootstrap.Module")
	assertFileContains(t, dir+"/internal/interface/http/product_module.go", "func (m *ProductModule) Migrate(ctx context.Context) error")
	assertFileContains(t, dir+"/internal/interface/http/product_handler.go", "/products")
}

func TestPlanModuleLayered(t *testing.T) {
	md := ModuleData{Layout: "layered", Type: "Product", Pkg: "product", Plural: "products"}
	var dests []string
	for _, s := range planModule(md) {
		dests = append(dests, s.dest)
	}
	assert.Contains(t, dests, "internal/model/product.go")
	assert.Contains(t, dests, "internal/repository/product_repo.go")
	assert.Contains(t, dests, "internal/service/product_service.go")
	assert.Contains(t, dests, "internal/handler/product_handler.go")
	assert.Contains(t, dests, "internal/handler/product_module.go")
}

func TestRenderModuleLayeredGofmtClean(t *testing.T) {
	dir := t.TempDir()
	md := ModuleData{Module: "github.com/me/app", Layout: "layered", Type: "Product", Pkg: "product", Plural: "products"}
	for _, s := range planModule(md) {
		require.NoError(t, renderFile(s, md, dir))
	}
	assertFileContains(t, dir+"/internal/handler/product_module.go", "func NewProductModule(db *bun.DB) bootstrap.Module")
	assertFileContains(t, dir+"/internal/handler/product_handler.go", "/products")
	assertFileContains(t, dir+"/internal/repository/product_repo.go", "func MigrateProduct(ctx context.Context, db *bun.DB) error")
}
