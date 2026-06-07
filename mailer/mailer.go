// Copyright 2026 Rahmad Afandi. MIT License.

// Package mailer sends transactional email through a pluggable Sender.
package mailer

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"sync"
	texttemplate "text/template"

	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/wneessen/go-mail"
)

// Message is a single email. It is JSON-serializable so it can double as a job
// payload.
type Message struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// Sender delivers a Message.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPConfig configures the SMTP sender.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPSender sends via go-mail over SMTP.
type SMTPSender struct {
	cfg    SMTPConfig
	client *mail.Client
}

// NewSMTP builds an SMTP sender. SMTP auth is used only when Username is set;
// TLS is opportunistic (works against a local mailhog as well as real servers).
func NewSMTP(cfg SMTPConfig) (*SMTPSender, error) {
	opts := []mail.Option{mail.WithPort(cfg.Port), mail.WithTLSPolicy(mail.TLSOpportunistic)}
	if cfg.Username != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.Username), mail.WithPassword(cfg.Password))
	}
	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mailer: smtp client: %w", err)
	}
	return &SMTPSender{cfg: cfg, client: client}, nil
}

// Send delivers msg over SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	m := mail.NewMsg()
	if err := m.From(s.cfg.From); err != nil {
		return fmt.Errorf("mailer: from: %w", err)
	}
	if err := m.To(msg.To...); err != nil {
		return fmt.Errorf("mailer: to: %w", err)
	}
	m.Subject(msg.Subject)
	switch {
	case msg.Text != "" && msg.HTML != "":
		m.SetBodyString(mail.TypeTextPlain, msg.Text)
		m.AddAlternativeString(mail.TypeTextHTML, msg.HTML)
	case msg.HTML != "":
		m.SetBodyString(mail.TypeTextHTML, msg.HTML)
	default:
		m.SetBodyString(mail.TypeTextPlain, msg.Text)
	}
	return s.client.DialAndSendWithContext(ctx, m)
}

// LogSender logs the recipient and subject instead of sending. Dev fallback.
type LogSender struct {
	log *logger.Logger
}

// NewLogSender returns a LogSender backed by the default logger.
func NewLogSender() *LogSender { return &LogSender{log: logger.Default()} }

// Send logs the message metadata and returns nil.
func (s *LogSender) Send(ctx context.Context, msg Message) error {
	s.log.Info("mailer: log transport (no SMTP configured)", "to", msg.To, "subject", msg.Subject)
	return nil
}

// MemorySender captures sent messages for tests.
type MemorySender struct {
	mu   sync.Mutex
	sent []Message
}

// Send records msg.
func (s *MemorySender) Send(ctx context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, msg)
	return nil
}

// Messages returns a copy of the captured messages.
func (s *MemorySender) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Message, len(s.sent))
	copy(out, s.sent)
	return out
}

// New returns an SMTPSender when cfg.Host is set, otherwise a LogSender
// (fail-soft dev default).
func New(cfg SMTPConfig) (Sender, error) {
	if cfg.Host == "" {
		return NewLogSender(), nil
	}
	return NewSMTP(cfg)
}

// Render executes the HTML and text templates with data and returns both
// rendered bodies.
func Render(htmlSrc, textSrc string, data any) (string, string, error) {
	var htmlBuf, textBuf bytes.Buffer
	ht, err := htmltemplate.New("html").Parse(htmlSrc)
	if err != nil {
		return "", "", fmt.Errorf("mailer: parse html: %w", err)
	}
	if err := ht.Execute(&htmlBuf, data); err != nil {
		return "", "", fmt.Errorf("mailer: exec html: %w", err)
	}
	tt, err := texttemplate.New("text").Parse(textSrc)
	if err != nil {
		return "", "", fmt.Errorf("mailer: parse text: %w", err)
	}
	if err := tt.Execute(&textBuf, data); err != nil {
		return "", "", fmt.Errorf("mailer: exec text: %w", err)
	}
	return htmlBuf.String(), textBuf.String(), nil
}
