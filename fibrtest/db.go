// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// NewDB returns a fresh in-memory SQLite Bun DB. When t supports Cleanup (a real
// *testing.T), the DB is closed automatically at test end; otherwise the caller
// is responsible for closing it. A connection error calls Fatalf.
func NewDB(t TB) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("fibrtest: open sqlite: %v", err)
		return nil
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	if c, ok := t.(interface{ Cleanup(func()) }); ok {
		c.Cleanup(func() { _ = db.Close() })
	}
	return db
}
