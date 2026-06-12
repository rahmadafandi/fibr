// Copyright 2026 Rahmad Afandi. MIT License.

package openapi

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// bytes marshals the document once and caches the result.
func (s *Spec) bytes() ([]byte, error) {
	s.once.Do(func() {
		s.cached, s.cerr = json.Marshal(s.Build())
	})
	return s.cached, s.cerr
}

// SpecHandler serves the OpenAPI document as JSON. The document is marshaled
// once on first request and cached, so register all operations before serving.
func (s *Spec) SpecHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		b, err := s.bytes()
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		return c.Send(b)
	}
}

// UIHandler serves a Swagger UI page (loaded from a CDN) pointed at specURL.
func (s *Spec) UIHandler(specURL string) fiber.Handler {
	html := swaggerHTML(specURL)
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.SendString(html)
	}
}

func swaggerHTML(specURL string) string {
	u, _ := json.Marshal(specURL) // JSON-encode to safely embed in the script
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>API Docs</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui-bundle.js" crossorigin></script>
<script>
window.onload = function () {
  window.ui = SwaggerUIBundle({ url: ` + string(u) + `, dom_id: "#swagger-ui" });
};
</script>
</body>
</html>`
}
