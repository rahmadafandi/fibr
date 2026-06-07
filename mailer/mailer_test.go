// Copyright 2026 Rahmad Afandi. MIT License.

package mailer

import (
	"context"
	"testing"

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
