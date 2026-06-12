// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := newRootCmd()
	root.AddCommand(newNewCmd())
	root.AddCommand(newAddCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "fibr",
		Short:         "Fibr — scaffold and extend batteries-included Fiber projects",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
}

func newAddCmd() *cobra.Command {
	add := &cobra.Command{
		Use:   "add",
		Short: "Add components to an existing generated project",
	}
	add.AddCommand(newAddModuleCmd())
	return add
}

func newAddModuleCmd() *cobra.Command {
	var dir, layout string
	cmd := &cobra.Command{
		Use:           "module <name>",
		Short:         "Scaffold a new feature module (model, repo, service, handler, wiring)",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddModule(AddModuleOptions{Name: args[0], Dir: dir, Layout: layout}, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "project directory")
	cmd.Flags().StringVar(&layout, "layout", "", "layout: ddd|layered (auto-detected if empty)")
	return cmd
}

func newNewCmd() *cobra.Command {
	var o Options
	cmd := &cobra.Command{
		Use:           "new [name]",
		Short:         "Generate a ready-to-run Fiber project wired with fibr",
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
	f.BoolVar(&o.Auth, "auth", false, "include auth scaffold (JWT + accounts)")
	f.BoolVar(&o.Team, "auth-with-team", false, "include teams/workspaces scaffold (implies --auth)")
	f.BoolVar(&o.Queue, "queue", false, "include background job queue (asynq + asynqmon)")
	f.BoolVar(&o.Mailer, "mailer", false, "include transactional email (SMTP mailer)")
	f.BoolVar(&o.Realtime, "realtime", false, "include a realtime sample (WebSocket chat + SSE stream)")
	f.StringVar(&o.Dir, "dir", "", "output directory (default ./<name>)")
	f.BoolVar(&o.NoGit, "no-git", false, "skip git init")
	f.BoolVar(&o.NoTidy, "no-tidy", false, "skip go mod tidy")
	f.StringVar(&o.HelpersVersion, "helpers-version", "latest", "fibr version for go.mod")
	f.StringVar(&o.Local, "local", "", "replace fibr with a local path")
	return cmd
}
