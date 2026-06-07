// Copyright 2026 Rahmad Afandi. MIT License.

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDetectDialect(t *testing.T) {
	cases := []struct {
		dsn     string
		want    dialectKind
		wantErr bool
	}{
		{"postgres://user:pass@localhost:5432/db", dialectPostgres, false},
		{"postgresql://localhost/db", dialectPostgres, false},
		{"file::memory:?cache=shared", dialectSQLite, false},
		{":memory:", dialectSQLite, false},
		{"file:/tmp/app.db", dialectSQLite, false},
		{"/var/data/app.db", dialectSQLite, false},
		{"app.db", dialectSQLite, false},
		{"mysql://localhost/db", dialectUnknown, true},
		{"redis://localhost:6379", dialectUnknown, true},
	}
	for _, tc := range cases {
		got, err := detectDialect(tc.dsn)
		if tc.wantErr {
			assert.Error(t, err, tc.dsn)
			continue
		}
		assert.NoError(t, err, tc.dsn)
		assert.Equal(t, tc.want, got, tc.dsn)
	}
}

func TestNewBunSQLite(t *testing.T) {
	db, err := NewBun("file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	var n int
	err = db.NewRaw("SELECT 1").Scan(context.Background(), &n)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestNewBunUnknownDialect(t *testing.T) {
	_, err := NewBun("mysql://localhost/db")
	assert.Error(t, err)
}

func TestNewBunWithoutPing(t *testing.T) {
	db, err := NewBun("file::memory:?cache=shared", WithoutPing(), WithPingTimeout(time.Second))
	require.NoError(t, err)
	defer db.Close()
	assert.NotNil(t, db)
}

func TestDetectDialectRejectsMalformedScheme(t *testing.T) {
	for _, dsn := range []string{"postgres:/host", "weird:thing", "http:/x"} {
		_, err := detectDialect(dsn)
		assert.Error(t, err, dsn)
	}
}

func TestNewBunPostgresWithoutPing(t *testing.T) {
	// Exercises the Postgres branch without a live server (sql.OpenDB is lazy).
	db, err := NewBun("postgres://user:pass@localhost:5432/db", WithoutPing())
	require.NoError(t, err)
	defer db.Close()
	assert.NotNil(t, db)
}

func TestWithTracingRecordsQuerySpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	db, err := NewBun("file::memory:?cache=shared", WithTracing(), WithoutPing())
	require.NoError(t, err)
	defer db.Close()

	var n int
	require.NoError(t, db.NewSelect().ColumnExpr("1").Scan(context.Background(), &n))
	require.Equal(t, 1, n)

	require.NotEmpty(t, sr.Ended(), "expected at least one query span")
}

func TestWithoutTracingNoQuerySpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	db, err := NewBun("file::memory:?cache=shared", WithoutPing())
	require.NoError(t, err)
	defer db.Close()

	var n int
	require.NoError(t, db.NewSelect().ColumnExpr("1").Scan(context.Background(), &n))

	require.Empty(t, sr.Ended(), "no spans expected without WithTracing")
}
