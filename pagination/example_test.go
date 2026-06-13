// Copyright 2026 Rahmad Afandi. MIT License.

package pagination_test

import (
	"fmt"

	"github.com/rahmadafandi/fibr/pagination"
	"github.com/rahmadafandi/fibr/parser"
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

// Build a keyset (cursor) page from the rows a parser.Keyset query returns. The
// query fetches LIMIT+1 rows; NewCursorPage trims the sentinel and derives the
// navigation cursors. Here limit is 2 and three rows came back, so a next page
// exists; with no incoming cursor this is the first page, so there is no
// previous one.
func ExampleNewCursorPage() {
	type article struct {
		ID    int64
		Title string
	}
	rows := []article{{1, "first"}, {2, "second"}, {3, "third"}}
	cols := []parser.KeysetColumn{{Name: "id"}}
	kq := parser.KeysetQuery{Limit: 2}

	page := pagination.NewCursorPage(rows, kq, cols, func(a article) []any {
		return []any{a.ID}
	})

	fmt.Println("rows:", len(page.Data))
	fmt.Println("has next:", page.NextCursor != "")
	fmt.Println("has prev:", page.PrevCursor != "")
	// Output:
	// rows: 2
	// has next: true
	// has prev: false
}
