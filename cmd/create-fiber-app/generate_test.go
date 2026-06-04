// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
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
	assert.NotContains(t, ddd, "internal/domain/user/user.go")

	dddSample := dests(Data{Layout: "ddd", Sample: true})
	assert.Contains(t, dddSample, "internal/domain/user/user.go")

	layered := dests(Data{Layout: "layered", Sample: true})
	assert.Contains(t, layered, "internal/config/config.go")
	assert.Contains(t, layered, "internal/handler/user_handler.go")
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
	assertFileContains(t, filepath.Join(dir, "internal/domain/user/repository.go"), "type Repository interface")
	assertFileContains(t, filepath.Join(dir, "internal/infrastructure/persistence/user_repository_bun.go"), "func NewUserRepository")
	assertFileContains(t, filepath.Join(dir, "internal/interface/http/user_handler.go"), "/users")
	assertFileContains(t, filepath.Join(dir, "cmd/api/main.go"), "persistence.NewUserRepository")
}
