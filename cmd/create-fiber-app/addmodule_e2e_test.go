// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

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

func TestAuthCompilesE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the auth compile test (slow: go build)")
	}
	root := repoRoot(t)
	cases := []struct {
		layout string
		sample bool
	}{
		{"ddd", false},
		{"layered", false},
		{"ddd", true},
		{"layered", true},
	}
	for _, tc := range cases {
		name := tc.layout
		if tc.sample {
			name += "-sample"
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			require.NoError(t, Generate(Options{
				Name: "app", Module: "example.com/app",
				DB: "sqlite", Layout: tc.layout, Auth: true, Sample: tc.sample,
				Dir: dir, NoGit: true, NoTidy: false, Local: root,
			}, &strings.Builder{}))
			build := exec.Command("go", "build", "./...")
			build.Dir = dir
			out, err := build.CombinedOutput()
			require.NoError(t, err, "go build failed:\n%s", out)
		})
	}
}

func TestAuthFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the auth runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Auth: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "auth.db")
	port := "39517"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("a", 64),
	)

	mig := exec.Command("go", "run", "./cmd/api", "migrate", "up")
	mig.Dir, mig.Env = dir, env
	if out, err := mig.CombinedOutput(); err != nil {
		t.Fatalf("migrate up failed:\n%s", out)
	}

	srv := exec.Command("go", "run", "./cmd/api")
	srv.Dir, srv.Env = dir, env
	srv.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, srv.Start())
	defer func() { _ = syscall.Kill(-srv.Process.Pid, syscall.SIGKILL) }()

	base := "http://127.0.0.1:" + port
	waitReady(t, base+"/livez")

	code, _ := postJSON(t, base+"/auth/register", `{"email":"a@b.com","password":"secret123"}`, "")
	require.Equal(t, 201, code)

	code, body := postJSON(t, base+"/auth/login", `{"email":"a@b.com","password":"secret123"}`, "")
	require.Equal(t, 200, code)
	token := dataField(t, body, "access_token")
	require.NotEmpty(t, token)

	assert.Equal(t, 200, getCode(t, base+"/auth/me", token))
	assert.Equal(t, 401, getCode(t, base+"/auth/me", ""))
	assert.Equal(t, 403, getCode(t, base+"/auth/admin", token))
}

func TestAuthRefreshFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the auth refresh runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Auth: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "refresh.db")
	port := "39518"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("b", 64),
	)

	mig := exec.Command("go", "run", "./cmd/api", "migrate", "up")
	mig.Dir, mig.Env = dir, env
	if out, err := mig.CombinedOutput(); err != nil {
		t.Fatalf("migrate up failed:\n%s", out)
	}

	srv := exec.Command("go", "run", "./cmd/api")
	srv.Dir, srv.Env = dir, env
	srv.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, srv.Start())
	defer func() { _ = syscall.Kill(-srv.Process.Pid, syscall.SIGKILL) }()

	base := "http://127.0.0.1:" + port
	waitReady(t, base+"/livez")

	// register
	code, _ := postJSON(t, base+"/auth/register", `{"email":"r@b.com","password":"secret123"}`, "")
	require.Equal(t, 201, code)

	// login -> access + refresh pair
	code, body := postJSON(t, base+"/auth/login", `{"email":"r@b.com","password":"secret123"}`, "")
	require.Equal(t, 200, code)
	access := dataField(t, body, "access_token")
	refresh := dataField(t, body, "refresh_token")
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)

	// /me works with access token
	require.Equal(t, 200, getCode(t, base+"/auth/me", access))

	// refresh -> new pair
	code, body = postJSON(t, base+"/auth/refresh", `{"refresh_token":"`+refresh+`"}`, "")
	require.Equal(t, 200, code)
	newRefresh := dataField(t, body, "refresh_token")
	newAccess := dataField(t, body, "access_token")
	require.NotEmpty(t, newRefresh)
	require.NotEmpty(t, newAccess)
	require.NotEqual(t, refresh, newRefresh)
	require.NotEqual(t, access, newAccess)

	// old refresh now rejected; reuse kills the family so the new one dies too
	code, _ = postJSON(t, base+"/auth/refresh", `{"refresh_token":"`+refresh+`"}`, "")
	require.Equal(t, 401, code)
	code, _ = postJSON(t, base+"/auth/refresh", `{"refresh_token":"`+newRefresh+`"}`, "")
	require.Equal(t, 401, code)

	// fresh login, then logout blocklists the access token
	code, body = postJSON(t, base+"/auth/login", `{"email":"r@b.com","password":"secret123"}`, "")
	require.Equal(t, 200, code)
	access2 := dataField(t, body, "access_token")
	refresh2 := dataField(t, body, "refresh_token")
	require.Equal(t, 200, getCode(t, base+"/auth/me", access2))

	code, _ = postJSON(t, base+"/auth/logout", `{"refresh_token":"`+refresh2+`"}`, access2)
	require.Equal(t, 200, code)

	// access2 now blocklisted -> /me 401
	require.Equal(t, 401, getCode(t, base+"/auth/me", access2))

	// logout also revoked the refresh family -> refresh2 is dead
	code, _ = postJSON(t, base+"/auth/refresh", `{"refresh_token":"`+refresh2+`"}`, "")
	require.Equal(t, 401, code)
}

func waitReady(t *testing.T, url string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server did not become ready at %s", url)
}

func postJSON(t *testing.T, url, body, bearer string) (int, string) {
	t.Helper()
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func getCode(t *testing.T, url, bearer string) int {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func dataField(t *testing.T, body, field string) string {
	t.Helper()
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &parsed))
	s, ok := parsed.Data[field].(string)
	require.Truef(t, ok, "field %q missing or not a string in body: %s", field, body)
	return s
}
