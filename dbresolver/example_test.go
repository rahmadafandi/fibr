// Copyright 2026 Rahmad Afandi. MIT License.

package dbresolver_test

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/dbresolver"
)

func open(name string) *bun.DB {
	sqldb, _ := sql.Open(sqliteshim.ShimName, "file:"+name+"?mode=memory&cache=shared")
	return bun.NewDB(sqldb, sqlitedialect.New())
}

// Send writes to the primary and reads to replicas (round-robin).
func ExampleResolver() {
	r := dbresolver.New(open("primary"), open("replica1"), open("replica2"))
	defer r.Close()

	// Writes: r.Writer().NewInsert()...  Reads: r.Reader().NewSelect()...
	fmt.Println(r.Writer() != nil, r.Reader() != nil)
	// Output: true true
}
