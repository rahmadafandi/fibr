// Copyright 2026 Rahmad Afandi. MIT License.

package parser

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/database"
)

// Runs only when FIBR_TEST_POSTGRES_DSN points at a disposable database.
func TestPostgresSearchAndPaginate(t *testing.T) {
	dsn := os.Getenv("FIBR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("FIBR_TEST_POSTGRES_DSN not set")
	}
	db, err := database.NewBun(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	drop := func() { _, _ = db.NewDropTable().Model((*article)(nil)).IfExists().Exec(ctx) }
	drop()
	t.Cleanup(drop)
	_, err = db.NewCreateTable().Model((*article)(nil)).Exec(ctx)
	require.NoError(t, err)
	for _, name := range []string{"Alpha", "alphabet", "bravo"} {
		_, err = db.NewInsert().Model(&article{Name: name}).Exec(ctx)
		require.NoError(t, err)
	}

	n, err := db.NewSelect().Model((*article)(nil)).Apply(Count("ALPHA", []string{"name"})).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n, "search must be case-insensitive on Postgres")

	var got []article
	pq := &PaginationQuery{Search: "alpha", Sort: "name", Order: "desc", Limit: 1}
	require.NoError(t, db.NewSelect().Model(&got).Apply(Paginate(pq, []string{"name"})).Scan(ctx))
	require.Len(t, got, 1)
	assert.Equal(t, "alphabet", got[0].Name)
}
