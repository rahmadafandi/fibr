// Copyright 2026 Rahmad Afandi. MIT License.

package slug_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/rahmadafandi/fibr/slug"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// Generate slugifies the title and appends a random suffix, retrying until the
// `slug` column of the given table has no collision.
func ExampleGenerate() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE articles (slug TEXT)"); err != nil {
		panic(err)
	}

	s, err := slug.Generate(context.Background(), db, "articles", "Hello, World!")
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.HasPrefix(s, "hello-world-"))
	// Output:
	// true
}
