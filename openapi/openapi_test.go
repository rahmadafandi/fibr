// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type loginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
}

type listOpts struct {
	Page  int    `json:"page"`
	Query string `json:"q" validate:"required"`
}

func TestRegisterBuildsPathAndBody(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	s.Register("post", "/users/:id/posts", Op{
		Summary:  "Create",
		Tags:     []string{"posts"},
		Request:  loginReq{},
		Response: tokenResp{},
		Status:   201,
	})
	doc := s.Build()

	item := doc.Paths["/users/{id}/posts"]
	require.NotNil(t, item)
	require.NotNil(t, item.Post)
	require.Equal(t, "Create", item.Post.Summary)

	var hasID bool
	for _, p := range item.Post.Parameters {
		if p.In == "path" && p.Name == "id" {
			hasID = true
			require.True(t, p.Required)
		}
	}
	require.True(t, hasID)

	require.NotNil(t, item.Post.RequestBody)
	require.Equal(t, "#/components/schemas/loginReq",
		item.Post.RequestBody.Content["application/json"].Schema.Ref)

	resp, ok := item.Post.Responses["201"]
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/tokenResp",
		resp.Content["application/json"].Schema.Ref)

	require.Equal(t, "3.0.3", doc.OpenAPI)
}

func TestDefaultStatus200(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	s.Register("GET", "/ping", Op{Response: tokenResp{}})
	_, ok := s.Build().Paths["/ping"].Get.Responses["200"]
	require.True(t, ok)
}

func TestQueryParams(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	s.Register("GET", "/items", Op{Query: listOpts{}})
	params := s.Build().Paths["/items"].Get.Parameters
	names := map[string]Parameter{}
	for _, p := range params {
		require.Equal(t, "query", p.In)
		names[p.Name] = p
	}
	require.Contains(t, names, "page")
	require.Contains(t, names, "q")
	require.True(t, names["q"].Required)
	require.False(t, names["page"].Required)
}

func TestBearerAuthAndSecured(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"}).WithBearerAuth()
	s.Register("GET", "/me", Op{Secured: true})
	doc := s.Build()
	require.NotNil(t, doc.Components.SecuritySchemes["bearerAuth"])
	require.Equal(t, "bearer", doc.Components.SecuritySchemes["bearerAuth"].Scheme)
	sec := doc.Paths["/me"].Get.Security
	require.Len(t, sec, 1)
	_, ok := sec[0]["bearerAuth"]
	require.True(t, ok)
}

func TestMethodsMergeOnPath(t *testing.T) {
	s := New(Info{Title: "API", Version: "1.0.0"})
	s.Register("GET", "/x", Op{Summary: "get"})
	s.Register("DELETE", "/x", Op{Summary: "del"})
	item := s.Build().Paths["/x"]
	require.NotNil(t, item.Get)
	require.NotNil(t, item.Delete)
}
