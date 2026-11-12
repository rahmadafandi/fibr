// Copyright 2026 Rahmad Afandi. MIT License.

package mailer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySenderCaptures(t *testing.T) {
	var s MemorySender
	require.NoError(t, s.Send(context.Background(), Message{To: []string{"a@b.com"}, Subject: "Hi"}))
	require.NoError(t, s.Send(context.Background(), Message{To: []string{"c@d.com"}, Subject: "Yo"}))
	msgs := s.Messages()
	require.Len(t, msgs, 2)
	assert.Equal(t, "Hi", msgs[0].Subject)
	assert.Equal(t, "c@d.com", msgs[1].To[0])
}

func TestRender(t *testing.T) {
	html, text, err := Render("<p>Hi {{.Name}}</p>", "Hi {{.Name}}", struct{ Name string }{"Bob"})
	require.NoError(t, err)
	assert.Equal(t, "<p>Hi Bob</p>", html)
	assert.Equal(t, "Hi Bob", text)
}

func TestRenderBadTemplate(t *testing.T) {
	_, _, err := Render("{{.", "x", nil)
	assert.Error(t, err)
}

func TestNewEmptyHostIsLogSender(t *testing.T) {
	s, err := New(SMTPConfig{})
	require.NoError(t, err)
	_, ok := s.(*LogSender)
	assert.True(t, ok)
}

func TestNewWithHostIsSMTP(t *testing.T) {
	s, err := New(SMTPConfig{Host: "localhost", Port: 587})
	require.NoError(t, err)
	_, ok := s.(*SMTPSender)
	assert.True(t, ok)
}

func TestLogSenderSendNil(t *testing.T) {
	require.NoError(t, NewLogSender().Send(context.Background(), Message{To: []string{"a@b.com"}, Subject: "x"}))
}

// fakeSMTP is a minimal plaintext SMTP server that accepts one message and
// reports the DATA payload it received.
func fakeSMTP(t *testing.T) (host string, port int, data <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	got := make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		say("220 fake ESMTP")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case cmd == "DATA":
				say("354 go ahead")
				var body strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil || l == ".\r\n" {
						break
					}
					body.WriteString(l)
				}
				got <- body.String()
				say("250 queued")
			case cmd == "QUIT":
				say("221 bye")
				return
			default: // MAIL FROM, RCPT TO, RSET, NOOP
				say("250 ok")
			}
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, got
}

func TestSMTPSenderDeliversBothParts(t *testing.T) {
	host, port, data := fakeSMTP(t)
	s, err := NewSMTP(SMTPConfig{Host: host, Port: port, From: "noreply@example.com"})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, s.Send(ctx, Message{
		To: []string{"bob@example.com"}, Subject: "Welcome", Text: "plain body", HTML: "<p>html body</p>",
	}))

	select {
	case msg := <-data:
		assert.Contains(t, msg, "Subject: Welcome")
		assert.Contains(t, msg, "noreply@example.com")
		assert.Contains(t, msg, "bob@example.com")
		assert.Contains(t, msg, "plain body")
		assert.Contains(t, msg, "html body")
	case <-time.After(5 * time.Second):
		t.Fatal("server received no message")
	}
}

func TestSMTPSenderRejectsBadAddresses(t *testing.T) {
	s, err := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: 1, From: "not an address"})
	require.NoError(t, err)
	assert.ErrorContains(t, s.Send(context.Background(), Message{To: []string{"bob@example.com"}}), "mailer: from")

	s, err = NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: 1, From: "noreply@example.com", Username: "u", Password: "p"})
	require.NoError(t, err)
	assert.ErrorContains(t, s.Send(context.Background(), Message{To: []string{"nope"}}), "mailer: to")
}

func TestRenderExecErrors(t *testing.T) {
	_, _, err := Render("{{.Missing.Field}}", "x", struct{}{})
	assert.ErrorContains(t, err, "mailer: exec html")
	_, _, err = Render("ok", "{{.", nil)
	assert.ErrorContains(t, err, "mailer: parse text")
	_, _, err = Render("ok", "{{.Missing.Field}}", struct{}{})
	assert.ErrorContains(t, err, "mailer: exec text")
}
