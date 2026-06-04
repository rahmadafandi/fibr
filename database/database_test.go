// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
