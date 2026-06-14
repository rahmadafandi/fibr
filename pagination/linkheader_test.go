// Copyright 2026 Rahmad Afandi. MIT License.

package pagination_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rahmadafandi/fibr/pagination"
)

func TestLinkHeaderNextAndPrev(t *testing.T) {
	p := &pagination.CursorPage[int]{NextCursor: "abc", PrevCursor: "xyz"}
	h := p.LinkHeader("https://api.test/items?limit=20")

	// Both relations present.
	assert.Contains(t, h, `rel="next"`)
	assert.Contains(t, h, `rel="prev"`)
	// Cursors encoded into the query.
	assert.Contains(t, h, "cursor=abc")
	assert.Contains(t, h, "cursor=xyz")
	// Prev carries before=true; the base query is preserved.
	assert.Contains(t, h, "before=true")
	assert.Contains(t, h, "limit=20")
	// Two entries joined by ", ".
	assert.Len(t, strings.Split(h, ", "), 2)
}

func TestLinkHeaderOnlyNext(t *testing.T) {
	p := &pagination.CursorPage[int]{NextCursor: "abc"}
	h := p.LinkHeader("https://api.test/items")
	assert.Contains(t, h, `rel="next"`)
	assert.NotContains(t, h, `rel="prev"`)
	assert.NotContains(t, h, "before=true")
}

func TestLinkHeaderEmpty(t *testing.T) {
	p := &pagination.CursorPage[int]{}
	assert.Equal(t, "", p.LinkHeader("https://api.test/items"))
}
