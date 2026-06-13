// Copyright 2026 Rahmad Afandi. MIT License.

package audit_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/audit"
)

// Record an audit entry and read it back.
func ExampleRecorder_Record() {
	sqldb, _ := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()
	ctx := context.Background()
	_ = audit.Migrate(ctx, db)

	rec := audit.New(audit.NewBunSink(db))
	_ = rec.Record(ctx, audit.Entry{
		Actor:    "admin",
		Action:   "user.suspend",
		Target:   "user",
		TargetID: "99",
	})

	entries, _ := audit.List(ctx, db, audit.Filter{Actor: "admin"})
	fmt.Println(entries[0].Action, entries[0].TargetID)
	// Output: user.suspend 99
}
