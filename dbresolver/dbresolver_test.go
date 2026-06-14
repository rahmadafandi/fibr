// Copyright 2026 Rahmad Afandi. MIT License.

package dbresolver_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/dbresolver"
)

func newDB(t *testing.T, name string) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:"+name+"?mode=memory&cache=shared")
	require.NoError(t, err)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestWriterIsPrimary(t *testing.T) {
	primary := newDB(t, "p")
	r := dbresolver.New(primary, newDB(t, "r1"))
	assert.Same(t, primary, r.Writer())
}

func TestReaderRoundRobin(t *testing.T) {
	primary := newDB(t, "p")
	r1 := newDB(t, "r1")
	r2 := newDB(t, "r2")
	r := dbresolver.New(primary, r1, r2)

	assert.Same(t, r1, r.Reader())
	assert.Same(t, r2, r.Reader())
	assert.Same(t, r1, r.Reader()) // wraps
	assert.Same(t, r2, r.Reader())
}

func TestReaderNoReplicasFallsBackToPrimary(t *testing.T) {
	primary := newDB(t, "p")
	r := dbresolver.New(primary)
	assert.Same(t, primary, r.Reader())
}

func TestPing(t *testing.T) {
	r := dbresolver.New(newDB(t, "p"), newDB(t, "r1"))
	assert.NoError(t, r.Ping(context.Background()))
}

func TestConcurrentReaderRaceSafe(t *testing.T) {
	r := dbresolver.New(newDB(t, "p"), newDB(t, "r1"), newDB(t, "r2"))
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Reader()
		}()
	}
	wg.Wait()
}

func TestClose(t *testing.T) {
	// Separate from newDB's t.Cleanup close (double close of bun.DB is safe).
	primary := newDB(t, "cp")
	r := dbresolver.New(primary, newDB(t, "cr1"))
	assert.NoError(t, r.Close())
}
