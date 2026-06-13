// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	vals := []any{"echo", 5, true}
	got, err := DecodeCursor(EncodeCursor(vals))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "echo", got[0])
	assert.Equal(t, float64(5), got[1]) // JSON numbers decode as float64
	assert.Equal(t, true, got[2])
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	_, err := DecodeCursor("not base64 !!")
	assert.Error(t, err)

	// Valid base64 but not a JSON array.
	_, err = DecodeCursor(EncodeCursorRaw(`{"a":1}`))
	assert.Error(t, err)
}

// EncodeCursorRaw is a test helper that base64-encodes arbitrary JSON so we can
// feed DecodeCursor non-array payloads.
func EncodeCursorRaw(jsonText string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(jsonText))
}

func TestBuildSeekForwardAscending(t *testing.T) {
	cols := []KeysetColumn{{Name: "name"}, {Name: "id"}}
	pred, args := buildSeek(cols, []any{"bravo", float64(2)}, false)
	assert.Equal(t, "((name > ?) OR (name = ? AND id > ?))", pred)
	assert.Equal(t, []any{"bravo", "bravo", float64(2)}, args)
}

func TestBuildSeekDescendingAndBackwardInverts(t *testing.T) {
	cols := []KeysetColumn{{Name: "created_at", Desc: true}, {Name: "id", Desc: true}}

	// Forward, descending: use "<".
	fwd, _ := buildSeek(cols, []any{"t", float64(9)}, false)
	assert.Equal(t, "((created_at < ?) OR (created_at = ? AND id < ?))", fwd)

	// Backward inverts each column's effective direction: ">".
	bwd, _ := buildSeek(cols, []any{"t", float64(9)}, true)
	assert.Equal(t, "((created_at > ?) OR (created_at = ? AND id > ?))", bwd)
}

func TestKeysetGeneratesOrderAndLimit(t *testing.T) {
	db := newBunDB(t)
	cols := []KeysetColumn{{Name: "name"}, {Name: "id"}}

	sqlStr := db.NewSelect().Model((*article)(nil)).
		Apply(Keyset(KeysetQuery{}, cols)).String()
	assert.Contains(t, sqlStr, "ORDER BY")
	assert.Contains(t, sqlStr, "LIMIT 11") // default limit 10 + 1 sentinel
}

func TestKeysetRejectsInjection(t *testing.T) {
	db := newBunDB(t)
	cols := []KeysetColumn{{Name: "name; DROP TABLE articles"}}
	sqlStr := db.NewSelect().Model((*article)(nil)).
		Apply(Keyset(KeysetQuery{Limit: 5}, cols)).String()
	assert.NotContains(t, sqlStr, "DROP TABLE")
	assert.NotContains(t, sqlStr, "ORDER BY") // unsafe column -> ordering skipped
}

func TestKeysetExecutesForward(t *testing.T) {
	db := newBunDB(t) // names alpha,bravo,charlie,delta,echo with ids 1..5
	ctx := context.Background()
	cols := []KeysetColumn{{Name: "name"}, {Name: "id"}}

	// First page, limit 2 -> fetches 3 (limit+1).
	var first []article
	require.NoError(t, db.NewSelect().Model(&first).
		Apply(Keyset(KeysetQuery{Limit: 2}, cols)).Scan(ctx))
	require.Len(t, first, 3)
	assert.Equal(t, "alpha", first[0].Name)
	assert.Equal(t, "bravo", first[1].Name)

	// Next page from the bravo boundary.
	cursor := EncodeCursor([]any{"bravo", first[1].ID})
	var next []article
	require.NoError(t, db.NewSelect().Model(&next).
		Apply(Keyset(KeysetQuery{Limit: 2, Cursor: cursor}, cols)).Scan(ctx))
	require.NotEmpty(t, next)
	assert.Equal(t, "charlie", next[0].Name)
}
