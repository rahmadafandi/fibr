// Copyright 2026 Rahmad Afandi. MIT License.

package pagination_test

import (
	"fmt"

	"github.com/rahmadafandi/fiber-helpers/pagination"
)

// Build a page envelope from a slice of items plus the paging parameters. The
// derived metadata (PageCount, StartNumber, ...) is computed for you.
func ExampleNewPagination() {
	items := []string{"alice", "bob", "carol"}
	page := pagination.NewPagination(items, 10, 1, 23)

	fmt.Println("page:", page.PageNumber)
	fmt.Println("size:", page.PageSize)
	fmt.Println("on this page:", page.Count)
	fmt.Println("total:", page.TotalCount)
	fmt.Println("total pages:", page.PageCount)
	// Output:
	// page: 1
	// size: 10
	// on this page: 3
	// total: 23
	// total pages: 3
}
