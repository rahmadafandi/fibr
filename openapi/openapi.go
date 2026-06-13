// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Spec accumulates registered operations and builds an OpenAPI document.
type Spec struct {
	info             Info
	paths            map[string]*PathItem
	componentSchemas map[string]*Schema
	securitySchemes  map[string]*SecurityScheme

	once   sync.Once
	cached []byte
	cerr   error
}

const bearerSchemeName = "bearerAuth"

// New creates an empty spec with the given info.
func New(info Info) *Spec {
	return &Spec{
		info:             info,
		paths:            map[string]*PathItem{},
		componentSchemas: map[string]*Schema{},
		securitySchemes:  map[string]*SecurityScheme{},
	}
}

// WithBearerAuth registers an HTTP bearer (JWT) security scheme named
// "bearerAuth". Operations opt in via Op.Secured. Idempotent; returns the spec
// for chaining.
func (s *Spec) WithBearerAuth() *Spec {
	s.securitySchemes[bearerSchemeName] = &SecurityScheme{
		Type: "http", Scheme: "bearer", BearerFormat: "JWT",
	}
	return s
}

// Op describes one operation to register. Request/Response/Query are struct
// values (or nil) reflected into schemas.
type Op struct {
	Summary     string
	Description string
	Tags        []string
	Request     any
	Response    any
	Query       any
	Status      int  // success status; 0 -> 200
	Secured     bool // require the bearerAuth scheme
}

// Register records one operation under method+path. method is case-insensitive.
// Fiber-style ":param" / "*" path segments convert to OpenAPI "{param}".
// Returns the spec for chaining.
func (s *Spec) Register(method, path string, op Op) *Spec {
	oapiPath, pathParams := convertPath(path)
	item := s.paths[oapiPath]
	if item == nil {
		item = &PathItem{}
		s.paths[oapiPath] = item
	}

	o := &Operation{
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
		Responses:   map[string]Response{},
	}
	for _, name := range pathParams {
		o.Parameters = append(o.Parameters, Parameter{
			Name: name, In: "path", Required: true, Schema: &Schema{Type: "string"},
		})
	}
	if op.Query != nil {
		o.Parameters = append(o.Parameters, s.queryParams(op.Query)...)
	}
	if op.Request != nil {
		o.RequestBody = &RequestBody{
			Required: true,
			Content:  map[string]MediaType{"application/json": {Schema: s.schemaFor(reflect.TypeOf(op.Request))}},
		}
	}
	status := op.Status
	if status == 0 {
		status = http.StatusOK
	}
	desc := http.StatusText(status)
	if desc == "" {
		desc = "response"
	}
	resp := Response{Description: desc}
	if op.Response != nil {
		resp.Content = map[string]MediaType{"application/json": {Schema: s.schemaFor(reflect.TypeOf(op.Response))}}
	}
	o.Responses[strconv.Itoa(status)] = resp
	if op.Secured {
		o.Security = []map[string][]string{{bearerSchemeName: []string{}}}
	}
	setOp(item, method, o)
	return s
}

func (s *Spec) queryParams(v any) []Parameter {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var params []Parameter
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name, skip := jsonName(f)
		if skip {
			continue
		}
		rules := parseValidate(f.Tag.Get("validate"))
		sc := s.schemaFor(f.Type)
		applyValidate(sc, rules, f.Type)
		params = append(params, Parameter{
			Name: name, In: "query", Required: rules.required, Schema: sc,
		})
	}
	return params
}

func convertPath(path string) (string, []string) {
	var params []string
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		switch {
		case strings.HasPrefix(seg, ":"):
			name := seg[1:]
			params = append(params, name)
			segs[i] = "{" + name + "}"
		case seg == "*":
			params = append(params, "wildcard")
			segs[i] = "{wildcard}"
		}
	}
	return strings.Join(segs, "/"), params
}

func setOp(item *PathItem, method string, o *Operation) {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		item.Get = o
	case http.MethodPost:
		item.Post = o
	case http.MethodPut:
		item.Put = o
	case http.MethodPatch:
		item.Patch = o
	case http.MethodDelete:
		item.Delete = o
	case http.MethodHead:
		item.Head = o
	case http.MethodOptions:
		item.Options = o
	}
}

// Build assembles the OpenAPI document. Safe to call multiple times.
func (s *Spec) Build() *Document {
	doc := &Document{OpenAPI: "3.0.3", Info: s.info, Paths: s.paths}
	if len(s.componentSchemas) > 0 || len(s.securitySchemes) > 0 {
		c := &Components{}
		if len(s.componentSchemas) > 0 {
			c.Schemas = s.componentSchemas
		}
		if len(s.securitySchemes) > 0 {
			c.SecuritySchemes = s.securitySchemes
		}
		doc.Components = c
	}
	return doc
}

// MarshalJSON marshals the built document.
func (s *Spec) MarshalJSON() ([]byte, error) { return json.Marshal(s.Build()) }
