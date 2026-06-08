// Copyright 2026 Rahmad Afandi. MIT License.

// Command mailer demonstrates rendering an HTML/text email from templates and
// sending it. It uses the in-memory sender so it runs without an SMTP server —
// swap MemorySender for mailer.New(SMTPConfig{...}) in production. Run it with
// `go run ./mailer`.
package main

import (
	"context"
	"fmt"

	"github.com/rahmadafandi/fibr/mailer"
)

const (
	htmlTmpl = `<h1>Welcome, {{.Name}}!</h1><p>Thanks for joining {{.App}}.</p>`
	textTmpl = `Welcome, {{.Name}}! Thanks for joining {{.App}}.`
)

func main() {
	ctx := context.Background()

	// Render the body from templates plus per-recipient data.
	html, text, err := mailer.Render(htmlTmpl, textTmpl, map[string]string{
		"Name": "Ada",
		"App":  "fibr",
	})
	if err != nil {
		panic(err)
	}

	// MemorySender captures messages instead of delivering them — handy in tests
	// and demos. In production use mailer.New(mailer.SMTPConfig{...}).
	sender := &mailer.MemorySender{}
	msg := mailer.Message{
		To:      []string{"ada@example.com"},
		Subject: "Welcome aboard",
		HTML:    html,
		Text:    text,
	}
	if err := sender.Send(ctx, msg); err != nil {
		panic(err)
	}

	for _, m := range sender.Messages() {
		fmt.Println("sent to:", m.To)
		fmt.Println("subject:", m.Subject)
		fmt.Println("text:", m.Text)
	}
}
