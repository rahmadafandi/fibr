// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newSpec() *Spec {
	return &Spec{
		paths:            map[string]*PathItem{},
		componentSchemas: map[string]*Schema{},
		securitySchemes:  map[string]*SecurityScheme{},
	}
}

func TestSchemaScalars(t *testing.T) {
	s := newSpec()
	require.Equal(t, "string", s.schemaFor(reflect.TypeOf("")).Type)
	require.Equal(t, "boolean", s.schemaFor(reflect.TypeOf(true)).Type)
	require.Equal(t, "integer", s.schemaFor(reflect.TypeOf(0)).Type)

	i64 := s.schemaFor(reflect.TypeOf(int64(0)))
	require.Equal(t, "integer", i64.Type)
	require.Equal(t, "int64", i64.Format)

	f := s.schemaFor(reflect.TypeOf(float64(0)))
	require.Equal(t, "number", f.Type)
	require.Equal(t, "double", f.Format)

	tm := s.schemaFor(reflect.TypeOf(time.Time{}))
	require.Equal(t, "string", tm.Type)
	require.Equal(t, "date-time", tm.Format)
}

func TestSchemaContainers(t *testing.T) {
	s := newSpec()
	arr := s.schemaFor(reflect.TypeOf([]string{}))
	require.Equal(t, "array", arr.Type)
	require.Equal(t, "string", arr.Items.Type)

	m := s.schemaFor(reflect.TypeOf(map[string]int{}))
	require.Equal(t, "object", m.Type)
	require.Equal(t, "integer", m.AdditionalProperties.Type)
}

func TestSchemaPointerNullable(t *testing.T) {
	s := newSpec()
	p := s.schemaFor(reflect.TypeOf((*string)(nil)))
	require.Equal(t, "string", p.Type)
	require.True(t, p.Nullable)
}

type address struct {
	City string `json:"city"`
}

type person struct {
	Name   string  `json:"name" validate:"required"`
	Email  string  `json:"email" validate:"required,email"`
	Age    int     `json:"age" validate:"min=18,max=120"`
	Role   string  `json:"role" validate:"oneof=admin user"`
	Nick   string  `json:"nick,omitempty"`
	Secret string  `json:"-"`
	Addr   address `json:"addr"`
	hidden string  //nolint:unused // unexported, must be skipped
}

func TestStructSchemaRefAndComponents(t *testing.T) {
	s := newSpec()
	ref := s.schemaFor(reflect.TypeOf(person{}))
	require.Equal(t, "#/components/schemas/person", ref.Ref)

	p := s.componentSchemas["person"]
	require.NotNil(t, p)
	require.Equal(t, "object", p.Type)

	require.Contains(t, p.Properties, "name")
	require.Contains(t, p.Properties, "nick")
	require.NotContains(t, p.Properties, "Secret")
	require.NotContains(t, p.Properties, "hidden")

	require.ElementsMatch(t, []string{"name", "email"}, p.Required)

	require.Equal(t, "email", p.Properties["email"].Format)
	require.NotNil(t, p.Properties["age"].Minimum)
	require.Equal(t, float64(18), *p.Properties["age"].Minimum)
	require.Equal(t, float64(120), *p.Properties["age"].Maximum)
	require.ElementsMatch(t, []any{"admin", "user"}, p.Properties["role"].Enum)

	require.Equal(t, "#/components/schemas/address", p.Properties["addr"].Ref)
	require.NotNil(t, s.componentSchemas["address"])
}

func TestStringLenConstraints(t *testing.T) {
	s := newSpec()
	type t1 struct {
		Code string `json:"code" validate:"min=2,max=5"`
	}
	_ = s.schemaFor(reflect.TypeOf(t1{}))
	c := s.componentSchemas["t1"].Properties["code"]
	require.NotNil(t, c.MinLength)
	require.Equal(t, 2, *c.MinLength)
	require.Equal(t, 5, *c.MaxLength)
}

type node struct {
	Name string `json:"name"`
	Next *node  `json:"next,omitempty"`
}

func TestRecursiveStructTerminates(t *testing.T) {
	s := newSpec()
	ref := s.schemaFor(reflect.TypeOf(node{}))
	require.Equal(t, "#/components/schemas/node", ref.Ref)
	n := s.componentSchemas["node"]
	require.Equal(t, "#/components/schemas/node", n.Properties["next"].Ref)
}
