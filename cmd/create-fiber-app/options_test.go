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

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		o := Options{Name: "app", Module: "github.com/me/app", DB: "postgres", Layout: "ddd"}
		assert.NoError(t, o.Validate())
		assert.Equal(t, "app", o.Dir)
	})
	t.Run("missing name", func(t *testing.T) {
		o := Options{Module: "m", DB: "postgres", Layout: "ddd"}
		assert.Error(t, o.Validate())
	})
	t.Run("missing module", func(t *testing.T) {
		o := Options{Name: "app", DB: "postgres", Layout: "ddd"}
		assert.Error(t, o.Validate())
	})
	t.Run("bad db", func(t *testing.T) {
		o := Options{Name: "app", Module: "m", DB: "mysql", Layout: "ddd"}
		assert.Error(t, o.Validate())
	})
	t.Run("bad layout", func(t *testing.T) {
		o := Options{Name: "app", Module: "m", DB: "sqlite", Layout: "mvc"}
		assert.Error(t, o.Validate())
	})
	t.Run("bad name path", func(t *testing.T) {
		o := Options{Name: "../evil", Module: "m", DB: "sqlite", Layout: "ddd"}
		assert.Error(t, o.Validate())
	})
	t.Run("bad name slash", func(t *testing.T) {
		o := Options{Name: "a/b", Module: "m", DB: "sqlite", Layout: "ddd"}
		assert.Error(t, o.Validate())
	})
}

func TestResolveInteractive(t *testing.T) {
	in := strings.NewReader("myapp\ngithub.com/me/myapp\n\n\ny\n")
	var out strings.Builder
	o := Options{DB: "postgres", Layout: "ddd"}
	err := o.Resolve(in, &out, true, func(string) bool { return false })
	assert.NoError(t, err)
	assert.Equal(t, "myapp", o.Name)
	assert.Equal(t, "github.com/me/myapp", o.Module)
	assert.True(t, o.Sample)
}

func TestResolveNonInteractiveMissingModule(t *testing.T) {
	var out strings.Builder
	o := Options{Name: "app", DB: "postgres", Layout: "ddd"}
	err := o.Resolve(strings.NewReader(""), &out, false, nil)
	assert.Error(t, err)
}

func TestResolveSkipsSetFlags(t *testing.T) {
	// name+module provided; db flag "changed" -> only sample/layout would prompt,
	// db must keep its flag value without consuming input for it.
	in := strings.NewReader("\nn\n") // layout: accept default, sample: no
	var out strings.Builder
	o := Options{Name: "app", Module: "m", DB: "sqlite", Layout: "ddd"}
	changed := func(f string) bool { return f == "db" }
	err := o.Resolve(in, &out, true, changed)
	assert.NoError(t, err)
	assert.Equal(t, "sqlite", o.DB)                   // kept from flag, not overwritten
	assert.NotContains(t, out.String(), "Database (") // db prompt skipped
}
