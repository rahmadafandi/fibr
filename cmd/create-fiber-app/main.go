// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"fmt"
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
		Use:           "create-fiber-app [name]",
		Short:         "Generate a ready-to-run Fiber project wired with fiber-helpers",
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.Name = args[0]
			}
			if err := o.Resolve(os.Stdin, os.Stdout, isTTY(os.Stdin), cmd.Flags().Changed); err != nil {
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
