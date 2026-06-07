// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AddModuleOptions configures the add-module command.
type AddModuleOptions struct {
	Name   string
	Dir    string // project directory (default ".")
	Layout string // "" = auto-detect
}

// readGoModule returns the module path from the go.mod in dir.
func readGoModule(dir string) (string, error) {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("not a Go module in %q (no go.mod): %w", dir, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("go.mod in %q has no module directive", dir)
}

// detectLayout infers the project layout from its directory structure.
func detectLayout(dir string) (string, error) {
	if isDir(filepath.Join(dir, "internal", "domain")) {
		return "ddd", nil
	}
	if isDir(filepath.Join(dir, "internal", "model")) {
		return "layered", nil
	}
	return "", fmt.Errorf("could not detect layout in %q; pass --layout ddd|layered", dir)
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// mountHint returns the import line and the app.Mount call line for the layout.
func mountHint(layout, modulePath string, md ModuleData) (importLine, mountLine string) {
	switch layout {
	case "ddd":
		importLine = fmt.Sprintf("\thttpiface \"%s/internal/interface/http\"", modulePath)
		mountLine = fmt.Sprintf("app.Mount(httpiface.New%sModule(db))", md.Type)
	case "layered":
		importLine = fmt.Sprintf("\t\"%s/internal/handler\"", modulePath)
		mountLine = fmt.Sprintf("app.Mount(handler.New%sModule(db))", md.Type)
	}
	return importLine, mountLine
}

// AddModule scaffolds a new feature module in an existing project.
func AddModule(o AddModuleOptions, out io.Writer) error {
	if o.Dir == "" {
		o.Dir = "."
	}
	modulePath, err := readGoModule(o.Dir)
	if err != nil {
		return err
	}

	layout := o.Layout
	if layout == "" {
		if layout, err = detectLayout(o.Dir); err != nil {
			return err
		}
	}
	if layout != "ddd" && layout != "layered" {
		return fmt.Errorf("--layout must be \"ddd\" or \"layered\"")
	}

	md, err := deriveModuleNames(o.Name)
	if err != nil {
		return err
	}
	md.Module = modulePath
	md.Layout = layout

	specs := planModule(md)
	for _, s := range specs {
		if _, err := os.Stat(filepath.Join(o.Dir, s.dest)); err == nil {
			return fmt.Errorf("module file already exists: %s (refusing to overwrite)", s.dest)
		}
	}
	var written []string
	for _, s := range specs {
		if err := renderFile(s, md, o.Dir); err != nil {
			for _, w := range written {
				_ = os.Remove(w)
			}
			return fmt.Errorf("render %s: %w", s.tmpl, err)
		}
		written = append(written, filepath.Join(o.Dir, s.dest))
	}

	migPath, err := renderMigration(md, o.Dir, newMigrationClock().next())
	if err != nil {
		for _, w := range written {
			_ = os.Remove(w)
		}
		return fmt.Errorf("render migration: %w", err)
	}

	importLine, mountLine := mountHint(layout, modulePath, md)
	fmt.Fprintf(out, "added module %q (%s).\n\n", md.Pkg, layout)
	fmt.Fprintf(out, "Wire it in cmd/api/main.go:\n")
	fmt.Fprintf(out, "  1. ensure this import is present in the import block:\n%s\n", importLine)
	fmt.Fprintf(out, "  2. inside runServe in cmd/api/main.go, after bootstrap.New(...), add:\n       if err := %s; err != nil {\n           return err\n       }\n", mountLine)
	fmt.Fprintf(out, "  3. run \"go run ./cmd/api migrate up\" to apply the new migration (%s)\n", filepath.Base(migPath))
	return nil
}
