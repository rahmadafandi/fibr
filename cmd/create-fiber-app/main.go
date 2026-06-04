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
	"io"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var o Options
	cmd := &cobra.Command{
		Use:   "create-fiber-app [name]",
		Short: "Generate a ready-to-run Fiber project wired with fiber-helpers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.Name = args[0]
			}
			if err := o.Resolve(os.Stdin, os.Stdout, isTTY(os.Stdin)); err != nil {
				return err
			}
			return Generate(o, os.Stdout)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Module, "module", "", "Go module path (required)")
	f.StringVar(&o.DB, "db", "postgres", "database driver: postgres|sqlite")
	f.StringVar(&o.Layout, "layout", "ddd", "project layout: ddd|layered")
	f.BoolVar(&o.Sample, "sample", false, "include a sample CRUD domain")
	f.StringVar(&o.Dir, "dir", "", "output directory (default ./<name>)")
	f.BoolVar(&o.NoGit, "no-git", false, "skip git init")
	f.BoolVar(&o.NoTidy, "no-tidy", false, "skip go mod tidy")
	f.StringVar(&o.HelpersVersion, "helpers-version", "latest", "fiber-helpers version for go.mod")
	f.StringVar(&o.Local, "local", "", "replace fiber-helpers with a local path")
	return cmd
}

// Generate is implemented in generate.go (Task 2); temporary stub for Task 1.
func Generate(o Options, out io.Writer) error { return nil }
