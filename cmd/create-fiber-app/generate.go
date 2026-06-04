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
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Data is the render context passed to every template.
type Data struct {
	Name           string
	Module         string
	DB             string
	Layout         string
	Sample         bool
	HelpersVersion string
	LocalReplace   string
}

type fileSpec struct {
	tmpl string // path under templates/
	dest string // path relative to the output directory
}

// plan returns the list of templates to render for the given options.
func plan(d Data) []fileSpec {
	specs := []fileSpec{
		{"common/gomod.tmpl", "go.mod"},
		{"common/gitignore.tmpl", ".gitignore"},
		{"common/env.tmpl", ".env.example"},
		{"common/dockerfile.tmpl", "Dockerfile"},
		{"common/compose.tmpl", "docker-compose.yml"},
		{"common/makefile.tmpl", "Makefile"},
		{"common/readme.tmpl", "README.md"},
	}
	switch d.Layout {
	case "ddd":
		specs = append(specs,
			fileSpec{"ddd/main.tmpl", "cmd/api/main.go"},
			fileSpec{"ddd/config.tmpl", "internal/infrastructure/config/config.go"},
			fileSpec{"ddd/database.tmpl", "internal/infrastructure/database/database.go"},
			fileSpec{"ddd/router.tmpl", "internal/interface/http/router.go"},
		)
		if d.Sample {
			specs = append(specs,
				fileSpec{"ddd/domain_user.tmpl", "internal/domain/user/user.go"},
				fileSpec{"ddd/domain_repository.tmpl", "internal/domain/user/repository.go"},
				fileSpec{"ddd/application_service.tmpl", "internal/application/user/service.go"},
				fileSpec{"ddd/persistence_repository.tmpl", "internal/infrastructure/persistence/user_repository_bun.go"},
				fileSpec{"ddd/user_handler.tmpl", "internal/interface/http/user_handler.go"},
			)
		}
	case "layered":
		specs = append(specs,
			fileSpec{"layered/main.tmpl", "cmd/api/main.go"},
			fileSpec{"layered/config.tmpl", "internal/config/config.go"},
			fileSpec{"layered/router.tmpl", "internal/router/router.go"},
		)
		if d.Sample {
			specs = append(specs,
				fileSpec{"layered/model.tmpl", "internal/model/user.go"},
				fileSpec{"layered/repository.tmpl", "internal/repository/user_repo.go"},
				fileSpec{"layered/service.tmpl", "internal/service/user_service.go"},
				fileSpec{"layered/handler.tmpl", "internal/handler/user_handler.go"},
			)
		}
	}
	return specs
}

// Generate renders all planned templates into o.Dir and runs post-gen steps.
func Generate(o Options, out io.Writer) error {
	if nonEmpty, err := dirNonEmpty(o.Dir); err != nil {
		return err
	} else if nonEmpty {
		return fmt.Errorf("target directory %q already exists and is not empty", o.Dir)
	}

	d := Data{
		Name: o.Name, Module: o.Module, DB: o.DB, Layout: o.Layout,
		Sample: o.Sample, HelpersVersion: o.HelpersVersion, LocalReplace: o.Local,
	}

	for _, fsp := range plan(d) {
		if err := renderFile(fsp, d, o.Dir); err != nil {
			return fmt.Errorf("render %s: %w", fsp.tmpl, err)
		}
	}

	if !o.NoTidy {
		if o.Local == "" && o.HelpersVersion != "" {
			_ = runCmd(out, o.Dir, "go", "get", "github.com/rahmadafandi/fiber-helpers@"+o.HelpersVersion)
		}
		if err := runCmd(out, o.Dir, "go", "mod", "tidy"); err != nil {
			fmt.Fprintf(out, "warning: go mod tidy failed (%v); project generated, resolve deps manually or use --local\n", err)
		}
	}
	if !o.NoGit {
		_ = runCmd(out, o.Dir, "git", "init")
	}

	fmt.Fprintf(out, "created %s (%s, %s)\n", o.Dir, o.Layout, o.DB)
	return nil
}

func renderFile(fsp fileSpec, d Data, root string) error {
	raw, err := templatesFS.ReadFile("templates/" + fsp.tmpl)
	if err != nil {
		return err
	}
	tmpl, err := template.New(fsp.tmpl).Parse(string(raw))
	if err != nil {
		return err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, d); err != nil {
		return err
	}

	content := []byte(buf.String())
	if strings.HasSuffix(fsp.dest, ".go") {
		formatted, ferr := format.Source(content)
		if ferr != nil {
			return fmt.Errorf("gofmt %s: %w", fsp.dest, ferr)
		}
		content = formatted
	}

	full := filepath.Join(root, fsp.dest)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	return os.WriteFile(full, content, 0o644)
}

func dirNonEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func runCmd(out io.Writer, dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
