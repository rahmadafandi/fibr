// Copyright 2026 Rahmad Afandi. MIT License.

package openapi_test

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/openapi"
)

type createUserReq struct {
	Email string `json:"email" validate:"required,email"`
}

type userResp struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func ExampleNew() {
	oapi := openapi.New(openapi.Info{Title: "Users API", Version: "1.0.0"}).WithBearerAuth()
	oapi.Register("POST", "/users", openapi.Op{
		Summary:  "Create user",
		Tags:     []string{"users"},
		Request:  createUserReq{},
		Response: userResp{},
		Status:   201,
		Secured:  true,
	})

	doc := oapi.Build()
	fmt.Println(doc.OpenAPI)
	fmt.Println(doc.Info.Title)
	// Output:
	// 3.0.3
	// Users API
}

func ExampleSpec_SpecHandler() {
	oapi := openapi.New(openapi.Info{Title: "API", Version: "1.0.0"})
	oapi.Register("GET", "/ping", openapi.Op{Summary: "Ping"})

	app := fiber.New()
	app.Get("/openapi.json", oapi.SpecHandler())
	app.Get("/docs", oapi.UIHandler("/openapi.json"))
	_ = app
	fmt.Println("docs mounted")
	// Output: docs mounted
}
