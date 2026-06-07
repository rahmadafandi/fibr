// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestTeamFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the team runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "team.db")
	port := "39519"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("c", 64),
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

	// owner registers (personal team auto-created) + logs in
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"owner@b.com","password":"secret123"}`, "")))
	_, body := postJSON(t, base+"/auth/login", `{"email":"owner@b.com","password":"secret123"}`, "")
	ownerTok := dataField(t, body, "access_token")
	require.NotEmpty(t, ownerTok)

	// /me shows an active team + owner role
	_, meBody := postGet(t, base+"/auth/me", ownerTok)
	team1 := dataField(t, meBody, "team")
	require.NotEmpty(t, team1)
	require.Equal(t, "owner", dataField(t, meBody, "role"))

	// owner creates a second team, then lists teams (2)
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/", `{"name":"Acme"}`, ownerTok)))
	require.Equal(t, 200, getCode(t, base+"/teams/", ownerTok))

	// a member registers
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"member@b.com","password":"secret123"}`, "")))

	// owner adds the member to the active (personal) team as "member"
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/"+team1+"/members", `{"email":"member@b.com","role":"member"}`, ownerTok)))

	// owner can reach the team:manage route for the active team; ...
	require.Equal(t, 200, getCode(t, base+"/teams/"+team1+"/manage", ownerTok))

	// ... the member cannot (role "member" lacks team:manage) after switching into it
	_, mBody := postJSON(t, base+"/auth/login", `{"email":"member@b.com","password":"secret123"}`, "")
	memberTok := dataField(t, mBody, "access_token")
	swCode, swBody := postJSON(t, base+"/auth/switch-team", `{"team_id":`+team1+`}`, memberTok)
	require.Equal(t, 200, swCode)
	memberTeamTok := dataField(t, swBody, "access_token")
	require.Equal(t, 403, getCode(t, base+"/teams/"+team1+"/manage", memberTeamTok))
}

// mustCode returns just the status code from a postJSON result tuple.
func mustCode(code int, _ string) int { return code }

// postGet issues a GET with a bearer and returns code + body.
func postGet(t *testing.T, url, bearer string) (int, string) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
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

func TestTeamRolesFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the team-roles runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "roles.db")
	port := "39520"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("d", 64),
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

	// owner registers + logs in
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"owner@b.com","password":"secret123"}`, "")))
	_, body := postJSON(t, base+"/auth/login", `{"email":"owner@b.com","password":"secret123"}`, "")
	ownerTok := dataField(t, body, "access_token")
	require.NotEmpty(t, ownerTok)
	team1 := dataFieldFrom(t, base+"/auth/me", ownerTok, "team")
	require.NotEmpty(t, team1)

	// permission catalog
	require.Equal(t, 200, getCode(t, base+"/permissions", ownerTok))

	// owner lists seeded roles (owner/admin/member/viewer)
	require.Equal(t, 200, getCode(t, base+"/teams/"+team1+"/roles", ownerTok))

	// create a custom "editor" role
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/"+team1+"/roles", `{"name":"editor","permissions":["post:read","post:write"]}`, ownerTok)))
	// invalid permission -> 400
	require.Equal(t, 400, mustCode(postJSON(t, base+"/teams/"+team1+"/roles", `{"name":"bad","permissions":["does:not:exist"]}`, ownerTok)))

	// register a member, add to team, then set role to editor
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"member@b.com","password":"secret123"}`, "")))
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/"+team1+"/members", `{"email":"member@b.com","role":"member"}`, ownerTok)))
	require.Equal(t, 200, mustCode(postJSONMethod(t, "PUT", base+"/teams/"+team1+"/members", `{"email":"member@b.com","role":"editor"}`, ownerTok)))

	// member logs in, switches into the shared team, checks permissions
	_, mBody := postJSON(t, base+"/auth/login", `{"email":"member@b.com","password":"secret123"}`, "")
	memberTok := dataField(t, mBody, "access_token")
	_, swBody := postJSON(t, base+"/auth/switch-team", `{"team_id":`+team1+`}`, memberTok)
	memberTeamTok := dataField(t, swBody, "access_token")

	_, meBody := postGet(t, base+"/auth/me", memberTeamTok)
	require.Contains(t, meBody, "post:write")                                        // editor has it
	require.NotContains(t, meBody, "team:manage")                                    // editor lacks it
	require.Equal(t, 403, getCode(t, base+"/teams/"+team1+"/manage", memberTeamTok)) // lacks team:manage
	require.Equal(t, 200, getCode(t, base+"/teams/"+team1+"/manage", ownerTok))      // owner has it

	// delete-role guards: owner protected (400); editor in use (409)
	require.Equal(t, 400, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+team1+"/roles/owner", "", ownerTok)))
	require.Equal(t, 409, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+team1+"/roles/editor", "", ownerTok)))
}

// dataFieldFrom GETs url with a bearer and returns a string field under data.
func dataFieldFrom(t *testing.T, url, bearer, field string) string {
	t.Helper()
	_, body := postGet(t, url, bearer)
	return dataField(t, body, field)
}

// postJSONMethod issues an arbitrary-method JSON request with a bearer and
// returns status + body.
func postJSONMethod(t *testing.T, method, url, body, bearer string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
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

// dataFieldNum reads a numeric field under data and returns it as a string id.
func dataFieldNum(t *testing.T, body, field string) string {
	t.Helper()
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &parsed))
	f, ok := parsed.Data[field].(float64)
	require.Truef(t, ok, "field %q missing or not a number in body: %s", field, body)
	return strconv.FormatInt(int64(f), 10)
}

func TestTeamInvitationsFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the team-invitations runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "invites.db")
	port := "39521"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("d", 64),
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

	// owner registers + logs in; capture active team.
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"owner@b.com","password":"secret123"}`, "")))
	_, body := postJSON(t, base+"/auth/login", `{"email":"owner@b.com","password":"secret123"}`, "")
	ownerTok := dataField(t, body, "access_token")
	team1 := dataFieldFrom(t, base+"/auth/me", ownerTok, "team")
	require.NotEmpty(t, team1)

	// owner invites invitee@b.com as "member" -> 201, capture token.
	icode, ibody := postJSON(t, base+"/teams/"+team1+"/invitations", `{"email":"invitee@b.com","role":"member"}`, ownerTok)
	require.Equal(t, 201, icode)
	inviteTok := dataField(t, ibody, "token")
	require.NotEmpty(t, inviteTok)

	// unknown role -> 404.
	require.Equal(t, 404, mustCode(postJSON(t, base+"/teams/"+team1+"/invitations", `{"email":"x@b.com","role":"nope"}`, ownerTok)))

	// invitee and a third account register + log in.
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"invitee@b.com","password":"secret123"}`, "")))
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"third@b.com","password":"secret123"}`, "")))
	_, tbody := postJSON(t, base+"/auth/login", `{"email":"third@b.com","password":"secret123"}`, "")
	thirdTok := dataField(t, tbody, "access_token")

	// wrong account accepting -> 403 (email mismatch).
	require.Equal(t, 403, mustCode(postJSON(t, base+"/invitations/accept", `{"token":"`+inviteTok+`"}`, thirdTok)))

	// invitee accepts -> 200; their /teams now lists the shared team.
	_, mbody := postJSON(t, base+"/auth/login", `{"email":"invitee@b.com","password":"secret123"}`, "")
	inviteeTok := dataField(t, mbody, "access_token")
	require.Equal(t, 200, mustCode(postJSON(t, base+"/invitations/accept", `{"token":"`+inviteTok+`"}`, inviteeTok)))
	require.Equal(t, 200, getCode(t, base+"/teams/", inviteeTok))

	// after switching into the shared team, /me shows the "member" role.
	_, swbody := postJSON(t, base+"/auth/switch-team", `{"team_id":`+team1+`}`, inviteeTok)
	inviteeTeamTok := dataField(t, swbody, "access_token")
	_, meBody := postGet(t, base+"/auth/me", inviteeTeamTok)
	require.Equal(t, "member", dataField(t, meBody, "role"))

	// replay: accepting the same token again -> 409 (not pending / already member).
	require.Equal(t, 409, mustCode(postJSON(t, base+"/invitations/accept", `{"token":"`+inviteTok+`"}`, inviteeTok)))

	// revoke path: invite revoked@b.com, capture id, DELETE it, then accept -> 409.
	_, rbody := postJSON(t, base+"/teams/"+team1+"/invitations", `{"email":"revoked@b.com","role":"member"}`, ownerTok)
	revokeTok := dataField(t, rbody, "token")
	revokeID := dataFieldNum(t, rbody, "id")
	require.Equal(t, 200, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+team1+"/invitations/"+revokeID, "", ownerTok)))
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"revoked@b.com","password":"secret123"}`, "")))
	_, vbody := postJSON(t, base+"/auth/login", `{"email":"revoked@b.com","password":"secret123"}`, "")
	revokedTok := dataField(t, vbody, "access_token")
	require.Equal(t, 409, mustCode(postJSON(t, base+"/invitations/accept", `{"token":"`+revokeTok+`"}`, revokedTok)))

	// listing shows only pending invites (accepted + revoked excluded).
	_, lbody := postGet(t, base+"/teams/"+team1+"/invitations", ownerTok)
	require.NotContains(t, lbody, "invitee@b.com")
	require.NotContains(t, lbody, "revoked@b.com")
}

func TestTeamAdminFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the team-admin runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "admin.db")
	port := "39522"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"JWT_SECRET="+strings.Repeat("e", 64),
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

	// owner registers (personal team A) + logs in.
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"owner@b.com","password":"secret123"}`, "")))
	_, body := postJSON(t, base+"/auth/login", `{"email":"owner@b.com","password":"secret123"}`, "")
	ownerTok := dataField(t, body, "access_token")
	teamA := dataFieldFrom(t, base+"/auth/me", ownerTok, "team")

	// owner creates a second team B (so deletes/leaves have a non-last team).
	_, cbody := postJSON(t, base+"/teams/", `{"name":"Acme"}`, ownerTok)
	teamB := dataFieldNum(t, cbody, "id")

	// rename A -> "Renamed"; GET /teams reflects it.
	require.Equal(t, 200, mustCode(postJSONMethod(t, "PATCH", base+"/teams/"+teamA, `{"name":"Renamed"}`, ownerTok)))
	_, listBody := postGet(t, base+"/teams/", ownerTok)
	require.Contains(t, listBody, "Renamed")

	// add member@b.com to A; remove; re-add (proves removal worked).
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"member@b.com","password":"secret123"}`, "")))
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/"+teamA+"/members", `{"email":"member@b.com","role":"member"}`, ownerTok)))
	require.Equal(t, 200, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA+"/members", `{"email":"member@b.com"}`, ownerTok)))
	require.Equal(t, 201, mustCode(postJSON(t, base+"/teams/"+teamA+"/members", `{"email":"member@b.com","role":"member"}`, ownerTok)))

	// owner cannot remove self via /members.
	require.Equal(t, 403, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA+"/members", `{"email":"owner@b.com"}`, ownerTok)))

	// promote member -> admin (admin role has member:manage); member switches into A.
	require.Equal(t, 200, mustCode(postJSONMethod(t, "PUT", base+"/teams/"+teamA+"/members", `{"email":"member@b.com","role":"admin"}`, ownerTok)))
	_, mbody := postJSON(t, base+"/auth/login", `{"email":"member@b.com","password":"secret123"}`, "")
	memberTok := dataField(t, mbody, "access_token")
	_, swbody := postJSON(t, base+"/auth/switch-team", `{"team_id":`+teamA+`}`, memberTok)
	memberAdminTok := dataField(t, swbody, "access_token")

	// admin cannot remove the owner.
	require.Equal(t, 409, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA+"/members", `{"email":"owner@b.com"}`, memberAdminTok)))

	// owner transfers A to member -> 200.
	require.Equal(t, 200, mustCode(postJSON(t, base+"/teams/"+teamA+"/transfer", `{"email":"member@b.com"}`, ownerTok)))

	// old owner (now demoted to admin) can no longer delete A -> 403 (not owner).
	require.Equal(t, 403, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA, "", ownerTok)))

	// member re-switches into A to pick up owner scopes, then transfers back.
	_, sw2 := postJSON(t, base+"/auth/switch-team", `{"team_id":`+teamA+`}`, memberTok)
	memberOwnerTok := dataField(t, sw2, "access_token")
	require.Equal(t, 200, mustCode(postJSON(t, base+"/teams/"+teamA+"/transfer", `{"email":"owner@b.com"}`, memberOwnerTok)))

	// member (now admin again) leaves A voluntarily -> 200.
	_, sw3 := postJSON(t, base+"/auth/switch-team", `{"team_id":`+teamA+`}`, memberTok)
	memberBackTok := dataField(t, sw3, "access_token")
	require.Equal(t, 200, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA+"/members/me", "", memberBackTok)))

	// owner cannot leave A (owner must transfer first) -> 409.
	require.Equal(t, 409, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamA+"/members/me", "", ownerTok)))

	// last-team guard: a fresh solo account cannot delete its only team -> 409.
	require.Equal(t, 201, mustCode(postJSON(t, base+"/auth/register", `{"email":"solo@b.com","password":"secret123"}`, "")))
	_, vbody := postJSON(t, base+"/auth/login", `{"email":"solo@b.com","password":"secret123"}`, "")
	soloTok := dataField(t, vbody, "access_token")
	soloTeam := dataFieldFrom(t, base+"/auth/me", soloTok, "team")
	require.Equal(t, 409, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+soloTeam, "", soloTok)))

	// delete success: owner switches into B and deletes it (cascade) -> 200; gone from list.
	_, sw4 := postJSON(t, base+"/auth/switch-team", `{"team_id":`+teamB+`}`, ownerTok)
	ownerBTok := dataField(t, sw4, "access_token")
	require.Equal(t, 200, mustCode(postJSONMethod(t, "DELETE", base+"/teams/"+teamB, "", ownerBTok)))
	_, finalList := postGet(t, base+"/teams/", ownerTok)
	require.NotContains(t, finalList, "Acme")
}

func TestMetricsE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the metrics runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Sample: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "metrics.db")
	port := "39523"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"METRICS_ENABLED=true",
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

	require.Equal(t, 200, getCode(t, base+"/livez", ""))
	_, body := postGet(t, base+"/metrics", "")
	require.Contains(t, body, "http_requests_total")
	require.Contains(t, body, "go_goroutines")
}

func TestTracingE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the tracing runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Sample: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "tracing.db")
	port := "39524"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
		"TRACING_ENABLED=true",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4999",
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

	require.Equal(t, 200, getCode(t, base+"/livez", ""))
}

func TestQueueCompilesE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the queue compile test (slow: go build)")
	}
	root := repoRoot(t)
	cases := []struct {
		layout string
		auth   bool
	}{
		{"ddd", false},
		{"layered", false},
		{"ddd", true},
		{"layered", true},
	}
	for _, tc := range cases {
		name := tc.layout
		if tc.auth {
			name += "-auth"
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			require.NoError(t, Generate(Options{
				Name: "app", Module: "example.com/app",
				DB: "sqlite", Layout: tc.layout, Queue: true, Auth: tc.auth,
				Dir: dir, NoGit: true, NoTidy: false, Local: root,
			}, &strings.Builder{}))
			build := exec.Command("go", "build", "./...")
			build.Dir = dir
			out, err := build.CombinedOutput()
			require.NoError(t, err, "go build failed:\n%s", out)
		})
	}
}

func TestMailerCompilesE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the mailer compile test (slow: go build)")
	}
	root := repoRoot(t)
	cases := []struct {
		name string
		opts Options
	}{
		{"mailer", Options{Mailer: true}},
		{"mailer-queue", Options{Mailer: true, Queue: true}},
		{"mailer-team", Options{Mailer: true, Team: true}},
		{"mailer-team-queue", Options{Mailer: true, Team: true, Queue: true}},
	}
	for _, layout := range []string{"ddd", "layered"} {
		for _, tc := range cases {
			t.Run(layout+"-"+tc.name, func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "app")
				o := tc.opts
				o.Name, o.Module, o.DB, o.Layout = "app", "example.com/app", "sqlite", layout
				o.Dir, o.NoGit, o.NoTidy, o.Local = dir, true, false, root
				require.NoError(t, Generate(o, &strings.Builder{}))
				build := exec.Command("go", "build", "./...")
				build.Dir = dir
				out, err := build.CombinedOutput()
				require.NoError(t, err, "go build failed:\n%s", out)
			})
		}
	}
}

func TestMailerFlowE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run the mailer runtime test (slow: go run)")
	}
	root := repoRoot(t)
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Mailer: true,
		Dir: dir, NoGit: true, NoTidy: false, Local: root,
	}, &strings.Builder{}))

	dbPath := filepath.Join(t.TempDir(), "mail.db")
	port := "39526"
	env := append(os.Environ(),
		"DATABASE_URL=file:"+dbPath+"?cache=shared",
		"PORT="+port,
	)

	srv := exec.Command("go", "run", "./cmd/api")
	srv.Dir, srv.Env = dir, env
	srv.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, srv.Start())
	defer func() { _ = syscall.Kill(-srv.Process.Pid, syscall.SIGKILL) }()

	base := "http://127.0.0.1:" + port
	waitReady(t, base+"/livez")

	// No SMTP_HOST set -> LogSender -> send succeeds and logs.
	code, _ := postJSON(t, base+"/email/test", `{}`, "")
	require.Equal(t, 200, code)
}
