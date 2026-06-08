// Copyright 2026 Rahmad Afandi. MIT License.

package parser_test

import (
	"fmt"

	"github.com/rahmadafandi/fibr/parser"
)

// Validate guards the incoming query against an allow-list of sortable columns
// and the asc/desc order values before it touches the database.
func ExamplePaginationQuery_Validate() {
	pq := &parser.PaginationQuery{Page: 1, Limit: 20, Sort: "name", Order: "asc"}
	if err := pq.Validate([]string{"name", "created_at"}); err != nil {
		fmt.Println("err:", err)
	} else {
		fmt.Println("ok")
	}

	bad := &parser.PaginationQuery{Sort: "password", Order: "sideways"}
	fmt.Println("bad:", bad.Validate([]string{"name", "created_at"}))
	// Output:
	// ok
	// bad: validation failed: sort must be one of name, created_at, order must be one of asc or desc
}
