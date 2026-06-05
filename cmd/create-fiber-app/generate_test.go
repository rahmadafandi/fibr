// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanSelection(t *testing.T) {
	dests := func(d Data) []string {
		var out []string
		for _, s := range plan(d) {
			out = append(out, s.dest)
		}
		return out
	}
	ddd := dests(Data{Layout: "ddd", Sample: false})
	assert.Contains(t, ddd, "cmd/api/main.go")
	assert.Contains(t, ddd, "internal/infrastructure/config/config.go")
	assert.Contains(t, ddd, "internal/interface/http/router.go")

	// Sample files come from planModule now, not plan.
	assert.NotContains(t, ddd, "internal/domain/user/user.go")

	layered := dests(Data{Layout: "layered", Sample: false})
	assert.Contains(t, layered, "internal/config/config.go")
	assert.Contains(t, layered, "internal/router/router.go")
}

func TestGenerateCommonFiles(t *testing.T) {
	target := filepath.Join(t.TempDir(), "app")
	d := Data{Name: "app", Module: "github.com/me/app", DB: "postgres", Layout: "ddd"}
	for _, s := range plan(d) {
		if !strings.HasPrefix(s.tmpl, "common/") {
			continue // layout templates land in Task 3/4
		}
		require.NoError(t, renderFile(s, d, target))
	}
	gomod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(gomod), "module github.com/me/app")
	env, _ := os.ReadFile(filepath.Join(target, ".env.example"))
	assert.Contains(t, string(env), "postgres://")
	compose, _ := os.ReadFile(filepath.Join(target, "docker-compose.yml"))
	assert.Contains(t, string(compose), "postgres:16")
}

func TestGenerateSQLiteEnv(t *testing.T) {
	target := filepath.Join(t.TempDir(), "app")
	d := Data{Name: "app", Module: "m", DB: "sqlite", Layout: "ddd"}
	for _, s := range plan(d) {
		if s.tmpl != "common/env.tmpl" {
			continue
		}
		require.NoError(t, renderFile(s, d, target))
	}
	env, _ := os.ReadFile(filepath.Join(target, ".env.example"))
	assert.Contains(t, string(env), "file:")
}

func TestGenerateRefusesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x"), []byte("y"), 0o644))
	o := Options{Name: "app", Module: "m", DB: "sqlite", Layout: "ddd", Dir: dir, NoGit: true, NoTidy: true}
	assert.Error(t, Generate(o, &strings.Builder{}))
}

func generateInto(t *testing.T, o Options) string {
	t.Helper()
	if o.Dir == "" {
		o.Dir = filepath.Join(t.TempDir(), o.Name)
	}
	o.NoGit, o.NoTidy = true, true
	require.NoError(t, Generate(o, &strings.Builder{}))
	return o.Dir
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), want)
}

func TestGenerateDDDNoSample(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "postgres", Layout: "ddd"})
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "bootstrap.New")
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/config/config.go"), "func Load")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/router.go"), "func Register")
	_, err := os.Stat(filepath.Join(dir, "internal/domain/user/user.go"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenerateDDDSample(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd", Sample: true})
	assertFileContains(t, filepath.Join(dir, "internal/domain/user/user.go"), "type User struct")
	assertFileContains(t, filepath.Join(dir, "internal/domain/user/repository.go"), "type Repository interface")
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/persistence/user_repository_bun.go"), "func NewUserRepository")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/user_handler.go"), "/users")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/user_module.go"), "func NewUserModule(db *bun.DB) bootstrap.Module")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "app.Mount(httpiface.NewUserModule(db))")
}

func TestGenerateLayeredNoSample(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "postgres", Layout: "layered"})
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "router.Register")
	assertFileContains(t, filepath.Join(dir, "internal/config/config.go"), "func Load")
	assertFileContains(t, filepath.Join(dir, "internal/router/router.go"), "func Register")
	_, err := os.Stat(filepath.Join(dir, "internal/model/user.go"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenerateLayeredSample(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "layered", Sample: true})
	assertFileContains(t, filepath.Join(dir, "internal/model/user.go"), "bun.BaseModel")
	assertFileContains(t, filepath.Join(dir, "internal/repository/user_repo.go"), "func NewUserRepository")
	assertFileContains(t, filepath.Join(dir, "internal/service/user_service.go"), "func NewUserService")
	assertFileContains(t, filepath.Join(dir, "internal/handler/user_handler.go"), "func NewUserHandler")
	assertFileContains(t, filepath.Join(dir, "internal/handler/user_module.go"), "func NewUserModule(db *bun.DB) bootstrap.Module")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "app.Mount(handler.NewUserModule(db))")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestMatrixCompiles(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the matrix compile test (slow: runs go build x8)")
	}
	root := repoRoot(t)
	for _, layout := range []string{"ddd", "layered"} {
		for _, db := range []string{"postgres", "sqlite"} {
			for _, sample := range []bool{false, true} {
				name := layout + "-" + db
				if sample {
					name += "-sample"
				}
				t.Run(name, func(t *testing.T) {
					dir := filepath.Join(t.TempDir(), "app")
					o := Options{
						Name: "app", Module: "example.com/app",
						DB: db, Layout: layout, Sample: sample,
						Dir: dir, NoGit: true, NoTidy: false, Local: root,
					}
					require.NoError(t, Generate(o, &strings.Builder{}))
					cmd := exec.Command("go", "build", "./...")
					cmd.Dir = dir
					out, err := cmd.CombinedOutput()
					require.NoError(t, err, "go build failed:\n%s", out)
				})
			}
		}
	}
}
