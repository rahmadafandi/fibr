// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationVersions returns the sorted timestamp-version prefixes of every
// generated migration file (the digits before the first underscore).
func migrationVersions(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "internal/migrations", "*_create_*.go"))
	require.NoError(t, err)
	var versions []string
	for _, m := range matches {
		base := filepath.Base(m)
		i := strings.IndexByte(base, '_')
		require.Positive(t, i, "migration filename %q has no version prefix", base)
		versions = append(versions, base[:i])
	}
	return versions
}

func TestMigrationTimestampsDistinct(t *testing.T) {
	fixed := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	orig := now
	now = func() time.Time { return fixed }
	defer func() { now = orig }()

	cases := []struct {
		name  string
		opts  Options
		count int
	}{
		{"sample+auth", Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: "ddd", Sample: true, Auth: true}, 2},
		{"sample+team", Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: "ddd", Sample: true, Team: true}, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts.Dir = filepath.Join(t.TempDir(), "app")
			tc.opts.NoGit, tc.opts.NoTidy = true, true
			require.NoError(t, Generate(tc.opts, &strings.Builder{}))

			versions := migrationVersions(t, tc.opts.Dir)
			require.Len(t, versions, tc.count)

			seen := make(map[string]bool, len(versions))
			for _, v := range versions {
				require.False(t, seen[v], "duplicate migration version %q in %v", v, versions)
				seen[v] = true
			}
		})
	}
}

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
	assert.Contains(t, string(env), "AUTO_MIGRATE=false")
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
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "migrate.NewCommand")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), `Use:   "serve"`)
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/config/config.go"), "AutoMigrate")
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
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "migrate.NewCommand")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), `Use:   "serve"`)
	assertFileContains(t, filepath.Join(dir, "internal/config/config.go"), "AutoMigrate")
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

func globOne(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, matches, 1, "expected exactly one match for %s, got %v", pattern, matches)
	return matches[0]
}

func TestGenerateAlwaysHasMigrationsPkg(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd"})
	assertFileContains(t, filepath.Join(dir, "internal/migrations/migrations.go"), "var Migrations = migrate.NewMigrations")
}

func TestGenerateSampleEmitsUserMigration(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd", Sample: true})
	m := globOne(t, filepath.Join(dir, "internal/migrations/*_create_users.go"))
	assertFileContains(t, m, "Migrations.MustRegister")
	assertFileContains(t, m, `bun:"table:users"`)
}

func TestGenerateAuthSecretAndConfig(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd", Auth: true})
	env, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	require.NoError(t, err)
	line := ""
	for _, l := range strings.Split(string(env), "\n") {
		if strings.HasPrefix(l, "JWT_SECRET=") {
			line = strings.TrimPrefix(l, "JWT_SECRET=")
		}
	}
	require.Len(t, line, 64, "JWT_SECRET should be 32-byte hex")
	for _, ch := range line {
		assert.True(t, (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'), "hex only")
	}
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/config/config.go"), `mapstructure:"JWT_SECRET" required:"true"`)
}

func TestGenerateNoAuthNoSecret(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd"})
	env, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	require.NoError(t, err)
	assert.NotContains(t, string(env), "JWT_SECRET")
	cfg, err := os.ReadFile(filepath.Join(dir, "internal/infrastructure/config/config.go"))
	require.NoError(t, err)
	assert.NotContains(t, string(cfg), "JWTSecret")
}

func TestGenerateAuthDDD(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd", Auth: true})
	assertFileContains(t, filepath.Join(dir, "internal/domain/account/account.go"), "type Account struct")
	assertFileContains(t, filepath.Join(dir, "internal/domain/account/repository.go"), "FindByEmail")
	assertFileContains(t, filepath.Join(dir, "internal/application/account/service.go"), "func (s *Service) Register")
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/persistence/account_repository_bun.go"), "func NewAccountRepository")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/auth_handler.go"), "/auth")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/auth_module.go"), "func NewAuthModule(db *bun.DB, secret string, store auth.TokenStore) bootstrap.Module")
	m := globOne(t, filepath.Join(dir, "internal/migrations/*_create_accounts.go"))
	assertFileContains(t, m, `bun:"table:accounts"`)
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "httpiface.NewAuthModule(db, cfg.JWTSecret, authStore)")
}

func TestGenerateAuthLayered(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "layered", Auth: true})
	assertFileContains(t, filepath.Join(dir, "internal/model/account.go"), "type Account struct")
	assertFileContains(t, filepath.Join(dir, "internal/repository/account_repo.go"), "func NewAccountRepository")
	assertFileContains(t, filepath.Join(dir, "internal/service/auth_service.go"), "func (s *AuthService) Register")
	assertFileContains(t, filepath.Join(dir, "internal/handler/auth_handler.go"), "/auth")
	assertFileContains(t, filepath.Join(dir, "internal/handler/auth_module.go"), "func NewAuthModule(db *bun.DB, secret string, store auth.TokenStore) bootstrap.Module")
	m := globOne(t, filepath.Join(dir, "internal/migrations/*_create_accounts.go"))
	assertFileContains(t, m, `bun:"table:accounts"`)
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "handler.NewAuthModule(db, cfg.JWTSecret, authStore)")
}

func TestGenerateAuthAndSampleCoexist(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "ddd", Auth: true, Sample: true})
	assertFileContains(t, filepath.Join(dir, "internal/domain/user/user.go"), "type User struct")
	assertFileContains(t, filepath.Join(dir, "internal/domain/account/account.go"), "type Account struct")
	_ = globOne(t, filepath.Join(dir, "internal/migrations/*_create_users.go"))
	_ = globOne(t, filepath.Join(dir, "internal/migrations/*_create_accounts.go"))
}

func TestGenerateAuthAndSampleCoexistLayered(t *testing.T) {
	dir := generateInto(t, Options{Name: "app", Module: "github.com/me/app", DB: "sqlite", Layout: "layered", Auth: true, Sample: true})
	// sample user files
	assertFileContains(t, filepath.Join(dir, "internal/model/user.go"), "type User struct")
	assertFileContains(t, filepath.Join(dir, "internal/handler/user_module.go"), "func NewUserModule")
	// auth account files (same packages, distinct names)
	assertFileContains(t, filepath.Join(dir, "internal/model/account.go"), "type Account struct")
	assertFileContains(t, filepath.Join(dir, "internal/handler/auth_module.go"), "func NewAuthModule")
	// both migrations
	_ = globOne(t, filepath.Join(dir, "internal/migrations/*_create_users.go"))
	_ = globOne(t, filepath.Join(dir, "internal/migrations/*_create_accounts.go"))
	// main mounts both
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "handler.NewUserModule(db)")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "handler.NewAuthModule(db, cfg.JWTSecret, authStore)")
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

func TestAuthScaffoldRefreshArtifacts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Auth: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	handler := read("internal/interface/http/auth_handler.go")
	require.Contains(t, handler, `g.Post("/refresh", h.refresh)`)
	require.Contains(t, handler, `g.Post("/logout"`)
	require.Contains(t, handler, "auth.WithBlocklist(h.store)")
	require.Contains(t, handler, "h.issuer.Issue(")
	require.Contains(t, handler, "h.issuer.Refresh(")
	require.Contains(t, handler, "h.issuer.Logout(")

	module := read("internal/interface/http/auth_module.go")
	require.Contains(t, module, "store auth.TokenStore")

	cfg := read("internal/infrastructure/config/config.go")
	require.Contains(t, cfg, "RedisURL")
	require.Contains(t, cfg, `mapstructure:"REDIS_URL"`)
	require.NotContains(t, cfg, `mapstructure:"REDIS_URL" required`)

	mainGo := read("cmd/api/main.go")
	require.Contains(t, mainGo, "NewMemoryStore()")
	require.Contains(t, mainGo, "NewRedisStore(")
	require.Contains(t, mainGo, "NewAuthModule(db, cfg.JWTSecret, authStore)")

	env := read(".env.example")
	require.Contains(t, env, "# REDIS_URL=redis://localhost:6379/0")
}

func TestTeamImpliesAuthAndArtifacts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	// Team implies auth: the auth handler + accounts migration exist.
	handler := read("internal/interface/http/auth_handler.go")
	require.Contains(t, handler, `g.Post("/switch-team"`)
	require.Contains(t, handler, `t.Post("/:id/members", auth.RequireScope("member:manage")`)
	require.Contains(t, handler, "h.teams.CreatePersonalTeam(")

	// Team entities + service exist.
	require.Contains(t, read("internal/domain/team/team.go"), "type Membership struct")
	require.Contains(t, read("internal/application/team/service.go"), "defaultRoles")

	// Migrations for teams + memberships exist (find by suffix).
	migs, err := filepath.Glob(filepath.Join(dir, "internal/migrations/*.go"))
	require.NoError(t, err)
	var names string
	for _, m := range migs {
		names += filepath.Base(m) + "\n"
	}
	require.Contains(t, names, "_create_accounts.go")
	require.Contains(t, names, "_create_teams.go")
	require.Contains(t, names, "_create_memberships.go")
}

func TestNonTeamAuthHasNoTeamFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Auth: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))
	_, err := os.Stat(filepath.Join(dir, "internal/domain/team/team.go"))
	require.True(t, os.IsNotExist(err), "non-team --auth must not emit team files")
}

func TestTeamRoleMigrationsEmitted(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	migs, err := filepath.Glob(filepath.Join(dir, "internal/migrations/*.go"))
	require.NoError(t, err)
	var names string
	for _, m := range migs {
		names += filepath.Base(m) + "\n"
	}
	require.Contains(t, names, "_create_roles.go")
	require.Contains(t, names, "_create_role_permissions.go")
}

func TestTeamRoleScaffoldArtifacts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	svc := read("internal/application/team/service.go")
	require.Contains(t, svc, "var AllPermissions")
	require.Contains(t, svc, "var defaultRoles")
	require.Contains(t, svc, "func (s *Service) PermissionsForRole(")
	require.Contains(t, svc, "func (s *Service) CreateRole(")
	require.NotContains(t, svc, "func PermissionsFor(role string)") // static map removed

	handler := read("internal/interface/http/auth_handler.go")
	require.Contains(t, handler, `t.Post("/:id/roles", auth.RequireScope("role:manage")`)
	require.Contains(t, handler, `r.Get("/permissions"`)
	require.Contains(t, handler, `t.Put("/:id/members", auth.RequireScope("member:manage")`)

	require.Contains(t, read("internal/domain/team/team.go"), "type Role struct")
}

func TestTeamInvitationScaffoldArtifactsDDD(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	// Entity + repository port.
	entity := read("internal/domain/team/team.go")
	require.Contains(t, entity, "type Invitation struct")
	repo := read("internal/domain/team/repository.go")
	require.Contains(t, repo, "CreateInvitation(ctx context.Context, inv *Invitation) error")
	require.Contains(t, repo, "FindInvitationByJTI(ctx context.Context, jti string) (*Invitation, error)")

	// Persistence: model + MigrateTeam extension.
	pers := read("internal/infrastructure/persistence/team_repository_bun.go")
	require.Contains(t, pers, "type invitationModel struct")
	require.Contains(t, pers, "invitations_jti_uq")

	// Service: secret-bound, sentinels, use cases.
	svc := read("internal/application/team/service.go")
	require.Contains(t, svc, "ErrInviteInvalid")
	require.Contains(t, svc, "func (s *Service) Invite(")
	require.Contains(t, svc, "func (s *Service) AcceptInvitation(")
	require.Contains(t, svc, "func NewService(repo domain.Repository, accounts accountdomain.Repository, secret string) *Service")

	// Handler: routes.
	handler := read("internal/interface/http/auth_handler.go")
	require.Contains(t, handler, `t.Post("/:id/invitations", auth.RequireScope("member:manage")`)
	require.Contains(t, handler, `r.Post("/invitations/accept"`)

	// Module: NewService passes secret.
	mod := read("internal/interface/http/auth_module.go")
	require.Contains(t, mod, "appteam.NewService(teamRepo, accountRepo, secret)")

	// Migration emitted.
	migs, err := filepath.Glob(filepath.Join(dir, "internal/migrations/*.go"))
	require.NoError(t, err)
	var names string
	for _, m := range migs {
		names += filepath.Base(m) + "\n"
	}
	require.Contains(t, names, "_create_invitations.go")
}

func TestTeamInvitationScaffoldArtifactsLayered(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "layered", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	require.Contains(t, read("internal/model/team.go"), "type Invitation struct")
	repo := read("internal/repository/team_repo.go")
	require.Contains(t, repo, "func (r *TeamRepository) CreateInvitation(")
	require.Contains(t, repo, "invitations_jti_uq")

	svc := read("internal/service/team_service.go")
	require.Contains(t, svc, "ErrInviteInvalid")
	require.Contains(t, svc, "func (s *TeamService) AcceptInvitation(")
	require.Contains(t, svc, "func NewTeamService(repo *repository.TeamRepository, accounts *repository.AccountRepository, secret string) *TeamService")

	handler := read("internal/handler/auth_handler.go")
	require.Contains(t, handler, `r.Post("/invitations/accept"`)
	require.Contains(t, read("internal/handler/auth_module.go"), "service.NewTeamService(teamRepo, accountRepo, secret)")
}

func TestTeamAdminScaffoldArtifactsDDD(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "ddd", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	handler := read("internal/interface/http/auth_handler.go")
	require.Contains(t, handler, `t.Patch("/:id", auth.RequireScope("team:manage")`)
	require.Contains(t, handler, `t.Delete("/:id", auth.RequireScope("team:manage")`)
	require.Contains(t, handler, `t.Post("/:id/transfer", auth.RequireScope("team:manage")`)
	require.Contains(t, handler, `t.Delete("/:id/members", auth.RequireScope("member:manage")`)
	require.Contains(t, handler, `t.Delete("/:id/members/me"`)

	svc := read("internal/application/team/service.go")
	require.Contains(t, svc, "func (s *Service) RenameTeam(")
	require.Contains(t, svc, "func (s *Service) DeleteTeam(")
	require.Contains(t, svc, "func (s *Service) TransferOwnership(")
	require.Contains(t, svc, "func (s *Service) RemoveMember(")
	require.Contains(t, svc, "func (s *Service) LeaveTeam(")
	require.Contains(t, svc, "ErrNotOwner")
	require.Contains(t, svc, "ErrLastTeam")

	repo := read("internal/domain/team/repository.go")
	require.Contains(t, repo, "FindTeam(ctx context.Context, teamID int64) (*Team, error)")
	require.Contains(t, repo, "DeleteTeamCascade(ctx context.Context, teamID int64) error")
	require.Contains(t, repo, "TransferOwnership(ctx context.Context, teamID, newOwnerID, oldOwnerID int64) error")

	pers := read("internal/infrastructure/persistence/team_repository_bun.go")
	require.Contains(t, pers, "func (r *TeamRepository) DeleteTeamCascade(")
	require.Contains(t, pers, "func (r *TeamRepository) FindTeam(")
}

func TestTeamAdminScaffoldArtifactsLayered(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	require.NoError(t, Generate(Options{
		Name: "app", Module: "example.com/app",
		DB: "sqlite", Layout: "layered", Team: true,
		Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
	}, &strings.Builder{}))

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		return string(b)
	}

	handler := read("internal/handler/auth_handler.go")
	require.Contains(t, handler, `t.Patch("/:id", auth.RequireScope("team:manage")`)
	require.Contains(t, handler, `t.Delete("/:id/members/me"`)
	require.Contains(t, handler, `t.Post("/:id/transfer", auth.RequireScope("team:manage")`)

	svc := read("internal/service/team_service.go")
	require.Contains(t, svc, "func (s *TeamService) TransferOwnership(")
	require.Contains(t, svc, "func (s *TeamService) DeleteTeam(")
	require.Contains(t, svc, "ErrOwnerRemoval")

	repo := read("internal/repository/team_repo.go")
	require.Contains(t, repo, "func (r *TeamRepository) DeleteTeamCascade(")
	require.Contains(t, repo, "func (r *TeamRepository) FindTeam(")
}

func TestMetricsScaffoldWiring(t *testing.T) {
	for _, layout := range []string{"ddd", "layered"} {
		t.Run(layout, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			require.NoError(t, Generate(Options{
				Name: "app", Module: "example.com/app",
				DB: "sqlite", Layout: layout,
				Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
			}, &strings.Builder{}))

			read := func(rel string) string {
				b, err := os.ReadFile(filepath.Join(dir, rel))
				require.NoError(t, err)
				return string(b)
			}

			configPath := "internal/config/config.go"
			if layout == "ddd" {
				configPath = "internal/infrastructure/config/config.go"
			}

			cfg := read(configPath)
			require.Contains(t, cfg, "MetricsEnabled bool")
			require.Contains(t, cfg, "METRICS_ENABLED")

			main := read("cmd/api/main.go")
			require.Contains(t, main, "Metrics:")
			require.Contains(t, main, "cfg.MetricsEnabled")

			require.Contains(t, read(".env.example"), "METRICS_ENABLED")
		})
	}
}

func TestTracingScaffoldWiring(t *testing.T) {
	for _, layout := range []string{"ddd", "layered"} {
		t.Run(layout, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "app")
			require.NoError(t, Generate(Options{
				Name: "app", Module: "example.com/app",
				DB: "sqlite", Layout: layout,
				Dir: dir, NoGit: true, NoTidy: true, Local: repoRoot(t),
			}, &strings.Builder{}))

			read := func(rel string) string {
				b, err := os.ReadFile(filepath.Join(dir, rel))
				require.NoError(t, err)
				return string(b)
			}

			configPath := "internal/config/config.go"
			if layout == "ddd" {
				configPath = "internal/infrastructure/config/config.go"
			}

			cfg := read(configPath)
			require.Contains(t, cfg, "TracingEnabled bool")
			require.Contains(t, cfg, "TRACING_ENABLED")

			main := read("cmd/api/main.go")
			require.Contains(t, main, "tracing.Setup(")
			require.Contains(t, main, "Tracing:")
			require.Contains(t, main, "cfg.TracingEnabled")
			require.Contains(t, main, "Cleanup:")

			env := read(".env.example")
			require.Contains(t, env, "TRACING_ENABLED")
			require.Contains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT")
		})
	}
}

func TestMailerScaffoldWiring(t *testing.T) {
	t.Run("mailer-only-ddd", func(t *testing.T) {
		dir := generateInto(t, Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: "ddd", Mailer: true})
		assertFileContains(t, filepath.Join(dir, "internal/infrastructure/config/config.go"), "SMTP_FROM")
		assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "fhmailer.New")
		assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), `app.Post("/email/test"`)
		assertFileContains(t, filepath.Join(dir, ".env.example"), "SMTP_PORT=587")
	})

	t.Run("mailer-queue-layered", func(t *testing.T) {
		dir := generateInto(t, Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: "layered", Mailer: true, Queue: true})
		jobs := filepath.Join(dir, "internal/jobs/jobs.go")
		assertFileContains(t, jobs, "type QueueSender struct")
		assertFileContains(t, jobs, "func RegisterEmail(")
		assertFileContains(t, jobs, `const EmailType = "email:send"`)
		assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "appjobs.NewQueueSender(queueClient)")
		assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "appjobs.RegisterEmail(srv, base)")
	})

	t.Run("mailer-team-ddd", func(t *testing.T) {
		dir := generateInto(t, Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: "ddd", Team: true, Auth: true, Mailer: true})
		assertFileContains(t, filepath.Join(dir, "internal/application/team/service.go"), "sender   fhmailer.Sender")
		assertFileContains(t, filepath.Join(dir, "internal/application/team/service.go"), "You're invited")
		assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "NewAuthModule(db, cfg.JWTSecret, authStore, mailSender)")
	})
}

func TestQueueScaffoldWiring(t *testing.T) {
	for _, layout := range []string{"ddd", "layered"} {
		t.Run(layout, func(t *testing.T) {
			dir := generateInto(t, Options{Name: "app", Module: "example.com/app", DB: "sqlite", Layout: layout, Queue: true})

			jobsPath := filepath.Join(dir, "internal/jobs/jobs.go")
			cfgPath := filepath.Join(dir, "internal/config/config.go")
			if layout == "ddd" {
				jobsPath = filepath.Join(dir, "internal/application/jobs/jobs.go")
				cfgPath = filepath.Join(dir, "internal/infrastructure/config/config.go")
			}

			assertFileContains(t, jobsPath, "func Register(srv *fhjobs.Server)")
			assertFileContains(t, jobsPath, "WelcomeType")

			main := filepath.Join(dir, "cmd/api/main.go")
			assertFileContains(t, main, "func workerCmd()")
			assertFileContains(t, main, "Asynqmon:")
			assertFileContains(t, main, "fhjobs.NewClient")
			assertFileContains(t, main, `app.Post("/jobs/welcome"`)

			assertFileContains(t, cfgPath, "QUEUE_CONCURRENCY")
			assertFileContains(t, cfgPath, "ASYNQMON_PATH")
			assertFileContains(t, filepath.Join(dir, ".env.example"), "QUEUE_CONCURRENCY=10")
		})
	}
}
