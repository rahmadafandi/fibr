// Copyright 2026 Rahmad Afandi. MIT License.

package migrate

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
)

// DBProvider lazily opens the application database. Only up/down/status call it.
type DBProvider func(ctx context.Context) (*bun.DB, error)

// NewCommand builds the "migrate" cobra command with up/down/status/create
// children, wired to the core funcs. Mount it on the app root command.
func NewCommand(dbFn DBProvider, ms *Migrations) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "migrate",
		Short:         "database migration commands",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// run wraps a core migrate func so it handles DB open/close and output.
	// The db is opened lazily (only when the subcommand executes) and closed
	// via defer so it's released even if the migration func errors.
	run := func(fn func(ctx context.Context, db *bun.DB, ms *Migrations) (string, error)) func(*cobra.Command, []string) error {
		return func(c *cobra.Command, _ []string) error {
			ctx := c.Context()
			db, err := dbFn(ctx)
			if err != nil {
				return err
			}
			if db == nil {
				return fmt.Errorf("migrate: DBProvider returned a nil database")
			}
			defer db.Close() //nolint:errcheck
			msg, err := fn(ctx, db, ms)
			if err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), msg)
			return nil
		}
	}

	cmd.AddCommand(&cobra.Command{Use: "up", Short: "apply pending migrations", RunE: run(Up)})
	cmd.AddCommand(&cobra.Command{Use: "down", Short: "roll back the last migration group", RunE: run(Down)})
	cmd.AddCommand(&cobra.Command{Use: "status", Short: "show migration status", RunE: run(Status)})
	cmd.AddCommand(&cobra.Command{
		Use:   "create <name>",
		Short: "scaffold a new Go migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			path, err := Create(ms, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "created", path)
			return nil
		},
	})
	return cmd
}
