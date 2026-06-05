// Copyright 2026 Rahmad Afandi. MIT License.

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
	"time"
)

// now is the clock used for migration version stamps; overridable in tests.
var now = time.Now

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

// plan returns the skeleton templates to render for the given options. Sample
// module files are rendered separately by Generate via planModule.
func plan(d Data) []fileSpec {
	specs := []fileSpec{
		{"common/gomod.tmpl", "go.mod"},
		{"common/gitignore.tmpl", ".gitignore"},
		{"common/env.tmpl", ".env.example"},
		{"common/dockerfile.tmpl", "Dockerfile"},
		{"common/compose.tmpl", "docker-compose.yml"},
		{"common/makefile.tmpl", "Makefile"},
		{"common/readme.tmpl", "README.md"},
		{"common/migrations.tmpl", "internal/migrations/migrations.go"},
	}
	switch d.Layout {
	case "ddd":
		specs = append(specs,
			fileSpec{"ddd/main.tmpl", "cmd/api/main.go"},
			fileSpec{"ddd/config.tmpl", "internal/infrastructure/config/config.go"},
			fileSpec{"ddd/database.tmpl", "internal/infrastructure/database/database.go"},
			fileSpec{"ddd/router.tmpl", "internal/interface/http/router.go"},
		)
	case "layered":
		specs = append(specs,
			fileSpec{"layered/main.tmpl", "cmd/api/main.go"},
			fileSpec{"layered/config.tmpl", "internal/config/config.go"},
			fileSpec{"layered/router.tmpl", "internal/router/router.go"},
		)
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
			_ = os.RemoveAll(o.Dir)
			return fmt.Errorf("render %s: %w", fsp.tmpl, err)
		}
	}

	if o.Sample {
		md, err := deriveModuleNames("user")
		if err != nil {
			_ = os.RemoveAll(o.Dir)
			return err
		}
		md.Module = o.Module
		md.Layout = o.Layout
		for _, fsp := range planModule(md) {
			if err := renderFile(fsp, md, o.Dir); err != nil {
				_ = os.RemoveAll(o.Dir)
				return fmt.Errorf("render %s: %w", fsp.tmpl, err)
			}
		}
		if _, err := renderMigration(md, o.Dir); err != nil {
			_ = os.RemoveAll(o.Dir)
			return fmt.Errorf("render migration: %w", err)
		}
	}

	if !o.NoTidy {
		if o.Local == "" && o.HelpersVersion != "" {
			if err := runCmd(out, o.Dir, "go", "get", "github.com/rahmadafandi/fiber-helpers@"+o.HelpersVersion); err != nil {
				fmt.Fprintf(out, "warning: go get fiber-helpers@%s failed (%v); use --local for an unpublished library\n", o.HelpersVersion, err)
			}
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

// renderMigration writes a timestamped create-table migration for md into root.
func renderMigration(md ModuleData, root string) (string, error) {
	dest := filepath.Join("internal", "migrations",
		now().UTC().Format("20060102150405")+"_create_"+md.Plural+".go")
	if _, err := os.Stat(filepath.Join(root, dest)); err == nil {
		return "", fmt.Errorf("migration %s already exists (same-second collision; retry in a moment)", dest)
	}
	if err := renderFile(fileSpec{"migration/create_table.tmpl", dest}, md, root); err != nil {
		return "", err
	}
	return dest, nil
}

func renderFile(fsp fileSpec, data any, root string) error {
	raw, err := templatesFS.ReadFile("templates/" + fsp.tmpl)
	if err != nil {
		return err
	}
	tmpl, err := template.New(fsp.tmpl).Parse(string(raw))
	if err != nil {
		return err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
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
