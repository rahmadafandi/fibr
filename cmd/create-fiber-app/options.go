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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Options holds the resolved generator inputs.
type Options struct {
	Name           string
	Module         string
	DB             string
	Layout         string
	Sample         bool
	Dir            string
	NoGit          bool
	NoTidy         bool
	HelpersVersion string
	Local          string
}

// Resolve fills missing fields from interactive prompts (when interactive) then
// validates. changed reports whether a flag was explicitly set by the user;
// fields whose flag was set are not re-prompted. changed may be nil.
func (o *Options) Resolve(in io.Reader, out io.Writer, interactive bool, changed func(string) bool) error {
	if changed == nil {
		changed = func(string) bool { return false }
	}
	r := bufio.NewReader(in)
	if interactive {
		if o.Name == "" {
			o.Name = prompt(r, out, "Project name", "")
		}
		if o.Module == "" {
			o.Module = prompt(r, out, "Module path", "")
		}
		if !changed("db") {
			o.DB = prompt(r, out, "Database (postgres/sqlite)", o.DB)
		}
		if !changed("layout") {
			o.Layout = prompt(r, out, "Layout (ddd/layered)", o.Layout)
		}
		if !changed("sample") && !o.Sample {
			o.Sample = yesNo(r, out, "Include sample CRUD?", false)
		}
	}
	return o.Validate()
}

// Validate checks required fields and allowed values; it also defaults Dir.
func (o *Options) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.ContainsAny(o.Name, `/\`) || strings.Contains(o.Name, "..") {
		return fmt.Errorf("project name %q must be a simple name (no path separators or \"..\")", o.Name)
	}
	if o.Module == "" {
		return fmt.Errorf("--module is required")
	}
	if o.DB != "postgres" && o.DB != "sqlite" {
		return fmt.Errorf("--db must be \"postgres\" or \"sqlite\"")
	}
	if o.Layout != "ddd" && o.Layout != "layered" {
		return fmt.Errorf("--layout must be \"ddd\" or \"layered\"")
	}
	if o.Dir == "" {
		o.Dir = o.Name
	}
	return nil
}

func prompt(r *bufio.Reader, out io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func yesNo(r *bufio.Reader, out io.Writer, label string, def bool) bool {
	d := "y/N"
	if def {
		d = "Y/n"
	}
	fmt.Fprintf(out, "%s [%s]: ", label, d)
	line, _ := r.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

// isTTY reports whether r is an interactive terminal.
func isTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
