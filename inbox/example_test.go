// Copyright 2026 Rahmad Afandi. MIT License.

package inbox_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/inbox"
)

// Process a message exactly once, even if it is delivered twice (at-least-once).
func ExampleOnce() {
	sqldb, _ := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()
	ctx := context.Background()
	_ = inbox.Migrate(ctx, db)

	handle := func(_ context.Context, _ bun.Tx) error {
		fmt.Println("processing order.created")
		return nil
	}

	// Same message delivered twice.
	_ = inbox.Once(ctx, db, "order.created:42", handle)
	_ = inbox.Once(ctx, db, "order.created:42", handle)
	// Output:
	// processing order.created
}
