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

// patchMainForModule inserts the Mount call (and, for layered, the handler
// import) into a generated non-sample main.go so the added module is wired.
func patchMainForModule(t *testing.T, dir, layout, typeName string) {
	t.Helper()
	mainPath := filepath.Join(dir, "cmd/api/main.go")
	b, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	src := string(b)

	const anchor = "\tfmt.Printf(\"listening on :%s\\n\", cfg.Port)"
	require.Contains(t, src, anchor, "main.go anchor not found")

	var mount string
	switch layout {
	case "ddd":
		// httpiface already imported in the non-sample main.
		mount = "\tif err := app.Mount(httpiface.New" + typeName + "Module(db)); err != nil {\n\t\tlog.Fatal(err)\n\t}\n"
	case "layered":
		src = strings.Replace(src,
			"\t\"example.com/app/internal/router\"",
			"\t\"example.com/app/internal/router\"\n\t\"example.com/app/internal/handler\"",
			1)
		mount = "\tif err := app.Mount(handler.New" + typeName + "Module(db)); err != nil {\n\t\tlog.Fatal(err)\n\t}\n"
	}
	src = strings.Replace(src, anchor, mount+anchor, 1)
	require.NoError(t, os.WriteFile(mainPath, []byte(src), 0o644))
}

func TestMigrateRunsE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the migrate runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Sample: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "e2e.db")
	env := append(os.Environ(), "DATABASE_URL=file:"+dbPath+"?cache=shared")

	runApp := func(args ...string) (string, error) {
		cmd := exec.Command("go", append([]string{"run", "./cmd/api"}, args...)...)
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	out, err := runApp("migrate", "up")
	require.NoError(t, err, "migrate up failed:\n%s", out)
	assert.Contains(t, out, "migrated")

	out, err = runApp("migrate", "up")
	require.NoError(t, err, "second migrate up failed:\n%s", out)
	assert.Contains(t, out, "no new migrations")

	out, err = runApp("migrate", "status")
	require.NoError(t, err, "migrate status failed:\n%s", out)
	assert.Contains(t, out, "applied")
}

func TestAddModuleE2ECompiles(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the add-module compile test (slow: runs go build)")
	}
	root := repoRoot(t)
	for _, layout := range []string{"ddd", "layered"} {
		t.Run(layout, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			// Generate a base project (no sample) wired against the local lib.
			require.NoError(t, Generate(Options{
				Name: "app", Module: "example.com/app",
				DB: "sqlite", Layout: layout, Sample: false,
				Dir: dir, NoGit: true, NoTidy: false, Local: root,
			}, &strings.Builder{}))

			// Add a module, then wire it into main.go.
			require.NoError(t, AddModule(AddModuleOptions{Name: "product", Dir: dir, Layout: layout}, &strings.Builder{}))
			patchMainForModule(t, dir, layout, "Product")

			// go mod tidy to pull the (now-used) extra imports, then build.
			tidy := exec.Command("go", "mod", "tidy")
			tidy.Dir = dir
			if out, err := tidy.CombinedOutput(); err != nil {
				t.Fatalf("go mod tidy failed:\n%s", out)
			}
			build := exec.Command("go", "build", "./...")
			build.Dir = dir
			out, err := build.CombinedOutput()
			require.NoError(t, err, "go build failed:\n%s", out)
		})
	}
}
