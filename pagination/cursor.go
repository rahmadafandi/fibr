// Copyright 2026 Rahmad Afandi. MIT License.

package pagination

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rahmadafandi/fibr/parser"
)

// CursorPage is a single keyset (cursor) page together with the cursors needed
// to navigate to the next and previous pages. An empty cursor means there is no
// page in that direction.
type CursorPage[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor"` // "" = no next page
	PrevCursor string `json:"prev_cursor"` // "" = no previous page
}

// NewCursorPage builds a CursorPage from the rows returned by a parser.Keyset
// query (which fetched LIMIT+1 rows). It trims the sentinel extra row, restores
// display order for backward pages, and derives the Next/Prev cursors from the
// edge rows using extract, which must return the keyset column values for a row
// in the same order as cols.
func NewCursorPage[T any](rows []T, kq parser.KeysetQuery, cols []parser.KeysetColumn, extract func(T) []any) *CursorPage[T] {
	limit := kq.Limit
	if limit <= 0 {
		limit = 10
	}

	hasExtra := len(rows) > limit
	kept := rows
	if hasExtra {
		kept = rows[:limit]
	}
	if kq.Before {
		kept = reversed(kept)
	}

	page := &CursorPage[T]{Data: kept}
	if len(kept) == 0 {
		return page
	}

	first := extract(kept[0])
	last := extract(kept[len(kept)-1])

	if kq.Before {
		// Paged backward: a next page (the one we came from) always exists; a
		// previous page exists only if there were more rows beyond this one.
		page.NextCursor = parser.EncodeCursor(last)
		if hasExtra {
			page.PrevCursor = parser.EncodeCursor(first)
		}
		return page
	}

	// Paged forward: a next page exists only if there were more rows; a previous
	// page exists only if we arrived via a cursor (i.e. not the first page).
	if hasExtra {
		page.NextCursor = parser.EncodeCursor(last)
	}
	if kq.Cursor != "" {
		page.PrevCursor = parser.EncodeCursor(first)
	}
	return page
}

// LinkHeader returns an RFC 5988 Link header value pointing at the next and
// previous pages, derived from the cursors and baseURL (the request URL without
// the cursor/before params). It returns "" when there is neither a next nor a
// previous page. Set it with c.Set("Link", page.LinkHeader(url)).
func (p *CursorPage[T]) LinkHeader(baseURL string) string {
	var parts []string
	if p.NextCursor != "" {
		if l := cursorLink(baseURL, p.NextCursor, false, "next"); l != "" {
			parts = append(parts, l)
		}
	}
	if p.PrevCursor != "" {
		if l := cursorLink(baseURL, p.PrevCursor, true, "prev"); l != "" {
			parts = append(parts, l)
		}
	}
	return strings.Join(parts, ", ")
}

// cursorLink builds one Link header entry for the given cursor and rel.
func cursorLink(baseURL, cursor string, before bool, rel string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("cursor", cursor)
	if before {
		q.Set("before", "true")
	} else {
		q.Del("before")
	}
	u.RawQuery = q.Encode()
	return fmt.Sprintf("<%s>; rel=%q", u.String(), rel)
}

// reversed returns a new slice with the elements of s in reverse order, leaving
// s unmodified.
func reversed[T any](s []T) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}
