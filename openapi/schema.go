// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

// schemaFor builds a schema for t, dereferencing pointers (a pointer makes the
// schema nullable). Named structs are stored in componentSchemas and referenced
// via $ref.
func (s *Spec) schemaFor(t reflect.Type) *Schema {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	sc := s.baseSchema(t)
	// A $ref cannot carry sibling keywords in OpenAPI 3.0, so skip nullable on refs.
	if nullable && sc.Ref == "" {
		sc.Nullable = true
	}
	return sc
}

func (s *Spec) baseSchema(t reflect.Type) *Schema {
	if t == timeType {
		return &Schema{Type: "string", Format: "date-time"}
	}
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer"}
	case reflect.Int64, reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		return &Schema{Type: "array", Items: s.schemaFor(t.Elem())}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: s.schemaFor(t.Elem())}
	case reflect.Struct:
		return s.structSchema(t)
	case reflect.Interface:
		return &Schema{}
	default:
		return &Schema{}
	}
}

func (s *Spec) structSchema(t reflect.Type) *Schema {
	name := t.Name()
	if name == "" {
		return s.objectSchema(t) // anonymous struct -> inline
	}
	ref := &Schema{Ref: "#/components/schemas/" + name}
	if _, ok := s.componentSchemas[name]; ok {
		return ref
	}
	// Reserve the name before recursing so self-references resolve to the ref.
	s.componentSchemas[name] = &Schema{Type: "object"}
	s.componentSchemas[name] = s.objectSchema(t)
	return ref
}

func (s *Spec) objectSchema(t reflect.Type) *Schema {
	obj := &Schema{Type: "object", Properties: map[string]*Schema{}}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		// Untagged embedded struct: flatten its fields.
		if f.Anonymous && f.Tag.Get("json") == "" {
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				sub := s.objectSchema(ft)
				for k, v := range sub.Properties {
					obj.Properties[k] = v
				}
				required = append(required, sub.Required...)
				continue
			}
		}
		name, omitempty, skip := jsonName(f)
		if skip {
			continue
		}
		fs := s.schemaFor(f.Type)
		rules := parseValidate(f.Tag.Get("validate"))
		applyValidate(fs, rules, f.Type)
		if rules.required {
			required = append(required, name)
		}
		_ = omitempty
		obj.Properties[name] = fs
	}
	if len(required) > 0 {
		obj.Required = required
	}
	return obj
}

// jsonName returns the property name, whether it is omitempty, and whether the
// field should be skipped (json:"-" or empty name).
func jsonName(f reflect.StructField) (name string, omitempty, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	parts := strings.Split(tag, ",")
	name = f.Name
	if parts[0] != "" {
		name = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

type validateRules struct {
	required bool
	rules    map[string]string // rule name -> value
	flags    map[string]bool   // bare rules like email, uuid
}

func parseValidate(tag string) validateRules {
	vr := validateRules{rules: map[string]string{}, flags: map[string]bool{}}
	if tag == "" {
		return vr
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "required" {
			vr.required = true
			continue
		}
		if k, v, ok := strings.Cut(part, "="); ok {
			vr.rules[k] = v
		} else {
			vr.flags[part] = true
		}
	}
	return vr
}

func applyValidate(sc *Schema, vr validateRules, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	isString := t.Kind() == reflect.String
	isNumber := isNumericKind(t.Kind())
	isList := t.Kind() == reflect.Slice || t.Kind() == reflect.Array

	if v, ok := vr.rules["oneof"]; ok {
		sc.Enum = parseEnum(v, t.Kind())
	}
	if v, ok := vr.rules["min"]; ok {
		applyBound(sc, v, true, isString, isNumber, isList)
	}
	if v, ok := vr.rules["max"]; ok {
		applyBound(sc, v, false, isString, isNumber, isList)
	}
	if v, ok := vr.rules["len"]; ok {
		applyBound(sc, v, true, isString, isNumber, isList)
		applyBound(sc, v, false, isString, isNumber, isList)
	}
	switch {
	case vr.flags["email"]:
		sc.Format = "email"
	case vr.flags["url"], vr.flags["uri"]:
		sc.Format = "uri"
	case vr.flags["uuid"], vr.flags["uuid4"]:
		sc.Format = "uuid"
	}
}

func applyBound(sc *Schema, raw string, isMin, isString, isNumber, isList bool) {
	switch {
	case isNumber:
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			if isMin {
				sc.Minimum = &f
			} else {
				sc.Maximum = &f
			}
		}
	case isString:
		if n, err := strconv.Atoi(raw); err == nil {
			if isMin {
				sc.MinLength = &n
			} else {
				sc.MaxLength = &n
			}
		}
	case isList:
		if n, err := strconv.Atoi(raw); err == nil {
			if isMin {
				sc.MinItems = &n
			} else {
				sc.MaxItems = &n
			}
		}
	}
}

func parseEnum(raw string, kind reflect.Kind) []any {
	var out []any
	for _, item := range strings.Fields(raw) {
		if isNumericKind(kind) {
			if f, err := strconv.ParseFloat(item, 64); err == nil {
				out = append(out, f)
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
